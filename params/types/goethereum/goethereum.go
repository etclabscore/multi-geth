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

package goethereum

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
)

type ChainConfig struct {
	NetworkID                 uint64   `json:"-"`
	ChainID                   *big.Int `json:"chainId"`                             // chainId identifies the current chain and is used for replay protection
	SupportedProtocolVersions []uint   `json:"supportedProtocolVersions,omitempty"` // supportedProtocolVersions identifies the supported eth protocol versions for the current chain

	// HF: Homestead
	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	// HF: DAO
	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// HF: Tangerine Whistle
	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	// HF: Spurious Dragon
	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block, includes implementations of 158/161, 160, and 170

	// HF: Byzantium
	ByzantiumBlock *big.Int `json:"byzantiumBlock,omitempty"` // Byzantium switch block (nil = no fork, 0 = already on byzantium)

	// HF: Constantinople
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)

	PetersburgBlock *big.Int `json:"petersburgBlock,omitempty"` // Petersburg switch block (nil = same as Constantinople)

	// HF: Istanbul
	IstanbulBlock    *big.Int `json:"istanbulBlock,omitempty"`    // Istanbul switch block (nil = no fork, 0 = already on istanbul)
	MuirGlacierBlock *big.Int `json:"muirGlacierBlock,omitempty"` // Eip-2384 (bomb delay) switch block (nil = no fork, 0 = already activated)

	BerlinBlock *big.Int `json:"berlinBlock,omitempty"` // Berlin switch block
	YoloV3Block *big.Int `json:"yoloV3Block,omitempty"` // YOLO v3 (Ephemeral testnet)

	EWASMBlock    *big.Int `json:"ewasmBlock,omitempty"`    // EWASM switch block (nil = no fork, 0 = already activated)
	CatalystBlock *big.Int `json:"catalystBlock,omitempty"` // Catalyst switch block (nil = no fork, 0 = already on catalyst)

	// Various consensus engines
	Ethash *ctypes.EthashConfig `json:"ethash,omitempty"`
	Clique *ctypes.CliqueConfig `json:"clique,omitempty"`
	Keccak *ctypes.KeccakConfig `json:"keccak256,omitempty"`
	Lyra2  *ctypes.Lyra2Config  `json:"lyra2,omitempty"`

	// NOTE: These are not included in this type upstream.
	TrustedCheckpoint       *ctypes.TrustedCheckpoint      `json:"trustedCheckpoint"`
	TrustedCheckpointOracle *ctypes.CheckpointOracleConfig `json:"trustedCheckpointOracle"`

	EIP1706Transition  *big.Int `json:"-"`
	ECIP1080Transition *big.Int `json:"-"`

	// Cache types for use with testing, but will not show up in config API.
	ecbp1100Transition *big.Int
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.Clique != nil:
		engine = c.Clique
	case c.Keccak != nil:
		engine = c.Keccak
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v DAO: %v DAOSupport: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Constantinople: %v ConstantinopleFix: %v Istanbul: %v MuirGlacier: %v Berlin: %v Engine: %v}",
		c.ChainID,
		c.HomesteadBlock,
		c.DAOForkBlock,
		c.DAOForkSupport,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		c.ConstantinopleBlock,
		c.PetersburgBlock,
		c.IstanbulBlock,
		c.MuirGlacierBlock,
		c.BerlinBlock,
		engine,
	)
}
