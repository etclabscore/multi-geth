// Copyright 2019 The multi-geth Authors
// This file is part of the multi-geth library.
//
// The multi-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The multi-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the multi-geth library. If not, see <http://www.gnu.org/licenses/>.

package ctypes

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// This file holds the Configurator interfaces.
// Interface methods follow distinct naming and signature patterns
// to enable abstracted logic.
//
// All methods are in pairs; Get'ers and Set'ers.
// Some Set methods are prefixed with Must, ie MustSet. These methods
// are allowed to return errors for debugging and logging, but
// any non-nil errors returned should stop program execution.
//
// All Forking methods (getting and setting Hard-fork requiring protocol changes)
// are suffixed with "Transition", and use *uint64 as in- and out-put variable types.
// A pointer is used because it's important that the fields can be nil, to signal
// being unset (as opposed to zero value uint64 == 0, ie from Genesis).

type Configurator interface {
	ChainConfigurator
	GenesisBlocker
}

type ChainConfigurator interface {
	ProtocolSpecifier
	Forker
	ConsensusEnginator // Consensus Engine
	// CHTer
}

// ProtocolSpecifier defines protocol interfaces that are agnostic of consensus engine.
type ProtocolSpecifier interface {
	GetAccountStartNonce() *uint64
	SetAccountStartNonce(n *uint64) error
	GetMaximumExtraDataSize() *uint64
	SetMaximumExtraDataSize(n *uint64) error
	GetMinGasLimit() *uint64
	SetMinGasLimit(n *uint64) error
	GetGasLimitBoundDivisor() *uint64
	SetGasLimitBoundDivisor(n *uint64) error
	GetNetworkID() *uint64
	SetNetworkID(n *uint64) error
	GetChainID() *big.Int
	SetChainID(i *big.Int) error
	GetMaxCodeSize() *uint64
	SetMaxCodeSize(n *uint64) error

	// Be careful with EIP2.
	// It is a messy EIP, specifying diverse changes, like difficulty, intrinsic gas costs for contract creation,
	// txpool management, and contract OoG handling.
	// It is both Ethash-specific and _not_.
	GetEIP2Transition() *uint64
	SetEIP2Transition(n *uint64) error

	GetEIP7Transition() *uint64
	SetEIP7Transition(n *uint64) error
	GetEIP150Transition() *uint64
	SetEIP150Transition(n *uint64) error
	GetEIP152Transition() *uint64
	SetEIP152Transition(n *uint64) error
	GetEIP160Transition() *uint64
	SetEIP160Transition(n *uint64) error
	GetEIP161abcTransition() *uint64
	SetEIP161abcTransition(n *uint64) error
	GetEIP161dTransition() *uint64
	SetEIP161dTransition(n *uint64) error
	GetEIP170Transition() *uint64
	SetEIP170Transition(n *uint64) error
	GetEIP155Transition() *uint64
	SetEIP155Transition(n *uint64) error
	GetEIP140Transition() *uint64
	SetEIP140Transition(n *uint64) error
	GetEIP198Transition() *uint64
	SetEIP198Transition(n *uint64) error
	GetEIP211Transition() *uint64
	SetEIP211Transition(n *uint64) error
	GetEIP212Transition() *uint64
	SetEIP212Transition(n *uint64) error
	GetEIP213Transition() *uint64
	SetEIP213Transition(n *uint64) error
	GetEIP214Transition() *uint64
	SetEIP214Transition(n *uint64) error
	GetEIP658Transition() *uint64
	SetEIP658Transition(n *uint64) error
	GetEIP145Transition() *uint64
	SetEIP145Transition(n *uint64) error
	GetEIP1014Transition() *uint64
	SetEIP1014Transition(n *uint64) error
	GetEIP1052Transition() *uint64
	SetEIP1052Transition(n *uint64) error
	GetEIP1283Transition() *uint64
	SetEIP1283Transition(n *uint64) error
	GetEIP1283DisableTransition() *uint64
	SetEIP1283DisableTransition(n *uint64) error
	GetEIP1108Transition() *uint64
	SetEIP1108Transition(n *uint64) error
	GetEIP2200Transition() *uint64
	SetEIP2200Transition(n *uint64) error
	GetEIP2200DisableTransition() *uint64
	SetEIP2200DisableTransition(n *uint64) error
	GetEIP1344Transition() *uint64
	SetEIP1344Transition(n *uint64) error
	GetEIP1884Transition() *uint64
	SetEIP1884Transition(n *uint64) error
	GetEIP2028Transition() *uint64
	SetEIP2028Transition(n *uint64) error
	GetECIP1080Transition() *uint64
	SetECIP1080Transition(n *uint64) error
	GetEIP1706Transition() *uint64
	SetEIP1706Transition(n *uint64) error
	GetEIP2537Transition() *uint64
	SetEIP2537Transition(n *uint64) error
	GetECBP1100Transition() *uint64
	SetECBP1100Transition(n *uint64) error
	GetEIP2315Transition() *uint64
	SetEIP2315Transition(n *uint64) error
	GetEIP2929Transition() *uint64
	SetEIP2929Transition(n *uint64) error
}

type Forker interface {
	// IsEnabled tells if interface has met or exceeded a fork block number.
	// eg. IsEnabled(c.GetEIP1108Transition, big.NewInt(42)))
	IsEnabled(fn func() *uint64, n *big.Int) bool

	// ForkCanonHash yields arbitrary number/hash pairs.
	// This is an abstraction derived from the original EIP150 implementation.
	GetForkCanonHash(n uint64) common.Hash
	SetForkCanonHash(n uint64, h common.Hash) error
	GetForkCanonHashes() map[uint64]common.Hash
}

type ConsensusEnginator interface {
	GetConsensusEngineType() ConsensusEngineT
	MustSetConsensusEngineType(t ConsensusEngineT) error
	EthashConfigurator
	CliqueConfigurator
}

type EthashConfigurator interface {
	GetEthashMinimumDifficulty() *big.Int
	SetEthashMinimumDifficulty(i *big.Int) error
	GetEthashDifficultyBoundDivisor() *big.Int
	SetEthashDifficultyBoundDivisor(i *big.Int) error
	GetEthashDurationLimit() *big.Int
	SetEthashDurationLimit(i *big.Int) error
	GetEthashHomesteadTransition() *uint64
	SetEthashHomesteadTransition(n *uint64) error

	// GetEthashEIP779Transition should return the block if the node wants the fork.
	// Otherwise, nil should be returned.
	GetEthashEIP779Transition() *uint64 // DAO

	// SetEthashEIP779Transition should turn DAO support on (nonnil) or off (nil).
	SetEthashEIP779Transition(n *uint64) error
	GetEthashEIP649Transition() *uint64
	SetEthashEIP649Transition(n *uint64) error
	GetEthashEIP1234Transition() *uint64
	SetEthashEIP1234Transition(n *uint64) error
	GetEthashEIP2384Transition() *uint64
	SetEthashEIP2384Transition(n *uint64) error
	GetEthashECIP1010PauseTransition() *uint64
	SetEthashECIP1010PauseTransition(n *uint64) error
	GetEthashECIP1010ContinueTransition() *uint64
	SetEthashECIP1010ContinueTransition(n *uint64) error
	GetEthashECIP1017Transition() *uint64
	SetEthashECIP1017Transition(n *uint64) error
	GetEthashECIP1017EraRounds() *uint64
	SetEthashECIP1017EraRounds(n *uint64) error
	GetEthashEIP100BTransition() *uint64
	SetEthashEIP100BTransition(n *uint64) error
	GetEthashECIP1041Transition() *uint64
	SetEthashECIP1041Transition(n *uint64) error
	GetEthashECIP1099Transition() *uint64
	SetEthashECIP1099Transition(n *uint64) error

	GetEthashDifficultyBombDelaySchedule() Uint64BigMapEncodesHex
	SetEthashDifficultyBombDelaySchedule(m Uint64BigMapEncodesHex) error
	GetEthashBlockRewardSchedule() Uint64BigMapEncodesHex
	SetEthashBlockRewardSchedule(m Uint64BigMapEncodesHex) error
}

type CliqueConfigurator interface {
	GetCliquePeriod() uint64
	SetCliquePeriod(n uint64) error
	GetCliqueEpoch() uint64
	SetCliqueEpoch(n uint64) error

	/*
		Clique protocol rules to avoid deadlocks caused by ambiguity in the original consensus protocol definition.
		https://github.com/ethereum/EIPs/blob/master/EIPS/eip-3436.md
	*/
	GetCliqueEIP3436Transition() *uint64
	SetCliqueEIP3436Transition(n *uint64) error
}

type BlockSealer interface {
	GetSealingType() BlockSealingT
	SetSealingType(t BlockSealingT) error
	BlockSealerEthereum
}

type BlockSealerEthereum interface {
	GetGenesisSealerEthereumNonce() uint64
	SetGenesisSealerEthereumNonce(n uint64) error
	GetGenesisSealerEthereumMixHash() common.Hash
	SetGenesisSealerEthereumMixHash(h common.Hash) error
}

type GenesisBlocker interface {
	BlockSealer
	Accounter
	GetGenesisDifficulty() *big.Int
	SetGenesisDifficulty(i *big.Int) error
	GetGenesisAuthor() common.Address
	SetGenesisAuthor(a common.Address) error
	GetGenesisTimestamp() uint64
	SetGenesisTimestamp(u uint64) error
	GetGenesisParentHash() common.Hash
	SetGenesisParentHash(h common.Hash) error
	GetGenesisExtraData() []byte
	SetGenesisExtraData(b []byte) error
	GetGenesisGasLimit() uint64
	SetGenesisGasLimit(u uint64) error
}

type Accounter interface {
	ForEachAccount(fn func(address common.Address, bal *big.Int, nonce uint64, code []byte, storage map[common.Hash]common.Hash) error) error
	UpdateAccount(address common.Address, bal *big.Int, nonce uint64, code []byte, storage map[common.Hash]common.Hash) error
}
