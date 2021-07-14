// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package Keccak implements the Keccak proof-of-work consensus engine.
package keccak

import (
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// two256 is a big integer representing 2^256
	two256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

// Config are the configuration parameters of the Keccak.
type Config struct {
	PowMode ethash.Mode

	Log log.Logger `toml:"-"`
	// ECIP-1099
	ECIP1099Block *uint64 `toml:"-"`
}

// Keccak is a consensus engine based on proof-of-work implementing the Keccak hashing algorithm.
type Keccak struct {
	config Config

	// Mining related fields
	rand     *rand.Rand    // Properly seeded random source for nonces
	threads  int           // Number of threads to mine on if mining
	update   chan struct{} // Notification channel to update mining parameters
	hashrate metrics.Meter // Meter tracking the average hashrate
	remote   *remoteSealer

	// The fields below are hooks for testing
	fakeFail  uint64        // Block number which fails PoW check even in fake mode
	fakeDelay time.Duration // Time delay to sleep for before returning from verify

	lock      sync.Mutex // Ensures thread safety for the in-memory caches and mining fields
	closeOnce sync.Once  // Ensures exit channel will not be closed twice.
}

// New creates a Keccak PoW scheme and starts a background thread for
// remote mining, also optionally notifying a batch of remote services of new work
// packages.
func New(config Config, notify []string, noverify bool) *Keccak {
	if config.Log == nil {
		config.Log = log.Root()
	}
	Keccak := &Keccak{
		config:   config,
		update:   make(chan struct{}),
		hashrate: metrics.NewMeterForced(),
	}
	Keccak.remote = startRemoteSealer(Keccak, notify, noverify)
	return Keccak
}

// NewTester creates a small sized keccak PoW scheme useful only for testing
// purposes.
func NewTester(notify []string, noverify bool) *Keccak {
	Keccak := &Keccak{
		config:   Config{PowMode: ethash.ModeTest, Log: log.Root()},
		update:   make(chan struct{}),
		hashrate: metrics.NewMeterForced(),
	}
	Keccak.remote = startRemoteSealer(Keccak, notify, noverify)
	return Keccak
}

// NewFaker creates a keccak consensus engine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the Ethereum
// consensus rules.
func NewFaker() *Keccak {
	return &Keccak{
		config: Config{
			PowMode: ethash.ModeFake,
			Log:     log.Root(),
		},
	}
}

// NewFakeFailer creates a keccak consensus engine with a fake PoW scheme that
// accepts all blocks as valid apart from the single one specified, though they
// still have to conform to the Ethereum consensus rules.
func NewFakeFailer(fail uint64) *Keccak {
	return &Keccak{
		config: Config{
			PowMode: ethash.ModeFake,
			Log:     log.Root(),
		},
		fakeFail: fail,
	}
}

// NewFakeDelayer creates a keccak consensus engine with a fake PoW scheme that
// accepts all blocks as valid, but delays verifications by some time, though
// they still have to conform to the Ethereum consensus rules.
func NewFakeDelayer(delay time.Duration) *Keccak {
	return &Keccak{
		config: Config{
			PowMode: ethash.ModeFake,
			Log:     log.Root(),
		},
		fakeDelay: delay,
	}
}

// NewPoissonFaker creates a Keccak consensus engine with a fake PoW scheme that
// accepts all blocks as valid, but delays mining by some time based on miner.threads, though
// they still have to conform to the Ethereum consensus rules.
func NewPoissonFaker() *Keccak {
	return &Keccak{
		config: Config{
			PowMode: ethash.ModePoissonFake,
			Log:     log.Root(),
		},
	}
}

// NewFullFaker creates a Keccak consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFullFaker() *Keccak {
	return &Keccak{
		config: Config{
			PowMode: ethash.ModeFullFake,
			Log:     log.Root(),
		},
	}
}

// Close closes the exit channel to notify all backend threads exiting.
func (Keccak *Keccak) Close() error {
	var err error
	Keccak.closeOnce.Do(func() {
		// Short circuit if the exit channel is not allocated.
		if Keccak.remote == nil {
			return
		}
		close(Keccak.remote.requestExit)
		<-Keccak.remote.exitCh
	})
	return err
}

// Threads returns the number of mining threads currently enabled. This doesn't
// necessarily mean that mining is running!
func (Keccak *Keccak) Threads() int {
	Keccak.lock.Lock()
	defer Keccak.lock.Unlock()

	return Keccak.threads
}

// SetThreads updates the number of mining threads currently enabled. Calling
// this method does not start mining, only sets the thread count. If zero is
// specified, the miner will use all cores of the machine. Setting a thread
// count below zero is allowed and will cause the miner to idle, without any
// work being done.
func (Keccak *Keccak) SetThreads(threads int) {
	Keccak.lock.Lock()
	defer Keccak.lock.Unlock()

	// Update the threads and ping any running seal to pull in any changes
	Keccak.threads = threads
	select {
	case Keccak.update <- struct{}{}:
	default:
	}
}

// Hashrate implements PoW, returning the measured rate of the search invocations
// per second over the last minute.
// Note the returned hashrate includes local hashrate, but also includes the total
// hashrate of all remote miner.
func (Keccak *Keccak) Hashrate() float64 {
	// Short circuit if we are run the Keccak in normal/test mode.
	if Keccak.config.PowMode != ethash.ModeNormal && Keccak.config.PowMode != ethash.ModeTest {
		return Keccak.hashrate.Rate1()
	}
	var res = make(chan uint64, 1)

	select {
	case Keccak.remote.fetchRateCh <- res:
	case <-Keccak.remote.exitCh:
		// Return local hashrate only if Keccak is stopped.
		return Keccak.hashrate.Rate1()
	}

	// Gather total submitted hash rate of remote sealers.
	return Keccak.hashrate.Rate1() + float64(<-res)
}

// APIs implements consensus.Engine, returning the user facing RPC APIs.
func (Keccak *Keccak) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &API{Keccak},
			Public:    true,
		},
	}
}

// SeedHash is the seed to use for generating a verification cache and the mining
// dataset.
func SeedHash(epoch uint64, epochLength uint64) []byte {
	return seedHash(epoch, epochLength)
}
