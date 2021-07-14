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

package keccak

import (
	"encoding/binary"
	"hash"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

const (
	epochLengthDefault  = 30000 // Default epoch length (blocks per epoch)
	epochLengthECIP1099 = 60000 // Blocks per epoch if ECIP-1099 is activated
)

// calcEpochLength returns the epoch length for a given block number (ECIP-1099)
func calcEpochLength(block uint64, ecip1099FBlock *uint64) uint64 {
	if ecip1099FBlock != nil {
		if block >= *ecip1099FBlock {
			return epochLengthECIP1099
		}
	}
	return epochLengthDefault
}

// calcEpoch returns the epoch for a given block number (ECIP-1099)
func calcEpoch(block uint64, epochLength uint64) uint64 {
	epoch := block / epochLength
	return epoch
}

// calcEpochBlock returns the epoch start block for a given epoch (ECIP-1099)
func calcEpochBlock(epoch uint64, epochLength uint64) uint64 {
	return epoch*epochLength + 1
}

// hasher is a repetitive hasher allowing the same hash data structures to be
// reused between hash runs instead of requiring new ones to be created.
type hasher func(dest []byte, data []byte)

// makeHasher creates a repetitive hasher, allowing the same hash data structures to
// be reused between hash runs instead of requiring new ones to be created. The returned
// function is not thread safe!
func makeHasher(h hash.Hash) hasher {
	// sha3.state supports Read to get the sum, use it to avoid the overhead of Sum.
	// Read alters the state but we reset the hash before every operation.
	type readerHash interface {
		hash.Hash
		Read([]byte) (int, error)
	}
	rh, ok := h.(readerHash)
	if !ok {
		panic("can't find Read method on hash")
	}
	outputLen := rh.Size()
	return func(dest []byte, data []byte) {
		rh.Reset()
		rh.Write(data)
		rh.Read(dest[:outputLen])
	}
}

// seedHash is the seed to use for generating a verification cache and the mining
// dataset. The block number passed should be pre-rounded to an epoch boundary + 1
// e.g: seedHash(calcEpochBlock(epoch, epochLength))
func seedHash(epoch uint64, epochLength uint64) []byte {
	block := calcEpochBlock(epoch, epochLength)

	seed := make([]byte, 32)
	if block < epochLengthDefault {
		return seed
	}

	keccak256 := makeHasher(sha3.NewLegacyKeccak256())
	for i := 0; i < int(block/epochLengthDefault); i++ {
		keccak256(seed, seed)
	}
	return seed
}

// keccakHasher produces a hash from combining the hash of the block header contents
// combined with a nonce.
func keccakHasher(hash []byte, nonce uint64) ([]byte, []byte) {

	// Combine hash+nonce into a 64 byte seed
	seed := make([]byte, 40)
	copy(seed, hash)
	binary.BigEndian.PutUint64(seed[32:], nonce)

	solution := crypto.Keccak256(seed)
	return solution, solution
}
