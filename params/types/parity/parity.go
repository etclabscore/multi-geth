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

package parity

import (
	"encoding/binary"
	"encoding/json"
	"math/big"
	"reflect"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
)

// ParityChainSpec is the chain specification format used by Parity.
type ParityChainSpec struct {
	Name    string `json:"name"`
	Datadir string `json:"dataDir"`
	Engine  struct {
		Ethash struct {
			Params struct {
				MinimumDifficulty      *math.HexOrDecimal256         `json:"minimumDifficulty"`
				DifficultyBoundDivisor *math.HexOrDecimal256         `json:"difficultyBoundDivisor"`
				DurationLimit          *math.HexOrDecimal256         `json:"durationLimit"`
				BlockReward            ctypes.Uint64BigValOrMapHex   `json:"blockReward"`
				DifficultyBombDelays   ctypes.Uint64BigMapEncodesHex `json:"difficultyBombDelays,omitempty"`

				// Caches.
				// These inferences require computation.
				// This makes it so that the 'heavy-lifting' only has to run once.
				// See ctypes.MapMeetsSpecification for this bespoke logic.
				eip649Inferred    bool
				eip649Transition  *ParityU64
				eip1234Inferred   bool
				eip1234Transition *ParityU64
				eip2384Inferred   bool
				eip2384Transition *ParityU64

				HomesteadTransition *ParityU64 `json:"homesteadTransition"`
				EIP100bTransition   *ParityU64 `json:"eip100bTransition"`

				CatalystBlock *ParityU64 `json:"catalyst_block,omitempty"`

				// Note: DAO fields will NOT be written to Parity configs from multi-geth.
				// The chains with DAO settings are already canonical and have existing chainspecs.
				// There is no need to replicate this information.
				DaoHardforkTransition  *ParityU64       `json:"daoHardforkTransition,omitempty"`
				DaoHardforkBeneficiary *common.Address  `json:"daoHardforkBeneficiary,omitempty"`
				DaoHardforkAccounts    []common.Address `json:"daoHardforkAccounts,omitempty"`

				BombDefuseTransition       *ParityU64 `json:"bombDefuseTransition"`
				ECIP1010PauseTransition    *ParityU64 `json:"ecip1010PauseTransition,omitempty"`
				ECIP1010ContinueTransition *ParityU64 `json:"ecip1010ContinueTransition,omitempty"`
				ECIP1017EraRounds          *ParityU64 `json:"ecip1017EraRounds,omitempty"`
			} `json:"params"`
		} `json:"Ethash,omitempty"`
		Clique struct {
			Params struct {
				Period *ParityU64 `json:"period,omitempty"`
				Epoch  *ParityU64 `json:"epoch,omitempty"`
			} `json:"params,omitempty"`
		} `json:"Clique,omitempty"`
		Keccak struct {
			Params struct {
				BlockReward ctypes.Uint64BigValOrMapHex `json:"blockReward"`
			} `json:"params"`
		} `json:"Keccak256,omitempty"`
		Lyra2 struct {
			Params struct {
				MinimumDifficulty *math.HexOrDecimal256 `json:"minimumDifficulty"`
			} `json:"params,omitempty"`
		} `json:"Lyra2,omitempty"`
	} `json:"engine"`

	Params struct {
		AccountStartNonce         *ParityU64 `json:"accountStartNonce,omitempty"`
		MaximumExtraDataSize      *ParityU64 `json:"maximumExtraDataSize,omitempty"`
		MinGasLimit               *ParityU64 `json:"minGasLimit,omitempty"`
		GasLimitBoundDivisor      *ParityU64 `json:"gasLimitBoundDivisor,omitempty"`
		NetworkID                 *ParityU64 `json:"networkID,omitempty"`
		ChainID                   *ParityU64 `json:"chainID,omitempty"`
		MaxCodeSize               *ParityU64 `json:"maxCodeSize,omitempty"`
		MaxCodeSizeTransition     *ParityU64 `json:"maxCodeSizeTransition,omitempty"`
		EIP98Transition           *ParityU64 `json:"eip98Transition,omitempty"`
		EIP150Transition          *ParityU64 `json:"eip150Transition,omitempty"`
		EIP160Transition          *ParityU64 `json:"eip160Transition,omitempty"`
		EIP161abcTransition       *ParityU64 `json:"eip161abcTransition,omitempty"`
		EIP161dTransition         *ParityU64 `json:"eip161dTransition,omitempty"`
		EIP155Transition          *ParityU64 `json:"eip155Transition,omitempty"`
		EIP140Transition          *ParityU64 `json:"eip140Transition,omitempty"`
		EIP211Transition          *ParityU64 `json:"eip211Transition,omitempty"`
		EIP214Transition          *ParityU64 `json:"eip214Transition,omitempty"`
		EIP658Transition          *ParityU64 `json:"eip658Transition,omitempty"`
		EIP145Transition          *ParityU64 `json:"eip145Transition,omitempty"`
		EIP1014Transition         *ParityU64 `json:"eip1014Transition,omitempty"`
		EIP1052Transition         *ParityU64 `json:"eip1052Transition,omitempty"`
		EIP1283Transition         *ParityU64 `json:"eip1283Transition,omitempty"`
		EIP1283DisableTransition  *ParityU64 `json:"eip1283DisableTransition,omitempty"`
		EIP1283ReenableTransition *ParityU64 `json:"eip1283ReenableTransition,omitempty"`
		EIP1344Transition         *ParityU64 `json:"eip1344Transition,omitempty"`
		EIP1884Transition         *ParityU64 `json:"eip1884Transition,omitempty"`
		EIP2028Transition         *ParityU64 `json:"eip2028Transition,omitempty"`
		EIP2315Transition         *ParityU64 `json:"eip2315Transition,omitempty"`
		EIP2537Transition         *ParityU64 `json:"eip2537Transition,omitempty"`
		EIP1706Transition         *ParityU64 `json:"eip1706Transition,omitempty"`  // FIXME, when and if i'm implemented in Parity
		EIP2929Transition         *ParityU64 `json:"eip2929Transition,omitempty"`  // FIXME, when and if i'm implemented in Parity
		EIP2930Transition         *ParityU64 `json:"eip2930Transition,omitempty"`  // FIXME, when and if i'm implemented in Parity
		EIP2565Transition         *ParityU64 `json:"eip2565Transition,omitempty"`  // FIXME, when and if i'm implemented in Parity
		EIP2718Transition         *ParityU64 `json:"eip2718Transition,omitempty"`  // FIXME, when and if i'm implemented in Parity
		ECIP1080Transition        *ParityU64 `json:"ecip1080Transition,omitempty"` // FIXME, when and if i'm implemented in Parity

		// supportedProtocolVersions is left here as a caching field only.
		// I don't think this feature is supported by Parity, but
		// this value allows the configurator converter to gracefully handle test cases for Parity
		// and for implementation design to be functional despite Parity not actually supporting the feature.
		supportedProtocolVersions []uint

		ForkBlock     *ParityU64   `json:"forkBlock,omitempty"`
		ForkCanonHash *common.Hash `json:"forkCanonHash,omitempty"`
	} `json:"params"`

	Genesis struct {
		Seal struct {
			Ethereum struct {
				Nonce   BlockNonce    `json:"nonce"`
				MixHash hexutil.Bytes `json:"mixHash"`
			} `json:"ethereum"`
		} `json:"seal"`

		Difficulty *math.HexOrDecimal256 `json:"difficulty"`
		Author     common.Address        `json:"author"`
		Timestamp  math.HexOrDecimal64   `json:"timestamp"`
		ParentHash common.Hash           `json:"parentHash"`
		ExtraData  hexutil.Bytes         `json:"extraData"`
		GasLimit   math.HexOrDecimal64   `json:"gasLimit"`
	} `json:"genesis"`

	Nodes    []string                                             `json:"nodes"`
	Accounts map[common.UnprefixedAddress]*ParityChainSpecAccount `json:"accounts"`
}

func (c *ParityChainSpec) String() string {
	cc := &ParityChainSpec{}
	*cc = *c
	if cc.Engine.Ethash.Params.DaoHardforkAccounts != nil {
		cc.Engine.Ethash.Params.DaoHardforkAccounts = nil
	}
	j, _ := json.MarshalIndent(cc, "", "    ")
	return string(j)
}

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a sufficient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

// ParityU64 implements special Unmarshal interface for math2.HexOrDecimal64
// as well as a convenience method for converting to *big.Int.
type ParityU64 math.HexOrDecimal64

func (i ParityU64) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(hexutil.EncodeUint64(uint64(i)))), nil
}

func (i *ParityU64) UnmarshalJSON(input []byte) error {
	if len(input) == 0 {
		return nil
	}

	// 0x04
	var n math.HexOrDecimal64
	err := json.Unmarshal(input, &n)
	if err == nil {
		*i = (ParityU64)(n)
		return nil
	}

	// "4"
	s := string(input)
	s, _ = strconv.Unquote(s)
	b, ok := new(big.Int).SetString(s, 10)
	if ok {
		*i = ParityU64(b.Uint64())
		return nil
	}

	// 4
	var nu uint64
	err = json.Unmarshal(input, &nu)
	if err != nil {
		return err
	}
	*i = ParityU64(nu)
	return nil
}

func (i *ParityU64) Big() *big.Int {
	if i == nil {
		return nil
	}
	return new(big.Int).SetUint64(uint64(*i))
}

func (i *ParityU64) Uint64P() *uint64 {
	if i == nil {
		return nil
	}
	u := uint64(*i)
	return &u
}

func (i *ParityU64) SetUint64(n *uint64) *ParityU64 {
	if n == nil {
		i = nil
		return i
	}
	u := ParityU64(*n)
	*i = u
	return i
}

// ParityChainSpecAccount is the prefunded genesis account and/or precompiled
// contract definition.
type ParityChainSpecAccount struct {
	Balance math.HexOrDecimal256        `json:"balance"`
	Nonce   math.HexOrDecimal64         `json:"nonce,omitempty"`
	Code    hexutil.Bytes               `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	Builtin *ParityChainSpecBuiltin     `json:"builtin,omitempty"`
}

// ParityChainSpecBuiltin is the precompiled contract definition.
type ParityChainSpecBuiltin struct {
	Name              string                       `json:"name"`                         // Each builtin should has it own name
	Pricing           *ParityChainSpecPricingMaybe `json:"pricing"`                      // Each builtin should has it own price strategy
	ActivateAt        *ParityU64                   `json:"activate_at,omitempty"`        // ActivateAt can't be omitted if empty, default means no fork
	EIP1108Transition *ParityU64                   `json:"eip1108_transition,omitempty"` // EIP1108Transition can't be omitted if empty, default means no fork
}

type ParityChainSpecPricingMaybe struct {
	Map     map[*math.HexOrDecimal256]ParityChainSpecPricingPrice
	Pricing *ParityChainSpecPricing
}

type ParityChainSpecPricingPrice struct {
	ParityChainSpecPricing `json:"price"`
}

func (p *ParityChainSpecPricingMaybe) UnmarshalJSON(input []byte) error {

	// If old schema structure with "pricing" field
	pricing := ParityChainSpecPricing{}
	err := json.Unmarshal(input, &pricing)
	if err == nil && !reflect.DeepEqual(pricing, ParityChainSpecPricing{}) {
		p.Pricing = &pricing
		return nil
	}

	// Otherwise it's a map keyed on activation block numbers,
	// where the keys are strings and could be duplicates.
	// According to JSON specification we should use the last lexicographically
	// ordered value in case of duplicates.
	mm := make(map[string]ParityChainSpecPricingPrice)
	err = json.Unmarshal(input, &mm)
	if err != nil {
		return err
	}
	sl := []string{}
	for k := range mm {
		sl = append(sl, k)
	}
	sort.Strings(sl)
	p.Map = make(map[*math.HexOrDecimal256]ParityChainSpecPricingPrice)
	for _, s := range sl {
		p.Map[(*math.HexOrDecimal256)(math.MustParseBig256(s))] = mm[s]
	}
	if len(p.Map) == 0 {
		panic("0map")
	}
	return nil
}

func (p ParityChainSpecPricingMaybe) MarshalJSON() ([]byte, error) {
	if p.Map != nil {
		return json.Marshal(p.Map)
	}
	return json.Marshal(p.Pricing)
}

// ParityChainSpecPricing represents the different pricing models that builtin
// contracts might advertise using.
type ParityChainSpecPricing struct {
	Linear              *ParityChainSpecLinearPricing              `json:"linear,omitempty"`
	ModExp              *ParityChainSpecModExpPricing              `json:"modexp,omitempty"`
	AltBnPairing        *ParityChainSpecAltBnPairingPricing        `json:"alt_bn128_pairing,omitempty"`
	AltBnConstOperation *ParityChainSpecAltBnConstOperationPricing `json:"alt_bn128_const_operations,omitempty"`

	// Blake2F is the price per round of Blake2 compression
	Blake2F *ParityChainSpecBlakePricing `json:"blake2_f,omitempty"`

	BLS12ConstOperation *ParityChainSpecBLS12ConstOperationPricing `json:"bls12_const_operations,omitempty"`
	BLS12G1MultiExp     *ParityChainSpecBLS12MultiExpPricing       `json:"bls12_g1_multiexp,omitempty"`
	BLS12G2MultiExp     *ParityChainSpecBLS12MultiExpPricing       `json:"bls12_g2_multiexp,omitempty"`
	BLS12Pairing        *ParityChainSpecBLS12PairingPricing        `json:"bls12_pairing,omitempty"`
}

type ParityChainSpecLinearPricing struct {
	Base uint64 `json:"base"`
	Word uint64 `json:"word"`
}

type ParityChainSpecModExpPricing struct {
	Divisor uint64 `json:"divisor"`
}

type ParityChainSpecAltBnConstOperationPricing struct {
	Price                  uint64 `json:"price"`
	EIP1108TransitionPrice uint64 `json:"eip1108_transition_price,omitempty"` // Before Istanbul fork, this field is nil
}

type ParityChainSpecAltBnPairingPricing struct {
	Base                  uint64 `json:"base"`
	Pair                  uint64 `json:"pair"`
	EIP1108TransitionBase uint64 `json:"eip1108_transition_base,omitempty"` // Before Istanbul fork, this field is nil
	EIP1108TransitionPair uint64 `json:"eip1108_transition_pair,omitempty"` // Before Istanbul fork, this field is nil
}

type ParityChainSpecBlakePricing struct {
	GasPerRound uint64 `json:"gas_per_round"`
}

type ParityChainSpecBLS12ConstOperationPricing struct {
	Price uint64 `json:"price"`
}

type ParityChainSpecBLS12MultiExpPricing struct {
	Base uint64 `json:"base"`
}

type ParityChainSpecBLS12PairingPricing struct {
	Base uint64 `json:"base"`
	Pair uint64 `json:"pair"`
}

func (spec *ParityChainSpec) GetPrecompile(address common.Address, pricing ParityChainSpecPricing) *ParityU64 {
	if spec.Accounts == nil {
		return nil
	}
	acc, ok := spec.Accounts[common.UnprefixedAddress(address)]
	if !ok || acc.Builtin == nil || acc.Builtin.Pricing == nil {
		return nil
	}
	if acc.Builtin.Pricing.Map != nil {
		for k, v := range acc.Builtin.Pricing.Map {
			if reflect.DeepEqual(v.ParityChainSpecPricing, pricing) {
				activation := ParityU64(k.ToInt().Uint64())
				return &activation
			}
		}
		return nil
	}
	if reflect.DeepEqual(acc.Builtin.Pricing.Pricing, &pricing) {
		return acc.Builtin.ActivateAt
	}
	return nil
}

func (spec *ParityChainSpec) SetPrecompile(address byte, data *ParityChainSpecBuiltin) {
	if spec.Accounts == nil {
		spec.Accounts = make(map[common.UnprefixedAddress]*ParityChainSpecAccount)
	}
	a := common.UnprefixedAddress(common.BytesToAddress([]byte{address}))
	if _, exist := spec.Accounts[a]; !exist {
		spec.Accounts[a] = &ParityChainSpecAccount{}
	}
	spec.Accounts[a].Builtin = data
}

func (spec *ParityChainSpec) SetPrecompile2(address common.Address, name string, activationBlock *uint64, pricing ParityChainSpecPricing) {
	if activationBlock == nil {
		return
	}
	if spec.Accounts == nil {
		spec.Accounts = make(map[common.UnprefixedAddress]*ParityChainSpecAccount)
	}
	if _, exist := spec.Accounts[common.UnprefixedAddress(address)]; !exist {
		spec.Accounts[common.UnprefixedAddress(address)] = &ParityChainSpecAccount{}
	}

	bin := spec.Accounts[common.UnprefixedAddress(address)].Builtin
	defer func() {
		spec.Accounts[common.UnprefixedAddress(address)].Builtin = bin
	}()

	if bin == nil {
		bin = &ParityChainSpecBuiltin{}
	}

	bin.Name = name
	if bin.Pricing == nil {
		bin.Pricing = &ParityChainSpecPricingMaybe{
			Map:     make(map[*math.HexOrDecimal256]ParityChainSpecPricingPrice),
			Pricing: nil,
		}
	}

	// Always write in activation-map format.
	bin.Pricing.Map[math.NewHexOrDecimal256(int64(*activationBlock))] = ParityChainSpecPricingPrice{
		ParityChainSpecPricing: pricing,
	}
}
