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

package params

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params/types/genesisT"
)

var AstorGenesisHash = common.HexToHash("0xf9c534ad514bc380e23a74e788c3d1f2446f9ad1cb74f0dbb48ea56c0315f5bc")

func DefaultAstorGenesisBlock() *genesisT.Genesis {
	return &genesisT.Genesis{
		Config:     AstorChainConfig,
		Nonce:      hexutil.MustDecodeUint64("0x42"),
		ExtraData:  hexutil.MustDecode("0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa"),
		GasLimit:   hexutil.MustDecodeUint64("0x1fffffffffffff"),
		Difficulty: hexutil.MustDecodeBig("0x10000000000"),
		Timestamp:  hexutil.MustDecodeUint64("0x0"),
		Coinbase:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Mixhash:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		GasUsed:    hexutil.MustDecodeUint64("0x0"),
		Number:     uint64(0),
		ParentHash: common.HexToHash(""),
		Alloc: genesisT.GenesisAlloc{
			common.HexToAddress("0x6e3da3dd22f043958fdb862db5876e5e52a3d7b7"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0x46652437dc2f978952c5f5667533874c294e4e84"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0x3d650ae1134709a9f2f1c7785cd99400482cdd97"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0xe6b2d417af0e3686982dba572fcda575a4e8f575"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0xd118ab56776b158c70a0d03ed5ff03392a63deaf"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0x2bac997f3c4a51e655d14733aa1d75817443779a"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0xbd74228dca706ab4ab7d8b856ec6700624420831"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0x33bc5fb03008d7f0001638f48bfff2d628d16882"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0xef826b2e0f6bf62edfe628dbeaa1af58f40f3b06"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0x7defc96deecc05288ff177ca08c025ed5c139a95"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0x2958DB51a0b4c458d0aa183E8cFB4f2E95cf6E75"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0xB460ce2C10d959251792D72b7E2B3EC684F013f1"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
			common.HexToAddress("0x25848A733da7c782fB963D2e562cF7246bD5b6df"): genesisT.GenesisAccount{Balance: hexutil.MustDecodeBig("0x130EE8E7179044400000")},
		},
	}
}
