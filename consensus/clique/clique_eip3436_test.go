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

package clique

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/params/types/genesisT"
	"github.com/holiman/uint256"
)

type cliqueEIP3436TestCase struct {
	// The number of signers (aka validators).
	// These addresses will be randomly generated on demand for each scenario
	// and sorted per status quo Clique spec.
	// They are referenced by their cardinality in this sorted list.
	lenSigners int

	// commonBlocks defines blocks by which validator should sign them.
	// Validators are referenced by their index in the sorted signers list,
	// which sorts by address.
	commonBlocks []int

	// forks defines blocks by which validator should sign them.
	// These forks will be generated and attempted to be imported into
	// the chain. We expect that import should always succeed for all blocks,
	// with this test measuring only if the expected fork head actually gets
	// canonical preference.
	forks [][]int

	// orderForkImport elects the order in which to import the generated forks.
	// We need this because if we're testing for a random value like lowest hash (asserting
	// that the chain with the head having the lowest hash prevails), then
	// we need to make sure that that fork gets imported second so that we don't incidentally
	// use arbitrary canonical-election preferences.
	// If nil, the defined (coded) order will be used.
	orderForkImport func(forks [][]*types.Block)

	// assertions are functions used to assert the expectations of
	// fork heads in chain context against the specification's requirements, eg.
	// make sure that the forks really do have equal total difficulty, or equal block numbers.
	// This way we know that the deciding condition for achieving canonical status
	// really is what we think it is (and not, eg. total difficulty).
	assertions []func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header)

	cliqueConfig *ctypes.CliqueConfig
}

var cliqueEIP3436TestPeriod = uint64(1)
var cliqueConfigEIP3436 = &ctypes.CliqueConfig{
	Period:            cliqueEIP3436TestPeriod,
	Epoch:             0,
	EIP3436Transition: big.NewInt(0),
}
var cliqueConfigNoEIP3436 = &ctypes.CliqueConfig{
	Period:            cliqueEIP3436TestPeriod,
	Epoch:             0,
	EIP3436Transition: nil,
}

func TestCliqueEIP3436_Scenario1_positive(t *testing.T) {
	testCliqueEIP3436(t, cliqueEIP3436TestCase{
		// SCENARIO-1
		// signers: A..H (8 count)
		//
		/*
			Step 1
			A fully in-order chain exists and validator 8 has just produced an in-turn block.

			1A x
			2B  x
			3C   x
			4D    x
			5E     x
			6F      x
			7G       x
			8H        x

			Step 2
			... and then validators 5, 7, and 8 go offline.

			1A x
			2B  x
			3C   x
			4D    x
			5E     x   -
			6F      x
			7G       x -
			8H        x-

			Step 3
			Two forks form, one with an in-order block from validator 1
			and then an out of order block from validator 3.

			The second fork forms from validators 2, 4, and 6 in order.
			Both have a net total difficulty of 3 more than the common ancestor.

			1A x        y
			2B  x       z
			3C   x       y
			4D    x      z
			5E     x   -
			6F      x     z
			7G       x -
			8H        x-
		*/
		lenSigners:   8,
		commonBlocks: []int{1, 2, 3, 4, 5, 6, 7, 0, 1, 2, 3, 4, 5, 6, 7},
		forks: [][]int{
			{1, 3, 5}, // 2, 4, 6
			{0, 2},    // 1, 3
		},
		assertions: []func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header){
			assertEqualTotalDifficulties,
			assertCanonical(shorterFork),
		},
		cliqueConfig: cliqueConfigEIP3436,
	})
}

func TestCliqueEIP3436_Scenario1_negative(t *testing.T) {
	testCliqueEIP3436(t, cliqueEIP3436TestCase{
		// SCENARIO-1 without EIP-3436 (negative test)
		lenSigners:   8,
		commonBlocks: []int{1, 2, 3, 4, 5, 6, 7, 0, 1, 2, 3, 4, 5, 6, 7},
		forks: [][]int{
			{1, 3, 5}, // 2, 4, 6
			{0, 2},    // 1, 3
		},
		assertions: []func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header){
			assertEqualTotalDifficulties,
			assertCanonical(shorterFork),
		},
		cliqueConfig: cliqueConfigNoEIP3436,
	})
}

func TestCliqueEIP3436_Scenario2_positive(t *testing.T) {

	// This scenario was originally included in the specification, but
	// was discovered to be invalid.
	// It is persisted here with a deviation from the specification to use forks of
	// equal length.
	// See comments below for rationale about original brokenness.

	testCliqueEIP3436(t, cliqueEIP3436TestCase{
		// SCENARIO-2

		/*
			Step 1
			For the second scenario with the same validator set and in-order chain with
			validator 7 having just produced an in order block, then validators 7 and 8 go offline.

			1A x
			2B  x
			3C   x
			4D    x
			5E     x
			6F      x
			7G       x
			8H

			1A x
			2B  x
			3C   x
			4D    x
			5E     x
			6F      x
			7G       x-
			8H        -

			Two forks form, 1,3,5 on one side and 2,4,6 on the other.
			Both forks become aware of the other fork after producing their third block.
			In this case both forks have equal total difficulty and equal length.

			1A x       x
			2B  x      y
			3C   x      x
			4D    x     y
			5E     x     x
			6F      x    y
			7G       x-
			8H        -

			NOTE(meowsbits): This scenario (as originally defined) yields a "recently signed" error
			when attempting to import Signer 5 (really #6 b/c zero-indexing) into the
			second fork.

			On that fork, the sequence of signers is specified to be
			... 7, 0, 1, 2, 3, 4, 5, 6, 1, 3, 5

			(vs. the other fork)
			... 7, 0, 1, 2, 3, 4, 5, 6, 0, 2, 4

			The condition for "recently signed" is (from *Clique#verifySeal):

			// Signer is among recents, only fail if the current block doesn't shift it out
			if limit := uint64(len(snap.Signers)/2 + 1); seen > number-limit {
				return errRecentlySigned
			}

			Evaluated, this yields

			=> recently signed: limit=(8/2+1)=5 seen=13 number=17 number-limit=12

			RESOLUTION: Use forks with length 2 instead of 3.
			This tests at least that rules 3 and 4 function work.

		*/
		lenSigners:   8,
		commonBlocks: []int{1, 2, 3, 4, 5, 6, 7, 0, 1, 2, 3, 4, 5, 6},
		forks: [][]int{
			// These expected values have been truncated to 2 block each instead
			// of the EIP's described 3 blocks (per inline comments).
			{0, 2}, // 1, 3, 5
			{1, 3}, // 2, 4, 6
		},
		assertions: []func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header){
			assertEqualTotalDifficulties,
			assertEqualNumbers,
			assertEIP3436Head_Rule3Rule4,
		},
		cliqueConfig: cliqueConfigEIP3436,
	})
}

func TestCliqueEIP3436_Scenario2_negative(t *testing.T) {
	testCliqueEIP3436(t, cliqueEIP3436TestCase{
		// SCENARIO-2 without EIP-3436 (negative test)
		lenSigners:   8,
		commonBlocks: []int{1, 2, 3, 4, 5, 6, 7, 0, 1, 2, 3, 4, 5, 6},
		forks: [][]int{
			// These expected values have been truncated to 2 block each instead
			// of the EIP's described 3 blocks (per inline comments).
			{0, 2}, // 1, 3, 5
			{1, 3}, // 2, 4, 6
		},
		assertions: []func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header){
			assertEqualTotalDifficulties,
			assertEqualNumbers,
			assertCanonical(func(t *testing.T, forkHeads ...*types.Header) int {
				// Prefer the first-seen fork.
				return 0
			}),
		},
		cliqueConfig: cliqueConfigNoEIP3436,
	})
}

func TestCliqueEIP3436_Scenario3_positive(t *testing.T) {
	testCliqueEIP3436(t, cliqueEIP3436TestCase{
		// SCENARIO-3
		// https://ethereum-magicians.org/t/eip-3436-expanded-clique-block-choice-rule/5809/3?u=meowsbits

		/*
			Step 1
			For the second scenario with the same validator set and in-order chain with
			validator 7 having just produced an in order block, then validator 8 goes offline.

			1A x
			2B  x
			3C   x
			4D    x
			5E     x
			6F      x
			7G       x
			8H

			1A x
			2B  x
			3C   x
			4D    x
			5E     x
			6F      x
			7G       x
			8H        -

			Here’s a revised one. 8 nodes, zero based. 0-6 all produce in-order blocks,
			then a netsplit. 0, 2, and 3 on the first fork and 1, 4, 6, 7 on the second fork, and 5 goes offline.
			7, 0, and 1 all missed an important in-turn block.

			1A x       x
			2B  x      y
			3C   x       x
			4D    x     x
			5E     x     y
			6F      x -
			7G       x
			8H          y

		*/
		lenSigners:   8,
		commonBlocks: []int{1, 2, 3, 4, 5, 6, 7, 0, 1, 2, 3, 4, 5, 6},
		forks: [][]int{
			{0, 3, 2},
			{1, 7, 4},
		},
		orderForkImport: getSortHashDescendingFn(t),
		assertions: []func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header){
			assertEqualTotalDifficulties,
			assertEqualNumbers,
			assertEIP3436Head_Rule3Rule4,
		},
		cliqueConfig: cliqueConfigEIP3436,
	})
}

func getSortHashDescendingFn(t *testing.T) func(forks [][]*types.Block) {
	return func(forks [][]*types.Block) {
		sf := sortableForks_hashDescending(forks)
		sort.Sort(sf)
		headsOrdered := []string{}
		for _, s := range sf {
			headsOrdered = append(headsOrdered, s[len(s)-1].Hash().Hex()[:8])
		}
		t.Logf("SORTED DESC: %s", headsOrdered)
	}
}

type sortableForks_hashDescending [][]*types.Block

func (sf sortableForks_hashDescending) Len() int {
	return len(sf)
}

func (sf sortableForks_hashDescending) Less(i, j int) bool {
	heads := make([]*types.Header, sf.Len())
	for i2, i3 := range sf {
		if i2 == i || i2 == j {
			heads[i2] = i3[len(i3)-1].Header()
		}
	}
	return lowerHash(nil, heads...) == 0
}

func (sf sortableForks_hashDescending) Swap(i, j int) {
	sf[i], sf[j] = sf[j], sf[i]
}

func testCliqueEIP3436(t *testing.T, testConfig cliqueEIP3436TestCase) {
	// Create the account pool and generate the initial set of signerAddressesSorted
	accountsPool := newTesterAccountPool()

	db := rawdb.NewMemoryDatabase()

	// Assemble a chain of headers from the cast votes
	chainConfig := *params.TestChainConfig

	chainConfig.Ethash = nil
	chainConfig.Clique = testConfig.cliqueConfig
	engine := New(chainConfig.Clique, db)
	engine.fakeDiff = false

	signerAddressesSorted := make([]common.Address, testConfig.lenSigners)
	for i := 0; i < testConfig.lenSigners; i++ {
		signerAddressesSorted[i] = accountsPool.address([]string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K"}[i])
	}
	for j := 0; j < len(signerAddressesSorted); j++ {
		for k := j + 1; k < len(signerAddressesSorted); k++ {
			if bytes.Compare(signerAddressesSorted[j][:], signerAddressesSorted[k][:]) > 0 {
				signerAddressesSorted[j], signerAddressesSorted[k] = signerAddressesSorted[k], signerAddressesSorted[j]
			}
		}
	}

	// Pretty logging of the sorted signers list.
	logSortedSigners := ""
	for j, s := range signerAddressesSorted {
		logSortedSigners += fmt.Sprintf("%d: (%s) %s\n", j, accountsPool.name(s), s.Hex())
	}
	t.Logf("SORTED SIGNERS:\n%s\n", logSortedSigners)

	// Create the genesis block with the initial set of signerAddressesSorted
	genesis := &genesisT.Genesis{
		ExtraData: make([]byte, extraVanity+common.AddressLength*len(signerAddressesSorted)+extraSeal),
	}
	for j, signer := range signerAddressesSorted {
		copy(genesis.ExtraData[extraVanity+j*common.AddressLength:], signer[:])
	}

	genesisBlock := core.MustCommitGenesis(db, genesis)

	// Create a pristine blockchain with the genesis injected
	chain, err := core.NewBlockChain(db, nil, &chainConfig, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Errorf("failed to create test chain: %v", err)
		return
	}

	// getNextBlockWithSigner generates a block given a parent block and the
	// signer index of the validator that should sign it.
	// It will use the Clique snapshot function to see if the signer is in turn or not,
	// and will assign difficulty appropriately.
	// After signing, it sanity-checks the Engine.Author method against the newly-signed
	// block value to make sure that signing is done properly.
	getNextBlockWithSigner := func(chr consensus.ChainHeaderReader, parentBlock *types.Block, signerIndex int) *types.Block {
		signerAddress := signerAddressesSorted[signerIndex]
		signerName := accountsPool.name(signerAddress)

		generatedBlocks, _ := core.GenerateChain(&chainConfig, parentBlock, engine, db, 1, nil)
		block := generatedBlocks[0]

		// Get the header and prepare it for signing
		header := block.Header()
		header.ParentHash = parentBlock.Hash()

		header.Time = parentBlock.Time() + cliqueEIP3436TestPeriod

		// See if our required signer is in or out of turn and assign difficulty respectively.
		difficulty := diffInTurn

		// If the snapshot reports this signer is out of turn, use out-of-turn difficulty.
		snap, err := engine.snapshot(chr, parentBlock.NumberU64(), parentBlock.Hash(), nil)
		if err != nil {
			t.Fatalf("snap err: %v", err)
		}
		inturn := snap.inturn(parentBlock.NumberU64()+1, signerAddress)
		if !inturn {
			difficulty = diffNoTurn
		}
		header.Difficulty = difficulty

		// Generate the signature, embed it into the header and the block.
		header.Extra = make([]byte, extraVanity+extraSeal) // allocate byte slice

		// Sign the header with the associated validator's key.
		accountsPool.sign(header, signerName)

		// Double check to see what the Clique Engine thinks the signer of this block is.
		// It obviously should be the address we just used.
		author, err := engine.Author(header)
		if err != nil {
			t.Fatalf("author error: %v", err)
		}
		if wantSigner := accountsPool.address(signerName); author != wantSigner {
			t.Fatalf("header author != wanted signer: author: %s, signer: %s", author.Hex(), wantSigner.Hex())
		}

		out := block.WithSeal(header)
		t.Logf("BLOCK: num=%d hash=%x d=%d signer[%d name=%s inturn=%v addr=%s]\n", out.NumberU64(), out.Hash().Bytes()[:4], out.Difficulty(),
			signerIndex, signerName, inturn, signerAddress.Hex()[:8])
		return out
	}

	mockHeaderReader := &mockChainHeaderReader{}

	// Build the common segment
	commonSegmentBlocks := []*types.Block{genesisBlock}
	for i := 0; i < len(testConfig.commonBlocks); i++ {

		signerIndex := testConfig.commonBlocks[i]
		parentBlock := commonSegmentBlocks[len(commonSegmentBlocks)-1]

		bl := getNextBlockWithSigner(chain, parentBlock, signerIndex)

		if k, err := chain.InsertChain([]*types.Block{bl}); err != nil || k != 1 {
			t.Fatalf("failed to import block %d, count: %d, err: %v", i, k, err)
		}
		mockHeaderReader.push(bl)

		commonSegmentBlocks = append(commonSegmentBlocks, bl)
	}

	t.Logf("--- COMMON SEGMENT, td=%v", chain.GetTd(chain.CurrentHeader().Hash(), chain.CurrentHeader().Number.Uint64()))

	// forkHeads holds the heads of each scenario's fork.
	forkHeads := make([]*types.Header, len(testConfig.forks))

	// forks will hold the blocks of each fork.
	// It will be allowed to be mutated by a test configuration function
	// to pass import order election responsibility to the test configuration.
	forks := make([][]*types.Block, len(testConfig.forks))

	// Create and import blocks for all the scenario's forks.
	for scenarioForkIndex, forkBlockSigners := range testConfig.forks {

		forkBlocks := []*types.Block{}

		// Range over the blocks to be created for each fork, defined by the signer of that block.
		for _, signerInt := range forkBlockSigners {

			var parent *types.Block
			if len(forkBlocks) == 0 {
				// If this is the first block of the fork, the parent should be the last common block.
				parent = commonSegmentBlocks[len(commonSegmentBlocks)-1]
			} else {
				parent = forkBlocks[len(forkBlocks)-1]
			}

			bl := getNextBlockWithSigner(mockHeaderReader, parent, signerInt)

			mockHeaderReader.push(bl)
			forkBlocks = append(forkBlocks, bl)

		} // End fork block imports.

		// Purge mock header reader blocks from the last fork, cleaning up for next.
		mockHeaderReader.blocks = mockHeaderReader.blocks[:len(mockHeaderReader.blocks)-len(forkBlocks)]

		forks[scenarioForkIndex] = forkBlocks

		forkHeads[scenarioForkIndex] = forkBlocks[len(forkBlocks)-1].Header()

		t.Logf("--- FORK: %d len=%d", scenarioForkIndex, len(forkBlocks))

	} // End scenario fork imports.

	// Order the forks if prescribed.
	if testConfig.orderForkImport != nil {
		testConfig.orderForkImport(forks)
	}

	// Finally, import the forks' blocks in the fork order the config wants.
	for fi, fork := range forks {
		if k, err := chain.InsertChain(fork); err != nil {
			t.Fatalf("failed to import block %d, count: %d, err: %v", fi, k, err)
		} else {
			bl := fork[len(fork)-1]
			t.Logf("INSERTED TD=%v", chain.GetTd(bl.Hash(), bl.NumberU64()))
		}
	}

	// Run arbitrary assertion tests, ie. make sure that we've created forks that meet
	// the expected scenario characteristics.
	for _, f := range testConfig.assertions {
		f(t, chain, forkHeads...)
	}
}

// mockChainHeaderReader implements ChainHeaderReader for testing.
// We can't use the native Blockchain type because we want to postpone
// the actual import of blocks, but we still want to be able to use the snapshot method
// to check in/out-turn of signers.
type mockChainHeaderReader struct {
	blocks []*types.Block
}

func (m *mockChainHeaderReader) push(block *types.Block) {
	if len(m.blocks) == 0 {
		m.blocks = []*types.Block{}
	}
	m.blocks = append(m.blocks, block)
}

func (m *mockChainHeaderReader) Config() ctypes.ChainConfigurator {
	panic("implement me") // disused
}

func (m *mockChainHeaderReader) CurrentHeader() *types.Header {
	panic("implement me") // disused
}

func (m *mockChainHeaderReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	for _, block := range m.blocks {
		if block.Hash() == hash && block.NumberU64() == number {
			return block.Header()
		}
	}
	return nil
}

func (m *mockChainHeaderReader) GetHeaderByNumber(number uint64) *types.Header {
	for _, block := range m.blocks {
		if block.NumberU64() == number {
			return block.Header()
		}
	}
	return nil
}

func (m *mockChainHeaderReader) GetHeaderByHash(hash common.Hash) *types.Header {
	panic("implement me") // disused
}

// shorterFork defines logic returning the fork head the lower head number.
func shorterFork(t *testing.T, forkHeads ...*types.Header) int {
	// Prefer the shorter fork.
	minHeight := math.MaxBig63.Uint64()
	n := -1
	for i, head := range forkHeads {
		if h := head.Number.Uint64(); h < minHeight {
			n = i
			minHeight = h
		}
	}
	return n
}

// assertEqualTotalDifficulties fatals if the fork heads do not have equal total difficulties.
func assertEqualTotalDifficulties(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header) {
	d := new(big.Int)
	for i, head := range forkHeads {
		td := chain.GetTd(head.Hash(), head.Number.Uint64())
		if i == 0 {
			d.Set(td)
			continue
		}
		if d.Cmp(td) != 0 {
			t.Fatalf("want equal fork heads total difficulty")
		}
	}
}

// assertEqualNumbers fatals if the fork heads do not have equal numbers.
func assertEqualNumbers(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header) {
	n := new(big.Int)
	for i, head := range forkHeads {
		if i == 0 {
			n.Set(head.Number)
			continue
		}
		if n.Cmp(head.Number) != 0 {
			t.Fatalf("want equal fork head numbers")
		}
	}
}

// lowerHash defines logic returning the fork head having the lesser block hash.
func lowerHash(t *testing.T, forkHeads ...*types.Header) int {
	// Prefer the lowest hash.
	minHashV, overflow := uint256.FromBig(big.NewInt(0))
	if overflow {
		t.Fatalf("uint256 overflowed: 0")
	}
	n := -1
	for i, head := range forkHeads {
		hv, err := hashToUint256(head.Hash())
		if err != nil {
			t.Fatalf("uint256 err: %v, head.hex: %s", err, head.Hash().Hex())
		}
		if n == -1 || hv.Cmp(minHashV) < 0 {
			minHashV.Set(hv)
			n = i
		}
	}
	return n
}

// assertEIP3436Head_Rule3Rule4 checks that the combined logic of EIP3436's rules 3 and 4
// meet the outcome state of the got chain.
func assertEIP3436Head_Rule3Rule4(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header) {
	if len(forkHeads) != 2 {
		t.Fatalf("test only supports 2 heads")
	}
	head := chain.CurrentHeader()
	shouldSecond, err := chain.Engine().(*Clique).Eip3436Rule3rule4(chain, forkHeads[0], forkHeads[1])
	if err != nil {
		t.Fatalf("encountered error: %v", err)
	}
	want := forkHeads[0]
	if shouldSecond {
		want = forkHeads[1]
	}
	if head.Hash() != want.Hash() {
		t.Errorf("want: %s, got: %s", want.Hash().Hex()[:8], head.Hash().Hex()[:8])
	}
}

func assertCanonical(forkHeadChooser func(t *testing.T, forkHeads ...*types.Header) int) func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header) {
	return func(t *testing.T, chain *core.BlockChain, forkHeads ...*types.Header) {

		// Finally, check that the current chain head matches the
		// head of the wanted fork index.
		forkHeadHashes := make([]string, len(forkHeads))
		for i := 0; i < len(forkHeads); i++ {
			forkHeadHashes[i] = forkHeads[i].Hash().Hex()[:8]
		}
		if chain.CurrentHeader().Hash() != forkHeads[forkHeadChooser(t, forkHeads...)].Hash() {
			t.Errorf("wrong fork index head: got: %s\nFork heads:\n%s", chain.CurrentHeader().Hash().Hex(), forkHeadHashes)
		} else {
			t.Logf("CHAIN CURRENT HEAD: %x", chain.CurrentHeader().Hash().Bytes()[:8])
		}
	}
}

// TestClique_Eip3436Rule4_hashToUint256 unit tests to make sure that EIP3436's Rule 4
// logic converting hashes to uint256's works as expected.
func TestClique_Eip3436Rule4_hashToUint256(t *testing.T) {
	// "0x0f73844a381e7a0e002f128376a848d5f1193d07dc0570f1531d1c95f9e95a93"
	cases := []struct {
		hash common.Hash
		want *uint256.Int
	}{
		// This batch of cases is primarily concerned with checking
		// that hashes containing leading 0's don't return errors
		// (and that we're allowed to pass odd-length hex strings).
		{common.HexToHash("0xff73844a381e7a0e002f128376a848d5f1193d07dc0570f1531d1c95f9e95a93"), nil},
		{common.HexToHash("0x0f73844a381e7a0e002f128376a848d5f1193d07dc0570f1531d1c95f9e95a93"), nil},
		{common.HexToHash("0xf73844a381e7a0e002f128376a848d5f1193d07dc0570f1531d1c95f9e95a93"), nil},
	}
	for i, c := range cases {
		got, err := hashToUint256(c.hash)
		if err != nil {
			t.Errorf("case %d, error: %v (hash.hex=%s)", i, err, c.hash.Hex())
		}
		// Skip tests with want=nil
		if c.want != nil && ((got == nil && c.want != nil) || (got.Cmp(c.want) != 0)) {
			t.Errorf("case: %d, want: %v, got: %v", i, c.want, got)
		}
	}
}
