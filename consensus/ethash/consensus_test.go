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

package ethash

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/types/goethereum"
)

type diffTest struct {
	ParentTimestamp    uint64
	ParentDifficulty   *big.Int
	CurrentTimestamp   uint64
	CurrentBlocknumber *big.Int
	CurrentDifficulty  *big.Int
}

func (d *diffTest) UnmarshalJSON(b []byte) (err error) {
	var ext struct {
		ParentTimestamp    string
		ParentDifficulty   string
		CurrentTimestamp   string
		CurrentBlocknumber string
		CurrentDifficulty  string
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return err
	}

	d.ParentTimestamp = math.MustParseUint64(ext.ParentTimestamp)
	d.ParentDifficulty = math.MustParseBig256(ext.ParentDifficulty)
	d.CurrentTimestamp = math.MustParseUint64(ext.CurrentTimestamp)
	d.CurrentBlocknumber = math.MustParseBig256(ext.CurrentBlocknumber)
	d.CurrentDifficulty = math.MustParseBig256(ext.CurrentDifficulty)

	return nil
}

func TestCalcDifficulty(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "tests", "testdata", "BasicTests", "difficulty.json"))
	if err != nil {
		t.Skip(err)
	}
	defer file.Close()

	tests := make(map[string]diffTest)
	err = json.NewDecoder(file).Decode(&tests)
	if err != nil {
		t.Fatal(err)
	}

	config := &goethereum.ChainConfig{HomesteadBlock: big.NewInt(1150000)}

	for name, test := range tests {
		number := new(big.Int).Sub(test.CurrentBlocknumber, big.NewInt(1))
		diff := CalcDifficulty(config, test.CurrentTimestamp, &types.Header{
			Number:     number,
			Time:       test.ParentTimestamp,
			Difficulty: test.ParentDifficulty,
		})
		if diff.Cmp(test.CurrentDifficulty) != 0 {
			t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
		}
	}
}

func TestEthash_ElectCanonical(t *testing.T) {
	var (
		one = big.NewInt(1)
		two = big.NewInt(2)
	)
	cases := []struct {
		currentTD, proposedTD uint64
		current, proposed     *types.Header
		preserve              func(header *types.Header) bool

		expect     bool
		stochastic bool // expect that we cannot expect with precision
		// no error check (error will never be returned)
	}{
		// Total difficulty condition (prefer greater).
		{
			expect:    true,
			currentTD: 42, proposedTD: 43,
		},
		{
			expect:    false,
			currentTD: 43, proposedTD: 42,
		},

		// Number condition (prefer lesser).
		{
			expect: true,

			current: &types.Header{Number: two}, proposed: &types.Header{Number: one},
			currentTD: 42, proposedTD: 42,
		},
		{
			expect:     false,
			stochastic: true,

			current: &types.Header{Number: two}, proposed: &types.Header{Number: two},
			currentTD: 42, proposedTD: 42,
		},
	}

	eh := NewFaker()
	for i, c := range cases {
		preferProposed, err := eh.ElectCanonical(nil, new(big.Int).SetUint64(c.currentTD), new(big.Int).SetUint64(c.proposedTD), c.current, c.proposed, c.preserve)
		if err != nil {
			t.Fatalf("case: %d, want: <nil>, got: %v", i, err)
		}

		if !c.stochastic && preferProposed != c.expect {
			t.Errorf("case: %d, want: %v, got: %v", i, c.expect, preferProposed)
		}

	}
}
