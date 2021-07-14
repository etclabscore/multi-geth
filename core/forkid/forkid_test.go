// Copyright 2019 The go-ethereum Authors
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

package forkid

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/confp"
	"github.com/ethereum/go-ethereum/params/types/coregeth"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestCreation tests that different genesis and fork rule combinations result in
// the correct fork ID.
func TestCreation(t *testing.T) {
	type testcase struct {
		head uint64
		want ID
	}
	tests := []struct {
		name    string
		config  ctypes.ChainConfigurator
		genesis common.Hash
		cases   []testcase
	}{
		// Mainnet test cases
		{
			"mainnet",
			params.MainnetChainConfig,
			params.MainnetGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},         // Unsynced
				{1149999, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},   // Last Frontier block
				{1150000, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},   // First Homestead block
				{1919999, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},   // Last Homestead block
				{1920000, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},   // First DAO block
				{2462999, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},   // Last DAO block
				{2463000, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},   // First Tangerine block
				{2674999, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},   // Last Tangerine block
				{2675000, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},   // First Spurious block
				{4369999, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},   // Last Spurious block
				{4370000, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},   // First Byzantium block
				{7279999, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},   // Last Byzantium block
				{7280000, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},   // First and last Constantinople, first Petersburg block
				{9068999, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},   // Last Petersburg block
				{9069000, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},   // First Istanbul and first Muir Glacier block
				{9199999, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},   // Last Istanbul and first Muir Glacier block
				{9200000, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},  // First Muir Glacier block
				{12243999, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}}, // Last Muir Glacier block
				{12244000, ID{Hash: checksumToBytes(0x0eb440f6), Next: 0}},        // First Berlin block
				{20000000, ID{Hash: checksumToBytes(0x0eb440f6), Next: 0}},        // Future Berlin block
			},
		},
		// Ropsten test cases
		{
			"ropsten",
			params.RopstenChainConfig,
			params.RopstenGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0x30c7ddbc), Next: 10}},            // Unsynced, last Frontier, Homestead and first Tangerine block
				{9, ID{Hash: checksumToBytes(0x30c7ddbc), Next: 10}},            // Last Tangerine block
				{10, ID{Hash: checksumToBytes(0x63760190), Next: 1700000}},      // First Spurious block
				{1699999, ID{Hash: checksumToBytes(0x63760190), Next: 1700000}}, // Last Spurious block
				{1700000, ID{Hash: checksumToBytes(0x3ea159c7), Next: 4230000}}, // First Byzantium block
				{4229999, ID{Hash: checksumToBytes(0x3ea159c7), Next: 4230000}}, // Last Byzantium block
				{4230000, ID{Hash: checksumToBytes(0x97b544f3), Next: 4939394}}, // First Constantinople block
				{4939393, ID{Hash: checksumToBytes(0x97b544f3), Next: 4939394}}, // Last Constantinople block
				{4939394, ID{Hash: checksumToBytes(0xd6e2149b), Next: 6485846}}, // First Petersburg block
				{6485845, ID{Hash: checksumToBytes(0xd6e2149b), Next: 6485846}}, // Last Petersburg block
				{6485846, ID{Hash: checksumToBytes(0x4bc66396), Next: 7117117}}, // First Istanbul block
				{7117116, ID{Hash: checksumToBytes(0x4bc66396), Next: 7117117}}, // Last Istanbul block
				{7117117, ID{Hash: checksumToBytes(0x6727ef90), Next: 9812189}}, // First Muir Glacier block
				{9812188, ID{Hash: checksumToBytes(0x6727ef90), Next: 9812189}}, // Last Muir Glacier block
				{9812189, ID{Hash: checksumToBytes(0xa157d377), Next: 0}},       // First Berlin block
				{10000000, ID{Hash: checksumToBytes(0xa157d377), Next: 0}},      // Future Berlin block
			},
		},
		// Rinkeby test cases
		{
			"rinkeby",
			params.RinkebyChainConfig,
			params.RinkebyGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0x3b8e0691), Next: 1}},             // Unsynced, last Frontier block
				{1, ID{Hash: checksumToBytes(0x60949295), Next: 2}},             // First and last Homestead block
				{2, ID{Hash: checksumToBytes(0x8bde40dd), Next: 3}},             // First and last Tangerine block
				{3, ID{Hash: checksumToBytes(0xcb3a64bb), Next: 1035301}},       // First Spurious block
				{1035300, ID{Hash: checksumToBytes(0xcb3a64bb), Next: 1035301}}, // Last Spurious block
				{1035301, ID{Hash: checksumToBytes(0x8d748b57), Next: 3660663}}, // First Byzantium block
				{3660662, ID{Hash: checksumToBytes(0x8d748b57), Next: 3660663}}, // Last Byzantium block
				{3660663, ID{Hash: checksumToBytes(0xe49cab14), Next: 4321234}}, // First Constantinople block
				{4321233, ID{Hash: checksumToBytes(0xe49cab14), Next: 4321234}}, // Last Constantinople block
				{4321234, ID{Hash: checksumToBytes(0xafec6b27), Next: 5435345}}, // First Petersburg block
				{5435344, ID{Hash: checksumToBytes(0xafec6b27), Next: 5435345}}, // Last Petersburg block
				{5435345, ID{Hash: checksumToBytes(0xcbdb8838), Next: 8290928}}, // First Istanbul block
				{8290927, ID{Hash: checksumToBytes(0xcbdb8838), Next: 8290928}}, // Last Istanbul block
				{8290928, ID{Hash: checksumToBytes(0x6910c8bd), Next: 0}},       // First Berlin block
				{10000000, ID{Hash: checksumToBytes(0x6910c8bd), Next: 0}},      // Future Berlin block
			},
		},
		// Goerli test cases
		{
			"goerli",
			params.GoerliChainConfig,
			params.GoerliGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0xa3f5ab08), Next: 1561651}},       // Unsynced, last Frontier, Homestead, Tangerine, Spurious, Byzantium, Constantinople and first Petersburg block
				{1561650, ID{Hash: checksumToBytes(0xa3f5ab08), Next: 1561651}}, // Last Petersburg block
				{1561651, ID{Hash: checksumToBytes(0xc25efa5c), Next: 4460644}}, // First Istanbul block
				{4460643, ID{Hash: checksumToBytes(0xc25efa5c), Next: 4460644}}, // Last Istanbul block
				{4460644, ID{Hash: checksumToBytes(0x757a1c47), Next: 0}},       // First Berlin block
				{5000000, ID{Hash: checksumToBytes(0x757a1c47), Next: 0}},       // Future Berlin block
			},
		},
		{
			"classic",
			params.ClassicChainConfig,
			params.MainnetGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{1, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{2, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{3, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{9, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{10, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{1149999, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{1150000, ID{Hash: checksumToBytes(0x97c2c34c), Next: 2500000}},
				{1150001, ID{Hash: checksumToBytes(0x97c2c34c), Next: 2500000}},
				{2499999, ID{Hash: checksumToBytes(0x97c2c34c), Next: 2500000}},
				{2500000, ID{Hash: checksumToBytes(0xdb06803f), Next: 3000000}},
				{2500001, ID{Hash: checksumToBytes(0xdb06803f), Next: 3000000}},
				{2999999, ID{Hash: checksumToBytes(0xdb06803f), Next: 3000000}},
				{3000000, ID{Hash: checksumToBytes(0xaff4bed4), Next: 5000000}},
				{3000001, ID{Hash: checksumToBytes(0xaff4bed4), Next: 5000000}},
				{4999999, ID{Hash: checksumToBytes(0xaff4bed4), Next: 5000000}},
				{5000000, ID{Hash: checksumToBytes(0xf79a63c0), Next: 5900000}},
				{5000001, ID{Hash: checksumToBytes(0xf79a63c0), Next: 5900000}},
				{5899999, ID{Hash: checksumToBytes(0xf79a63c0), Next: 5900000}},
				{5900000, ID{Hash: checksumToBytes(0x744899d6), Next: 8772000}},
				{5900001, ID{Hash: checksumToBytes(0x744899d6), Next: 8772000}},
				{8771999, ID{Hash: checksumToBytes(0x744899d6), Next: 8772000}},
				{8772000, ID{Hash: checksumToBytes(0x518b59c6), Next: 9573000}},
				{8772001, ID{Hash: checksumToBytes(0x518b59c6), Next: 9573000}},
				{9572999, ID{Hash: checksumToBytes(0x518b59c6), Next: 9573000}},
				{9573000, ID{Hash: checksumToBytes(0x7ba22882), Next: 10500839}},
				{9573001, ID{Hash: checksumToBytes(0x7ba22882), Next: 10500839}},
				{10500838, ID{Hash: checksumToBytes(0x7ba22882), Next: 10500839}},
				{10500839, ID{Hash: checksumToBytes(0x9007bfcc), Next: 11_700_000}},
				{10500840, ID{Hash: checksumToBytes(0x9007bfcc), Next: 11_700_000}},
				{11_699_999, ID{Hash: checksumToBytes(0x9007bfcc), Next: 11_700_000}},
				{11_700_000, ID{Hash: checksumToBytes(0xdb63a1ca), Next: 13_189_133}},
				{11_700_001, ID{Hash: checksumToBytes(0xdb63a1ca), Next: 13_189_133}},
				{13_189_132, ID{Hash: checksumToBytes(0xdb63a1ca), Next: 13_189_133}},
				{13_189_133, ID{Hash: checksumToBytes(0x0f6bf187), Next: 0}},
				{13_189_134, ID{Hash: checksumToBytes(0x0f6bf187), Next: 0}},
			},
		},
		{
			"kotti",
			params.KottiChainConfig,
			params.KottiGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0x0550152e), Next: 716617}},
				{716616, ID{Hash: checksumToBytes(0x0550152e), Next: 716617}},
				{716617, ID{Hash: checksumToBytes(0xa3270822), Next: 1705549}},
				{716618, ID{Hash: checksumToBytes(0xa3270822), Next: 1705549}},
				{1705548, ID{Hash: checksumToBytes(0xa3270822), Next: 1705549}},
				{1705549, ID{Hash: checksumToBytes(0x8f3698e0), Next: 2200013}},
				{1705550, ID{Hash: checksumToBytes(0x8f3698e0), Next: 2200013}},
				{2200012, ID{Hash: checksumToBytes(0x8f3698e0), Next: 2200013}},
				{2200013, ID{Hash: checksumToBytes(0x6f402821), Next: 4_368_634}},
				{2200014, ID{Hash: checksumToBytes(0x6f402821), Next: 4_368_634}},
				{4_368_633, ID{Hash: checksumToBytes(0x6f402821), Next: 4_368_634}},
				{4_368_634, ID{Hash: checksumToBytes(0xf03e54e7), Next: 0}},
				{4_368_635, ID{Hash: checksumToBytes(0xf03e54e7), Next: 0}},
			},
		},
		{
			"mordor",
			params.MordorChainConfig,
			params.MordorGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0x175782aa), Next: 301243}},
				{1, ID{Hash: checksumToBytes(0x175782aa), Next: 301243}},
				{2, ID{Hash: checksumToBytes(0x175782aa), Next: 301243}},
				{3, ID{Hash: checksumToBytes(0x175782aa), Next: 301243}},
				{9, ID{Hash: checksumToBytes(0x175782aa), Next: 301243}},
				{10, ID{Hash: checksumToBytes(0x175782aa), Next: 301243}},
				{301242, ID{Hash: checksumToBytes(0x175782aa), Next: 301243}},
				{301243, ID{Hash: checksumToBytes(0x604f6ee1), Next: 999983}},
				{301244, ID{Hash: checksumToBytes(0x604f6ee1), Next: 999983}},
				{999982, ID{Hash: checksumToBytes(0x604f6ee1), Next: 999983}},
				{999983, ID{Hash: checksumToBytes(0xf42f5539), Next: 2_520_000}},
				{999984, ID{Hash: checksumToBytes(0xf42f5539), Next: 2_520_000}},
				{2_519_999, ID{Hash: checksumToBytes(0xf42f5539), Next: 2_520_000}},
				{2_520_000, ID{Hash: checksumToBytes(0x66b5c286), Next: 3_985_893}},
				{3_985_892, ID{Hash: checksumToBytes(0x66b5c286), Next: 3_985_893}},
				{3_985_893, ID{Hash: checksumToBytes(0x92b323e0), Next: 0}},
				{3_985_894, ID{Hash: checksumToBytes(0x92b323e0), Next: 0}},
			},
		},
		// MintMe test cases
		{
			"mintme",
			params.MintMeChainConfig,
			params.MintMeGenesisHash,
			[]testcase{
				{0, ID{Hash: checksumToBytes(0x02bf4180), Next: 0}},
			},
		},
	}
	for i, tt := range tests {
		for j, ttt := range tt.cases {
			if have := NewID(tt.config, tt.genesis, ttt.head); have != ttt.want {
				t.Errorf("chain: %s, test %d, case %d: fork ID mismatch: have %x, want %x", tt.name, i, j, have, ttt.want)
			}
		}
	}
}

// TestValidation tests that a local peer correctly validates and accepts a remote
// fork ID.
func TestValidation(t *testing.T) {
	tests := []struct {
		head uint64
		id   ID
		err  error
	}{
		// Local is mainnet Petersburg, remote announces the same. No future fork is announced.
		{7987396, ID{Hash: checksumToBytes(0x668db0af), Next: 0}, nil},

		// Local is mainnet Petersburg, remote announces the same. Remote also announces a next fork
		// at block 0xffffffff, but that is uncertain.
		{7987396, ID{Hash: checksumToBytes(0x668db0af), Next: math.MaxUint64}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, but it's not yet aware of Petersburg (e.g. non updated node before the fork).
		// In this case we don't know if Petersburg passed yet or not.
		{7279999, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of Petersburg (e.g. updated node before the fork). We
		// don't know if Petersburg passed yet (will pass) or not.
		{7279999, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of some random fork (e.g. misconfigured Petersburg). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		{7279999, ID{Hash: checksumToBytes(0xa00bc324), Next: math.MaxUint64}, nil},

		// Local is mainnet Petersburg, remote announces Byzantium + knowledge about Petersburg. Remote
		// is simply out of sync, accept.
		{7987396, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet Petersburg, remote announces Spurious + knowledge about Byzantium. Remote
		// is definitely out of sync. It may or may not need the Petersburg update, we don't know yet.
		{7987396, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}, nil},

		// Local is mainnet Byzantium, remote announces Petersburg. Local is out of sync, accept.
		{7279999, ID{Hash: checksumToBytes(0x668db0af), Next: 0}, nil},

		// Local is mainnet Spurious, remote announces Byzantium, but is not aware of Petersburg. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		{4369999, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},

		// Local is mainnet Petersburg. remote announces Byzantium but is not aware of further forks.
		// Remote needs software update.
		{7987396, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, ErrRemoteStale},

		// Local is mainnet Petersburg, and isn't aware of more forks. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{7987396, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Byzantium, and is aware of Petersburg. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{7279999, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Petersburg, remote is Rinkeby Petersburg.
		{7987396, ID{Hash: checksumToBytes(0xafec6b27), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Berlin, far in the future. Remote announces Gopherium (non existing fork)
		// at some future block 88888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		{88888888, ID{Hash: checksumToBytes(0x0eb440f6), Next: 88888888}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Byzantium. Remote is also in Byzantium, but announces Gopherium (non existing
		// fork) at block 7279999, before Petersburg. Local is incompatible.
		{7279999, ID{Hash: checksumToBytes(0xa00bc324), Next: 7279999}, ErrLocalIncompatibleOrStale},
	}
	for i, tt := range tests {
		filter := newFilter(params.MainnetChainConfig, params.MainnetGenesisHash, func() uint64 { return tt.head })
		if err := filter(tt.id); err != tt.err {
			t.Errorf("test %d, head: %d: validation error mismatch: have %v, want %v", i, tt.head, err, tt.err)
		}
	}
}

// Tests that IDs are properly RLP encoded (specifically important because we
// use uint32 to store the hash, but we need to encode it as [4]byte).
func TestEncoding(t *testing.T) {
	tests := []struct {
		id   ID
		want []byte
	}{
		{ID{Hash: checksumToBytes(0), Next: 0}, common.Hex2Bytes("c6840000000080")},
		{ID{Hash: checksumToBytes(0xdeadbeef), Next: 0xBADDCAFE}, common.Hex2Bytes("ca84deadbeef84baddcafe,")},
		{ID{Hash: checksumToBytes(math.MaxUint32), Next: math.MaxUint64}, common.Hex2Bytes("ce84ffffffff88ffffffffffffffff")},
	}
	for i, tt := range tests {
		have, err := rlp.EncodeToBytes(tt.id)
		if err != nil {
			t.Errorf("test %d: failed to encode forkid: %v", i, err)
			continue
		}
		if !bytes.Equal(have, tt.want) {
			t.Errorf("test %d: RLP mismatch: have %x, want %x", i, have, tt.want)
		}
	}
}

func TestGatherForks(t *testing.T) {
	cases := []struct {
		name   string
		config ctypes.ChainConfigurator
		wantNs []uint64
	}{
		{
			"classic",
			params.ClassicChainConfig,
			[]uint64{1150000, 2500000, 3000000, 5000000, 5900000, 8772000, 9573000, 10500839, 11_700_000, 13_189_133},
		},
		{
			"mainnet",
			params.MainnetChainConfig,
			[]uint64{1150000, 1920000, 2463000, 2675000, 4370000, 7280000, 9069000, 9200000, 12_244_000},
		},
		{
			"mordor",
			params.MordorChainConfig,
			[]uint64{301_243, 999_983, 2_520_000, 3_985_893},
		},
		{
			"kotti",
			params.KottiChainConfig,
			[]uint64{716_617, 1_705_549, 2_200_013, 4_368_634},
		},
		{
			"mintme",
			params.MintMeChainConfig,
			[]uint64{},
		},
	}
	sliceContains := func(sl []uint64, u uint64) bool {
		for _, s := range sl {
			if s == u {
				return true
			}
		}
		return false
	}
	for _, c := range cases {
		gotForkNs := gatherForks(c.config)
		if len(gotForkNs) != len(c.wantNs) {
			for _, n := range c.wantNs {
				if !sliceContains(gotForkNs, n) {
					t.Errorf("config=%s missing wanted fork at block number: %d", c.name, n)
				}
			}
			for _, n := range gotForkNs {
				if !sliceContains(c.wantNs, n) {
					t.Errorf("config=%s gathered unwanted fork at block number: %d", c.name, n)
				}
			}
		}
	}
}

// TestGenerateSpecificationCases generates markdown formatted specification
// for network forkid values.
func TestGenerateSpecificationCases(t *testing.T) {
	if os.Getenv("COREGETH_GENERATE_FORKID_TEST_CASES") == "" {
		t.Skip()
	}
	type testCaseJSON struct {
		ChainConfig *coregeth.CoreGethChainConfig `json:"geth_chain_config"`
		GenesisHash common.Hash                   `json:"genesis_hash"`
		Head        uint64                        `json:"head"`
		ForkHash    common.Hash                   `json:"fork_hash"`
		ForkNext    uint64                        `json:"fork_next"`
		ForkIDRLP   common.Hash                   `json:"fork_id_rlp"`
	}

	generatedCases := []*testCaseJSON{}

	tests := []struct {
		name        string
		config      ctypes.ChainConfigurator
		genesisHash common.Hash
	}{
		{"Ethereum Classic Mainnet (ETC)",
			params.ClassicChainConfig,
			params.MainnetGenesisHash,
		},
		{
			"Kotti",
			params.KottiChainConfig,
			params.KottiGenesisHash,
		},
		{
			"Mordor",
			params.MordorChainConfig,
			params.MordorGenesisHash,
		},
		{
			"Astor",
			params.AstorChainConfig,
			params.AstorGenesisHash,
		},
		{
			"Morden",
			&coregeth.CoreGethChainConfig{
				Ethash:            &ctypes.EthashConfig{},
				EIP2FBlock:        big.NewInt(494000),
				EIP150Block:       big.NewInt(1783000),
				EIP155Block:       big.NewInt(1915000),
				ECIP1017FBlock:    big.NewInt(2000000),
				ECIP1017EraRounds: big.NewInt(2000000),
				DisposalBlock:     big.NewInt(2300000),
				EIP198FBlock:      big.NewInt(4729274), // Atlantis
				EIP1052FBlock:     big.NewInt(5000381), // Agharta
			},
			common.HexToHash("0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"),
		},
		{
			"MintMe",
			params.MintMeChainConfig,
			params.MintMeGenesisHash,
		},
	}
	for _, tt := range tests {
		cs := []uint64{0}
		for _, f := range gatherForks(tt.config) {
			cs = append(cs, f-1, f, f+1)
		}
		fmt.Printf("##### %s\n", tt.name)
		fmt.Println()
		fmt.Printf("- Genesis Hash: `0x%x`\n", tt.genesisHash)
		forks := gatherForks(tt.config)
		forksS := []string{}
		for _, fi := range forks {
			forksS = append(forksS, strconv.Itoa(int(fi)))
		}
		fmt.Printf("- Forks: `%s`\n", strings.Join(forksS, "`,`"))
		fmt.Println()
		fmt.Println("| Head Block Number | `FORK_HASH` | `FORK_NEXT` | RLP Encoded (Hex) |")
		fmt.Println("| --- | --- | --- | --- |")
		for _, c := range cs {
			id := NewID(tt.config, tt.genesisHash, c)
			isCanonical := false
			for _, fi := range forks {
				if c == fi {
					isCanonical = true
				}
			}
			r, _ := rlp.EncodeToBytes(id)
			if isCanonical {
				fmt.Printf("| __head=%d__ | FORK_HASH=%x | FORK_NEXT=%d | %x |\n", c, id.Hash, id.Next, r)

			} else {
				fmt.Printf("| head=%d | FORK_HASH=%x | FORK_NEXT=%d | %x |\n", c, id.Hash, id.Next, r)
			}

			gethConfig := &coregeth.CoreGethChainConfig{}
			err := confp.Convert(tt.config, gethConfig)
			if err != nil {
				t.Fatal(err)
			}
			generatedCases = append(generatedCases, &testCaseJSON{
				ChainConfig: gethConfig,
				GenesisHash: tt.genesisHash,
				Head:        c,
				ForkHash:    common.BytesToHash(id.Hash[:]),
				ForkNext:    id.Next,
				ForkIDRLP:   common.BytesToHash(r),
			})
		}
		fmt.Println()
		fmt.Println()
	}
}
