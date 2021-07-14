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

package confp

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/params/types/ctypes"
)

var (
	// compatibleProtocolNameSchemes define matchable naming schemes used by configuration methods
	// that are not incompatible with configuration either having or lacking them.
	compatibleProtocolNameSchemes = []string{
		"ECBP", // "Ethereum Classic Best Practice"
		"EBP",  // "Ethereum Best Practice"
	}
)

func nameSignalsCompatibility(name string) bool {
	for _, s := range compatibleProtocolNameSchemes {
		if regexp.MustCompile(s).MatchString(name) {
			return true
		}
	}
	return false
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *uint64
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func NewCompatError(what string, storedblock, newblock *uint64) *ConfigCompatError {
	var rew *uint64
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || *storedblock < *newblock:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && *rew > 0 {
		err.RewindTo = *rew - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	var have, want interface{}
	if err.StoredConfig != nil {
		have = *err.StoredConfig
	}
	if err.NewConfig != nil {
		want = *err.NewConfig
	}
	if have == nil {
		have = "nil"
	}
	if want == nil {
		want = "nil"
	}
	return fmt.Sprintf("mismatching %s in database (old: %v, new: %v, rewindto %d)", err.What, have, want, err.RewindTo)
}

type ConfigValidError struct {
	What string
	A, B interface{}
}

func NewValidErr(what string, a, b interface{}) *ConfigValidError {
	return &ConfigValidError{
		What: what,
		A:    a,
		B:    b,
	}
}

func (err *ConfigValidError) Error() string {
	return fmt.Sprintf("%s, %v/%v", err.What, err.A, err.B)
}

func IsEmpty(anything interface{}) bool {
	if anything == nil {
		return true
	}
	return reflect.DeepEqual(anything, reflect.Zero(reflect.TypeOf(anything)).Interface())
}

func IsValid(conf ctypes.ChainConfigurator, head *uint64) *ConfigValidError {

	// head-agnostic logic
	if conf.GetNetworkID() == nil {
		return NewValidErr("NetworkID cannot be nil", "!=nil", conf.GetNetworkID())
	}
	if head == nil {
		return nil
	}

	// head-full logic
	var bhead = new(big.Int).SetUint64(*head)

	if conf.IsEnabled(conf.GetEIP155Transition, bhead) && conf.GetChainID() == nil {
		return NewValidErr("EIP155 requires ChainID. A:EIP155/B:ChainID", conf.GetEIP155Transition(), conf.GetChainID())
	}

	return nil
}

func Compatible(head *uint64, a, b ctypes.ChainConfigurator) *ConfigCompatError {
	// Iterate checkCompatible to find the lowest conflict.
	var lastErr *ConfigCompatError
	for {
		err := compatible(head, a, b)
		if err == nil || (lastErr != nil && err.RewindTo == lastErr.RewindTo) {
			break
		}
		lastErr = err
		head = &err.RewindTo
	}
	return lastErr
}

func compatible(head *uint64, a, b ctypes.ChainConfigurator) *ConfigCompatError {
	aFns, aNames := Transitions(a)
	bFns, _ := Transitions(b)
	for i, afn := range aFns {
		// Skip cross-compatible namespaced transition names, assuming
		// these will not be enforced as hardforks.
		if nameSignalsCompatibility(aNames[i]) {
			continue
		}
		if err := func(c1, c2, head *uint64) *ConfigCompatError {
			if isForkIncompatible(c1, c2, head) {
				return NewCompatError("incompatible fork value: "+aNames[i], c1, c2)
			}
			return nil
		}(afn(), bFns[i](), head); err != nil {
			return err
		}
	}
	if head == nil {
		return nil
	}
	if a.IsEnabled(a.GetEIP155Transition, new(big.Int).SetUint64(*head)) {
		if a.GetChainID().Cmp(b.GetChainID()) != 0 {
			return NewCompatError("mismatching chain ids after EIP155 transition", a.GetEIP155Transition(), b.GetEIP155Transition())
		}
	}

	return nil
}

func Equivalent(a, b ctypes.ChainConfigurator) error {
	if a.GetConsensusEngineType() != b.GetConsensusEngineType() {
		return fmt.Errorf("mismatch consensus engine types, A: %s, B: %s", a.GetConsensusEngineType(), b.GetConsensusEngineType())
	}

	// Check forks sameness.
	fa, fb := Forks(a), Forks(b)
	if len(fa) != len(fb) {
		return fmt.Errorf("different fork count: %d / %d (%v / %v)", len(fa), len(fb), fa, fb)
	}
	for i := range fa {
		if fa[i] != fb[i] {
			if fa[i] == math.MaxUint64 {
				return fmt.Errorf("fa bigmax: %d", fa[i])
			}
			if fb[i] == math.MaxUint64 {
				return fmt.Errorf("fb bigmax: %d", fb[i])
			}
			return fmt.Errorf("fork index %d not same: %d / %d", i, fa[i], fb[i])
		}
	}

	// Check initial, at- and around-fork, and eventual compatibility.
	var testForks = []uint64{}
	copy(testForks, fa)
	// Don't care about dupes.
	for _, f := range fa {
		testForks = append(testForks, f-1)
	}
	testForks = append(testForks, 0, math.MaxUint64)

	// essentiallyEquivalent treats nil and bitsize-max numbers as essentially equivalent.
	essentiallyEquivalent := func(x, y *uint64) bool {
		if x == nil && y != nil {
			return *y == math.MaxUint64 ||
				*y == 0x7FFFFFFFFFFFFFFF ||
				*y == 0x7FFFFFFFFFFFFFF ||
				*y == 0x7FFFFFFFFFFFFF
		}
		if x != nil && y == nil {
			return *x == math.MaxUint64 ||
				*x == 0x7FFFFFFFFFFFFFFF ||
				*x == 0x7FFFFFFFFFFFFFF ||
				*x == 0x7FFFFFFFFFFFFF
		}
		return false
	}
	for _, h := range testForks {
		if err := Compatible(&h, a, b); err != nil {
			if !essentiallyEquivalent(err.StoredConfig, err.NewConfig) {
				return err
			}
		}
	}

	if a.GetConsensusEngineType() == ctypes.ConsensusEngineT_Ethash {
		for _, f := range fa { // fa and fb are fork-equivalent
			ar := ctypes.EthashBlockReward(a, new(big.Int).SetUint64(f))
			br := ctypes.EthashBlockReward(b, new(big.Int).SetUint64(f))
			if ar.Cmp(br) != 0 {
				return fmt.Errorf("mismatch block reward, fork block: %v, A: %v, B: %v", f, ar, br)
			}
			// TODO: add difficulty comparison
			// Currently tough/complex to do because of necessary overhead (ie build a parent block).
		}
	} else if a.GetConsensusEngineType() == ctypes.ConsensusEngineT_Clique {
		if a.GetCliqueEpoch() != b.GetCliqueEpoch() {
			return fmt.Errorf("mismatch clique epochs: A: %v, B: %v", a.GetCliqueEpoch(), b.GetCliqueEpoch())
		}
		if a.GetCliquePeriod() != b.GetCliquePeriod() {
			return fmt.Errorf("mismatch clique periods: A: %v, B: %v", a.GetCliquePeriod(), b.GetCliquePeriod())
		}
	} else if a.GetConsensusEngineType() == ctypes.ConsensusEngineT_Keccak {
		for _, f := range fa { // fa and fb are fork-equivalent
			ar := ctypes.KeccakBlockReward(a, new(big.Int).SetUint64(f))
			br := ctypes.KeccakBlockReward(b, new(big.Int).SetUint64(f))
			if ar.Cmp(br) != 0 {
				return fmt.Errorf("mismatch block reward, fork block: %v, A: %v, B: %v", f, ar, br)
			}
			// TODO: add difficulty comparison
			// Currently tough/complex to do because of necessary overhead (ie build a parent block).
		}
	}
	return nil
}

// Transitions gets all available transition (fork) functions and their names for a ChainConfigurator.
func Transitions(conf ctypes.ChainConfigurator) (fns []func() *uint64, names []string) {
	names = []string{}
	fns = []func() *uint64{}
	k := reflect.TypeOf(conf)
	for i := 0; i < k.NumMethod(); i++ {
		method := k.Method(i)
		if !strings.HasPrefix(method.Name, "Get") || !strings.HasSuffix(method.Name, "Transition") {
			continue
		}
		m := reflect.ValueOf(conf).MethodByName(method.Name).Interface()
		fns = append(fns, m.(func() *uint64))
		names = append(names, method.Name)
	}
	return fns, names
}

// Forks returns non-nil, non <maxUin64>, unique sorted forks for a ChainConfigurator.
func Forks(conf ctypes.ChainConfigurator) []uint64 {
	var forks []uint64
	var forksM = make(map[uint64]struct{}) // Will key for uniqueness as fork numbers are appended to slice.

	transitions, names := Transitions(conf)
	for i, tr := range transitions {
		// Skip cross-compatible namespaced transition names, assuming
		// these will not be enforced as hardforks.
		if nameSignalsCompatibility(names[i]) {
			continue
		}
		// Extract the fork rule block number and aggregate it
		response := tr()
		if response == nil ||
			*response == math.MaxUint64 ||
			*response == 0x7fffffffffffff ||
			*response == 0x7FFFFFFFFFFFFFFF {
			continue
		}

		// Only append unique fork numbers, excluding 0 (genesis config is not considered a fork)
		if _, ok := forksM[*response]; !ok && *response != 0 {
			forks = append(forks, *response)
			forksM[*response] = struct{}{}
		}
	}
	sort.Slice(forks, func(i, j int) bool {
		return forks[i] < forks[j]
	})

	return forks
}

func isForkIncompatible(a, b, head *uint64) bool {
	return (isForked(a, head) || isForked(b, head)) && !u2Equal(a, b)
}

func isForked(x, head *uint64) bool {
	if x == nil || head == nil {
		return false
	}
	return *x <= *head
}

func u2Equal(x, y *uint64) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return *x == *y
}
