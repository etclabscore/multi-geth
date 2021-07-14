// Copyright 2021 The multi-geth Authors
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
package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params/types/coregeth"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
)

var (
	// AstorChainConfig is the chain parameters to run a node on the Ethereum Classic Astor test network (Keccak mining).
	AstorChainConfig = &coregeth.CoreGethChainConfig{
		NetworkID: 212,
		ChainID:   big.NewInt(212),
		Keccak:    new(ctypes.KeccakConfig),

		EIP2FBlock: big.NewInt(0),
		EIP7FBlock: big.NewInt(0),

		EIP150Block: big.NewInt(0),

		EIP155Block: big.NewInt(0),

		// EIP158 eq
		EIP160FBlock: big.NewInt(0),
		EIP161FBlock: big.NewInt(0),
		EIP170FBlock: big.NewInt(0),

		// Byzantium eq
		EIP100FBlock: big.NewInt(0),
		EIP140FBlock: big.NewInt(0),
		EIP198FBlock: big.NewInt(0),
		EIP211FBlock: big.NewInt(0),
		EIP212FBlock: big.NewInt(0),
		EIP213FBlock: big.NewInt(0),
		EIP214FBlock: big.NewInt(0),
		EIP658FBlock: big.NewInt(0),

		// Constantinople eq, aka Agharta
		EIP145FBlock:  big.NewInt(0),
		EIP1014FBlock: big.NewInt(0),
		EIP1052FBlock: big.NewInt(0),

		// Istanbul eq, aka Phoenix
		// ECIP-1088
		EIP152FBlock:  big.NewInt(0),
		EIP1108FBlock: big.NewInt(0),
		EIP1344FBlock: big.NewInt(0),
		EIP1884FBlock: big.NewInt(0),
		EIP2028FBlock: big.NewInt(0),
		EIP2200FBlock: big.NewInt(0), // RePetersburg (== re-1283)

		ECIP1099FBlock: big.NewInt(0), // Etchash

		DisposalBlock:      big.NewInt(0),
		ECIP1017FBlock:     big.NewInt(0),
		ECIP1017EraRounds:  big.NewInt(0),
		ECIP1010PauseBlock: nil,
		ECIP1010Length:     nil,
		ECBP1100FBlock:     big.NewInt(0), // ETA 29 Sept 2020, ~1500 UTC
	}
)
