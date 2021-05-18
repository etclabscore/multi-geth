package multigeth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/params/types/internal"
	"github.com/ethereum/go-ethereum/params/vars"
)

func newU64(u uint64) *uint64 {
	return &u
}

func bigNewU64(i *big.Int) *uint64 {
	if i == nil {
		return nil
	}
	return newU64(i.Uint64())
}

func setBig(i *big.Int, u *uint64) *big.Int {
	if u == nil {
		return nil
	}
	i = big.NewInt(int64(*u))
	return i
}

func (c *ChainConfig) GetAccountStartNonce() *uint64 {
	return internal.GlobalConfigurator().GetAccountStartNonce()
}
func (c *ChainConfig) SetAccountStartNonce(n *uint64) error {
	return internal.GlobalConfigurator().SetAccountStartNonce(n)
}
func (c *ChainConfig) GetMaximumExtraDataSize() *uint64 {
	return internal.GlobalConfigurator().GetMaximumExtraDataSize()
}
func (c *ChainConfig) SetMaximumExtraDataSize(n *uint64) error {
	return internal.GlobalConfigurator().SetMaximumExtraDataSize(n)
}
func (c *ChainConfig) GetMinGasLimit() *uint64 {
	return internal.GlobalConfigurator().GetMinGasLimit()
}
func (c *ChainConfig) SetMinGasLimit(n *uint64) error {
	return internal.GlobalConfigurator().SetMinGasLimit(n)
}
func (c *ChainConfig) GetGasLimitBoundDivisor() *uint64 {
	return internal.GlobalConfigurator().GetGasLimitBoundDivisor()
}
func (c *ChainConfig) SetGasLimitBoundDivisor(n *uint64) error {
	return internal.GlobalConfigurator().SetGasLimitBoundDivisor(n)
}

// GetNetworkID and the following Set/Getters for ChainID too
// are... opinionated... because of where and how currently the NetworkID
// value is designed.
// This can cause unexpected and/or counter-intuitive behavior, especially with SetNetworkID.
// In order to use these logic properly, one should call NetworkID setter before ChainID setter.
// FIXME.
func (c *ChainConfig) GetNetworkID() *uint64 {
	if c.NetworkID != 0 {
		return &c.NetworkID
	}
	if c.ChainID != nil {
		return newU64(c.ChainID.Uint64())
	}
	return newU64(vars.DefaultNetworkID)
}

func (c *ChainConfig) SetNetworkID(n *uint64) error {
	if n == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	if c.ChainID == nil {
		c.ChainID = new(big.Int).SetUint64(*n)
	}
	c.NetworkID = *n
	return nil
}

func (c *ChainConfig) GetChainID() *big.Int {
	return c.ChainID
}

func (c *ChainConfig) SetChainID(n *big.Int) error {
	c.ChainID = n
	return nil
}

func (c *ChainConfig) GetMaxCodeSize() *uint64 {
	return internal.GlobalConfigurator().GetMaxCodeSize()
}

func (c *ChainConfig) SetMaxCodeSize(n *uint64) error {
	return internal.GlobalConfigurator().SetMaxCodeSize(n)
}

func (c *ChainConfig) GetEIP7Transition() *uint64 {
	return bigNewU64(c.HomesteadBlock)
}

func (c *ChainConfig) SetEIP7Transition(n *uint64) error {
	c.HomesteadBlock = setBig(c.HomesteadBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP150Transition() *uint64 {
	return bigNewU64(c.EIP150Block)
}

func (c *ChainConfig) SetEIP150Transition(n *uint64) error {
	c.EIP150Block = setBig(c.EIP150Block, n)
	return nil
}

func (c *ChainConfig) GetEIP152Transition() *uint64 {
	return bigNewU64(c.IstanbulBlock)
}

func (c *ChainConfig) SetEIP152Transition(n *uint64) error {
	c.IstanbulBlock = setBig(c.IstanbulBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP160Transition() *uint64 {
	if c.EIP160Block != nil {
		return bigNewU64(c.EIP160Block)
	}
	return bigNewU64(c.EIP158Block)
}

func (c *ChainConfig) SetEIP160Transition(n *uint64) error {
	c.EIP160Block = setBig(c.EIP160Block, n)
	return nil
}

// GetEIP161dTransition gets the EIP161d transition.
// The configurator interface only supports single-event OFF-ON switches,
// making the support of OFF-ON-OFF-ON switches, as multi-geth does, impossible.
// The logic below _attempts_ to provide a reasonable inference, but
// should not be trusted blindly, if at all.
// At the time of writing, this configuration option at multi-geth is in
// use exclusively for the Ellaism network.
/*
https://github.com/multi-geth/multi-geth/blob/954b47d05dca919eaa15993b159ed90abcd8f071/params/config.go#L570
func (c *ChainConfig) IsEIP161(num *big.Int) bool {
	if isForked(c.EIP161ReenableBlock, num) {
		return true
	}

	if isForked(c.EIP161DisableBlock, num) {
		return false
	}

	if isForked(c.ByzantiumBlock, num) {
		return true
	}

	return false
}
*/
func (c *ChainConfig) GetEIP161dTransition() *uint64 {
	if c.EIP161DisableBlock != nil {
		if c.EIP161ReenableBlock != nil {
			return bigNewU64(c.EIP161ReenableBlock)
		}
		return nil
	}
	return bigNewU64(c.EIP158Block)
}

func (c *ChainConfig) SetEIP161dTransition(n *uint64) error {
	c.EIP158Block = setBig(c.EIP158Block, n)
	return nil
}

func (c *ChainConfig) GetEIP161abcTransition() *uint64 {
	if c.EIP161DisableBlock != nil {
		if c.EIP161ReenableBlock != nil {
			return bigNewU64(c.EIP161ReenableBlock)
		}
		return nil
	}
	return bigNewU64(c.EIP158Block)
}

func (c *ChainConfig) SetEIP161abcTransition(n *uint64) error {
	c.EIP158Block = setBig(c.EIP158Block, n)
	return nil
}

func (c *ChainConfig) GetEIP170Transition() *uint64 {
	return bigNewU64(c.EIP158Block)
}

func (c *ChainConfig) SetEIP170Transition(n *uint64) error {
	c.EIP158Block = setBig(c.EIP158Block, n)
	return nil
}

func (c *ChainConfig) GetEIP155Transition() *uint64 {
	return bigNewU64(c.EIP155Block)
}

func (c *ChainConfig) SetEIP155Transition(n *uint64) error {
	c.EIP155Block = setBig(c.EIP155Block, n)
	return nil
}

func (c *ChainConfig) GetEIP140Transition() *uint64 {
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEIP140Transition(n *uint64) error {
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP198Transition() *uint64 {
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEIP198Transition(n *uint64) error {
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP211Transition() *uint64 {
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEIP211Transition(n *uint64) error {
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP212Transition() *uint64 {
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEIP212Transition(n *uint64) error {
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP213Transition() *uint64 {
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEIP213Transition(n *uint64) error {
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP214Transition() *uint64 {
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEIP214Transition(n *uint64) error {
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP658Transition() *uint64 {
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEIP658Transition(n *uint64) error {
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP145Transition() *uint64 {
	return bigNewU64(c.ConstantinopleBlock)
}

func (c *ChainConfig) SetEIP145Transition(n *uint64) error {
	c.ConstantinopleBlock = setBig(c.ConstantinopleBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP1014Transition() *uint64 {
	return bigNewU64(c.ConstantinopleBlock)
}

func (c *ChainConfig) SetEIP1014Transition(n *uint64) error {
	c.ConstantinopleBlock = setBig(c.ConstantinopleBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP1052Transition() *uint64 {
	return bigNewU64(c.ConstantinopleBlock)
}

func (c *ChainConfig) SetEIP1052Transition(n *uint64) error {
	c.ConstantinopleBlock = setBig(c.ConstantinopleBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP1283Transition() *uint64 {
	if c.ConstantinopleBlock != nil && c.PetersburgBlock != nil {
		if c.ConstantinopleBlock.Cmp(c.PetersburgBlock) == 0 {
			return nil
		}
	}
	return bigNewU64(c.ConstantinopleBlock)
}

func (c *ChainConfig) SetEIP1283Transition(n *uint64) error {
	c.ConstantinopleBlock = setBig(c.ConstantinopleBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP1283DisableTransition() *uint64 {
	if c.ConstantinopleBlock != nil && c.PetersburgBlock != nil {
		if c.ConstantinopleBlock.Cmp(c.PetersburgBlock) == 0 {
			return nil
		}
	}
	return bigNewU64(c.PetersburgBlock)
}

func (c *ChainConfig) SetEIP1283DisableTransition(n *uint64) error {
	c.PetersburgBlock = setBig(c.PetersburgBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP1108Transition() *uint64 {
	return bigNewU64(c.IstanbulBlock)
}

func (c *ChainConfig) SetEIP1108Transition(n *uint64) error {
	c.IstanbulBlock = setBig(c.IstanbulBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP2200Transition() *uint64 {
	return bigNewU64(c.IstanbulBlock)
}

func (c *ChainConfig) SetEIP2200Transition(n *uint64) error {
	c.IstanbulBlock = setBig(c.IstanbulBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP2200DisableTransition() *uint64 {
	return nil
}

func (c *ChainConfig) SetEIP2200DisableTransition(n *uint64) error {
	if n == nil {
		return nil
	}
	return ctypes.ErrUnsupportedConfigFatal
}

func (c *ChainConfig) GetEIP1344Transition() *uint64 {
	return bigNewU64(c.IstanbulBlock)
}

func (c *ChainConfig) SetEIP1344Transition(n *uint64) error {
	c.IstanbulBlock = setBig(c.IstanbulBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP1884Transition() *uint64 {
	return bigNewU64(c.IstanbulBlock)
}

func (c *ChainConfig) SetEIP1884Transition(n *uint64) error {
	c.IstanbulBlock = setBig(c.IstanbulBlock, n)
	return nil
}

func (c *ChainConfig) GetEIP2028Transition() *uint64 {
	return bigNewU64(c.IstanbulBlock)
}

func (c *ChainConfig) SetEIP2028Transition(n *uint64) error {
	c.IstanbulBlock = setBig(c.IstanbulBlock, n)
	return nil
}

func (c *ChainConfig) GetECIP1080Transition() *uint64 {
	return nil
}

func (c *ChainConfig) SetECIP1080Transition(n *uint64) error {
	if n == nil {
		return nil
	}
	return ctypes.ErrUnsupportedConfigFatal
}

func (c *ChainConfig) GetEIP1706Transition() *uint64 {
	return nil
}

func (c *ChainConfig) SetEIP1706Transition(n *uint64) error {
	if n == nil {
		return nil
	}
	return ctypes.ErrUnsupportedConfigFatal
}

func (c *ChainConfig) GetEIP2537Transition() *uint64 {
	return nil
}

func (c *ChainConfig) SetEIP2537Transition(n *uint64) error {
	if n == nil {
		return nil
	}
	return ctypes.ErrUnsupportedConfigFatal
}

func (c *ChainConfig) GetECBP1100Transition() *uint64 {
	return nil
}

func (c *ChainConfig) SetECBP1100Transition(n *uint64) error {
	if n == nil {
		return nil
	}
	return ctypes.ErrUnsupportedConfigFatal
}

func (c *ChainConfig) GetEIP2315Transition() *uint64 {
	return bigNewU64(c.YoloV2Block)
}

func (c *ChainConfig) SetEIP2315Transition(n *uint64) error {
	c.YoloV2Block = setBig(c.YoloV2Block, n)
	return nil
}

func (c *ChainConfig) GetEIP2929Transition() *uint64 {
	return bigNewU64(c.YoloV2Block)
}

func (c *ChainConfig) SetEIP2929Transition(n *uint64) error {
	c.YoloV2Block = setBig(c.YoloV2Block, n)
	return nil
}

func (c *ChainConfig) IsEnabled(fn func() *uint64, n *big.Int) bool {
	f := fn()
	if f == nil || n == nil {
		return false
	}
	return big.NewInt(int64(*f)).Cmp(n) <= 0
}

func (c *ChainConfig) GetForkCanonHash(n uint64) common.Hash {
	if c.EIP150Block != nil && c.EIP150Block.Uint64() == n {
		return c.EIP150Hash
	}
	return common.Hash{}
}

func (c *ChainConfig) SetForkCanonHash(n uint64, h common.Hash) error {
	if c.GetEIP150Transition() != nil && *c.GetEIP150Transition() == n {
		c.EIP150Hash = h
		return nil
	}
	return ctypes.ErrUnsupportedConfigNoop
}

func (c *ChainConfig) GetForkCanonHashes() map[uint64]common.Hash {
	if c.EIP150Block == nil || c.EIP150Hash == (common.Hash{}) {
		return nil
	}
	return map[uint64]common.Hash{
		c.EIP150Block.Uint64(): c.EIP150Hash,
	}
}

func (c *ChainConfig) GetConsensusEngineType() ctypes.ConsensusEngineT {
	if c.Clique != nil {
		return ctypes.ConsensusEngineT_Clique
	}
	return ctypes.ConsensusEngineT_Ethash
}

func (c *ChainConfig) MustSetConsensusEngineType(t ctypes.ConsensusEngineT) error {
	switch t {
	case ctypes.ConsensusEngineT_Ethash:
		c.Ethash = new(ctypes.EthashConfig)
		c.Clique = nil
		return nil
	case ctypes.ConsensusEngineT_Clique:
		c.Clique = new(ctypes.CliqueConfig)
		c.Ethash = nil
		return nil
	default:
		return ctypes.ErrUnsupportedConfigFatal
	}
}

func (c *ChainConfig) GetEthashMinimumDifficulty() *big.Int {
	return internal.GlobalConfigurator().GetEthashMinimumDifficulty()
}

func (c *ChainConfig) SetEthashMinimumDifficulty(i *big.Int) error {
	return internal.GlobalConfigurator().SetEthashMinimumDifficulty(i)
}

func (c *ChainConfig) GetEthashDifficultyBoundDivisor() *big.Int {
	return internal.GlobalConfigurator().GetEthashDifficultyBoundDivisor()
}

func (c *ChainConfig) SetEthashDifficultyBoundDivisor(i *big.Int) error {
	return internal.GlobalConfigurator().SetEthashDifficultyBoundDivisor(i)
}

func (c *ChainConfig) GetEthashDurationLimit() *big.Int {
	return internal.GlobalConfigurator().GetEthashDurationLimit()
}

func (c *ChainConfig) SetEthashDurationLimit(i *big.Int) error {
	return internal.GlobalConfigurator().SetEthashDurationLimit(i)
}

// NOTE: Checking for if c.Ethash == nil is a consideration.
// If set, settings are strictly enforced, and can avoid misconfiguration.
// If not, settings are more lenient, and allow for more shorthand testing.
// For the current implementation I have chosen to USE the nil check
// for Set_ methods, and to abstain for Get_ methods.
// This allows for shorthand-initialized structs, eg. for testing,
// but refuses un-strict Conversion methods.

func (c *ChainConfig) GetEthashHomesteadTransition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	return bigNewU64(c.HomesteadBlock)
}

func (c *ChainConfig) SetEthashHomesteadTransition(i *uint64) error {
	c.HomesteadBlock = setBig(c.HomesteadBlock, i)
	return nil
}

func (c *ChainConfig) GetEIP2Transition() *uint64 {
	return bigNewU64(c.HomesteadBlock)
}

func (c *ChainConfig) SetEIP2Transition(i *uint64) error {
	c.HomesteadBlock = setBig(c.HomesteadBlock, i)
	return nil
}

func (c *ChainConfig) GetEthashEIP779Transition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	if !c.DAOForkSupport {
		return nil
	}
	return bigNewU64(c.DAOForkBlock)
}

func (c *ChainConfig) SetEthashEIP779Transition(n *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}

	if n == nil {
		c.DAOForkSupport = false
	} else {
		c.DAOForkSupport = true
	}
	c.DAOForkBlock = setBig(c.DAOForkBlock, n)

	return nil
}

func (c *ChainConfig) GetEthashEIP649Transition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	if c.ByzantiumBlock != nil && c.DisposalBlock != nil {
		if c.DisposalBlock.Cmp(c.ByzantiumBlock) <= 0 {
			return nil
		}
	}
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEthashEIP649Transition(n *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	if n == nil {
		return nil
	}
	if c.DisposalBlock != nil && c.DisposalBlock.Uint64() <= *n {
		return nil
	}
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, n)
	return nil
}

func (c *ChainConfig) GetEthashEIP1234Transition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	if c.ConstantinopleBlock != nil && c.DisposalBlock != nil {
		if c.DisposalBlock.Cmp(c.ConstantinopleBlock) <= 0 {
			return nil
		}
	}
	return bigNewU64(c.ConstantinopleBlock)
}

func (c *ChainConfig) SetEthashEIP1234Transition(n *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	if n == nil {
		return nil
	}
	if c.DisposalBlock != nil && c.DisposalBlock.Uint64() <= *n {
		return nil
	}
	c.ConstantinopleBlock = setBig(c.ConstantinopleBlock, n)
	return nil
}

// Muir Glacier difficulty bomb delay
func (c *ChainConfig) GetEthashEIP2384Transition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	return bigNewU64(c.MuirGlacierBlock)
}

func (c *ChainConfig) SetEthashEIP2384Transition(n *uint64) error {
	c.MuirGlacierBlock = setBig(c.MuirGlacierBlock, n)
	return nil
}

func (c *ChainConfig) GetEthashECIP1010PauseTransition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}

	// Disable ECIP1010 (difficulty bomb delay) if
	// concurrent with or preceded by the difficulty bomb disposal.
	// This case only happens on testnets, where these protocol changes are squashed to genesis, eg. Mordor.
	if c.DisposalBlock != nil && c.ECIP1010PauseBlock != nil {
		if c.DisposalBlock.Cmp(c.ECIP1010PauseBlock) <= 0 {
			return nil
		}
	}
	return bigNewU64(c.ECIP1010PauseBlock)
}

func (c *ChainConfig) SetEthashECIP1010PauseTransition(n *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	if c.ECIP1010PauseBlock == nil && c.ECIP1010Length != nil {
		c.ECIP1010PauseBlock = setBig(c.ECIP1010PauseBlock, n)
		c.ECIP1010Length = c.ECIP1010Length.Sub(c.ECIP1010Length, c.ECIP1010PauseBlock)
		return nil
	}
	c.ECIP1010PauseBlock = setBig(c.ECIP1010PauseBlock, n)
	return nil
}

func (c *ChainConfig) GetEthashECIP1010ContinueTransition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	if c.ECIP1010PauseBlock == nil {
		return nil
	}
	if c.ECIP1010Length == nil {
		return nil
	}

	// Disable ECIP1010 (difficulty bomb delay) if
	// concurrent with or preceded by the difficulty bomb disposal.
	// This case only happens on testnets, where these protocol changes are squashed to genesis, eg. Mordor.
	if c.DisposalBlock != nil {
		if c.DisposalBlock.Cmp(c.ECIP1010PauseBlock) <= 0 {
			return nil
		}
	}
	// transition = pause + length
	return bigNewU64(new(big.Int).Add(c.ECIP1010PauseBlock, c.ECIP1010Length))
}

func (c *ChainConfig) SetEthashECIP1010ContinueTransition(n *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	// length = continue - pause
	if n == nil {
		return ctypes.ErrUnsupportedConfigNoop
	}
	if c.ECIP1010PauseBlock == nil {
		c.ECIP1010Length = new(big.Int).SetUint64(*n)
		return nil
	}
	c.ECIP1010Length = new(big.Int).Sub(big.NewInt(int64(*n)), c.ECIP1010PauseBlock)
	return nil
}

func (c *ChainConfig) GetEthashECIP1017Transition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}

	// Multi-geth does not support transition vs. round logic,
	// so we have to assume that if the block is set that that will
	// incidentally define transition and rounds, where they are equal.
	// This is a coincidental equivalence on the ETC chain and Mordor,
	// where eg. transition=5M and rounds=5M, so activation
	// at 5M OR 0 are equivalent.

	// Manual override for Mordor test network, where the core-geth chain config
	// (https://github.com/eth-classic/mordor) specifies genesis activation with
	// 2M rounds. Multi-geth does not support this configuration, so we have to hack it in.
	if c.GetChainID() != nil && c.GetChainID().Uint64() == 63 &&
		c.ECIP1017EraBlock != nil && c.ECIP1017EraBlock.Uint64() == 2_000_000 {
		return bigNewU64(big.NewInt(0))
	}
	return bigNewU64(c.ECIP1017EraBlock)
}

func (c *ChainConfig) SetEthashECIP1017Transition(n *uint64) error {
	c.ECIP1017EraBlock = setBig(c.ECIP1017EraBlock, n)
	return nil
}

func (c *ChainConfig) GetEthashECIP1017EraRounds() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	return bigNewU64(c.ECIP1017EraBlock)
}

func (c *ChainConfig) SetEthashECIP1017EraRounds(n *uint64) error {
	c.ECIP1017EraBlock = setBig(c.ECIP1017EraBlock, n)
	return nil
}

func (c *ChainConfig) GetEthashEIP100BTransition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	return bigNewU64(c.ByzantiumBlock)
}

func (c *ChainConfig) SetEthashEIP100BTransition(i *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	c.ByzantiumBlock = setBig(c.ByzantiumBlock, i)
	return nil
}

func (c *ChainConfig) GetEthashECIP1041Transition() *uint64 {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	return bigNewU64(c.DisposalBlock)
}

func (c *ChainConfig) SetEthashECIP1041Transition(n *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	c.DisposalBlock = setBig(c.DisposalBlock, n)
	return nil
}

func (c *ChainConfig) GetEthashECIP1099Transition() *uint64 {
	return nil
}

func (c *ChainConfig) SetEthashECIP1099Transition(n *uint64) error {
	if c.Ethash == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	if n == nil {
		return nil
	}
	return ctypes.ErrUnsupportedConfigFatal
}

func (c *ChainConfig) GetEthashDifficultyBombDelaySchedule() ctypes.Uint64BigMapEncodesHex {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	return nil
}

func (c *ChainConfig) SetEthashDifficultyBombDelaySchedule(m ctypes.Uint64BigMapEncodesHex) error {
	return ctypes.ErrUnsupportedConfigNoop
}

func (c *ChainConfig) GetEthashBlockRewardSchedule() ctypes.Uint64BigMapEncodesHex {
	if c.GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return nil
	}
	return nil
}

func (c *ChainConfig) SetEthashBlockRewardSchedule(m ctypes.Uint64BigMapEncodesHex) error {
	return ctypes.ErrUnsupportedConfigNoop
}

func (c *ChainConfig) GetCliquePeriod() uint64 {
	if c.Clique == nil {
		return 0
	}
	return c.Clique.Period
}

func (c *ChainConfig) SetCliquePeriod(n uint64) error {
	if c.Clique == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	c.Clique.Period = n
	return nil
}

func (c *ChainConfig) GetCliqueEpoch() uint64 {
	if c.Clique == nil {
		return 0
	}
	return c.Clique.Epoch
}

func (c *ChainConfig) SetCliqueEpoch(n uint64) error {
	if c.Clique == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	c.Clique.Epoch = n
	return nil
}

func (c *ChainConfig) GetCliqueEIP3436Transition() *uint64 {
	if c.Clique == nil {
		return nil
	}
	return bigNewU64(c.Clique.EIP3436Transition)
}

func (c *ChainConfig) SetCliqueEIP3436Transition(n *uint64) error {
	if c.Clique == nil {
		return ctypes.ErrUnsupportedConfigFatal
	}
	c.Clique.EIP3436Transition = setBig(c.Clique.EIP3436Transition, n)
	return nil
}
