package core

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
)

// errReorgFinality represents an error caused by artificial finality mechanisms.
var errReorgFinality = errors.New("finality-enforced invalid new chain")

// ArtificialFinalityNoDisable overrides toggling of AF features, forcing it on.
// n  = 1 : ON
// n != 1 : OFF
func (bc *BlockChain) ArtificialFinalityNoDisable(n int32) {
	log.Warn("Deactivating ECBP1100 (MESS) disablers", "always on", true)
	bc.artificialFinalityNoDisable = new(int32)
	atomic.StoreInt32(bc.artificialFinalityNoDisable, n)
}

// EnableArtificialFinality enables and disable artificial finality features for the blockchain.
// Currently toggled features include:
// - ECBP1100-MESS: modified exponential subject scoring
//
// This level of activation works BELOW the chain configuration for any of the
// potential features. eg. If ECBP1100 is not activated at the chain config x block number,
// then calling bc.EnableArtificialFinality(true) will be a noop.
// The method is idempotent.
func (bc *BlockChain) EnableArtificialFinality(enable bool, logValues ...interface{}) {
	// Short circuit if AF state is enabled and nodisable=true.
	if bc.artificialFinalityNoDisable != nil && atomic.LoadInt32(bc.artificialFinalityNoDisable) == 1 &&
		bc.IsArtificialFinalityEnabled() && !enable {
		log.Warn("Preventing disable artificial finality", "enabled", true, "nodisable", true)
		return
	}

	// Store enable/disable value regardless of config activation.
	var statusLog string
	if enable {
		statusLog = "Enabled"
		atomic.StoreInt32(&bc.artificialFinalityEnabledStatus, 1)
	} else {
		statusLog = "Disabled"
		atomic.StoreInt32(&bc.artificialFinalityEnabledStatus, 0)
	}
	if !bc.chainConfig.IsEnabled(bc.chainConfig.GetECBP1100Transition, bc.CurrentHeader().Number) {
		// Don't log anything if the config hasn't enabled it yet.
		return
	}
	logFn := log.Warn // Deactivated and enabled
	if enable {
		logFn = log.Info // Activated and enabled
	}
	logFn(fmt.Sprintf("%s artificial finality features", statusLog), logValues...)
}

// IsArtificialFinalityEnabled returns the status of the blockchain's artificial
// finality feature setting.
// This status is agnostic of feature activation by chain configuration.
func (bc *BlockChain) IsArtificialFinalityEnabled() bool {
	if bc.Config().GetConsensusEngineType() != ctypes.ConsensusEngineT_Ethash {
		return false
	}
	return atomic.LoadInt32(&bc.artificialFinalityEnabledStatus) == 1
}

// getTDRatio is a helper function returning the total difficulty ratio of
// proposed over current chain segments.
func (bc *BlockChain) getTDRatio(commonAncestor, current, proposed *types.Header) float64 {
	// Get the total difficulty ratio of the proposed chain segment over the existing one.
	commonAncestorTD := bc.GetTd(commonAncestor.Hash(), commonAncestor.Number.Uint64())

	proposedParentTD := bc.GetTd(proposed.ParentHash, proposed.Number.Uint64()-1)
	proposedTD := new(big.Int).Add(proposed.Difficulty, proposedParentTD)

	localTD := bc.GetTd(current.Hash(), current.Number.Uint64())

	tdRatio, _ := new(big.Float).Quo(
		new(big.Float).SetInt(new(big.Int).Sub(proposedTD, commonAncestorTD)),
		new(big.Float).SetInt(new(big.Int).Sub(localTD, commonAncestorTD)),
	).Float64()
	return tdRatio
}

// ecbp1100 implements the "MESS" artificial finality mechanism
// "Modified Exponential Subjective Scoring" used to prefer known chain segments
// over later-to-come counterparts, especially proposed segments stretching far into the past.
func (bc *BlockChain) ecbp1100(commonAncestor, current, proposed *types.Header) error {

	// Get the total difficulties of the proposed chain segment and the existing one.
	commonAncestorTD := bc.GetTd(commonAncestor.Hash(), commonAncestor.Number.Uint64())
	proposedParentTD := bc.GetTd(proposed.ParentHash, proposed.Number.Uint64()-1)
	proposedTD := new(big.Int).Add(proposed.Difficulty, proposedParentTD)
	localTD := bc.GetTd(current.Hash(), current.Number.Uint64())

	// if proposed_subchain_td * CURVE_FUNCTION_DENOMINATOR < get_curve_function_numerator(proposed.Time - commonAncestor.Time) * local_subchain_td.
	proposedSubchainTD := new(big.Int).Sub(proposedTD, commonAncestorTD)
	localSubchainTD := new(big.Int).Sub(localTD, commonAncestorTD)

	xBig := big.NewInt(int64(current.Time - commonAncestor.Time))
	eq := ecbp1100PolynomialV(xBig)
	want := eq.Mul(eq, localSubchainTD)

	got := new(big.Int).Mul(proposedSubchainTD, ecbp1100PolynomialVCurveFunctionDenominator)

	if got.Cmp(want) < 0 {
		prettyRatio, _ := new(big.Float).Quo(
			new(big.Float).SetInt(got),
			new(big.Float).SetInt(want),
		).Float64()
		return fmt.Errorf(`%w: ECBP1100-MESS 🔒 status=rejected age=%v current.span=%v proposed.span=%v tdr/gravity=%0.6f common.bno=%d common.hash=%s current.bno=%d current.hash=%s proposed.bno=%d proposed.hash=%s`,
			errReorgFinality,
			common.PrettyAge(time.Unix(int64(commonAncestor.Time), 0)),
			common.PrettyDuration(time.Duration(current.Time-commonAncestor.Time)*time.Second),
			common.PrettyDuration(time.Duration(int32(xBig.Uint64()))*time.Second),
			prettyRatio,
			commonAncestor.Number.Uint64(), commonAncestor.Hash().Hex(),
			current.Number.Uint64(), current.Hash().Hex(),
			proposed.Number.Uint64(), proposed.Hash().Hex(),
		)
	}
	return nil
}

/*
ecbp1100PolynomialV is a cubic function that looks a lot like Option 3's sin function,
but adds the benefit that the calculation can be done with integers (instead of yucky floating points).
> https://github.com/ethereumclassic/ECIPs/issues/374#issuecomment-694156719

CURVE_FUNCTION_DENOMINATOR = 128

def get_curve_function_numerator(time_delta: int) -> int:
    xcap = 25132 # = floor(8000*pi)
    ampl = 15
    height = CURVE_FUNCTION_DENOMINATOR * (ampl * 2)
    if x > xcap:
        x = xcap
    # The sine approximator `y = 3*x**2 - 2*x**3` rescaled to the desired height and width
    return CURVE_FUNCTION_DENOMINATOR + (3 * x**2 - 2 * x**3 // xcap) * height // xcap ** 2


The if tdRatio < antiGravity check would then be

if proposed_subchain_td * CURVE_FUNCTION_DENOMINATOR < get_curve_function_numerator(current.Time - commonAncestor.Time) * local_subchain_td.
*/
func ecbp1100PolynomialV(x *big.Int) *big.Int {

	// Make a copy; do not mutate argument value.

	// if x > xcap:
	//    x = xcap
	xA := new(big.Int).Set(x)
	if xA.Cmp(ecbp1100PolynomialVXCap) > 0 {
		xA.Set(ecbp1100PolynomialVXCap)
	}

	xB := new(big.Int).Set(x)
	if xB.Cmp(ecbp1100PolynomialVXCap) > 0 {
		xB.Set(ecbp1100PolynomialVXCap)
	}

	out := big.NewInt(0)

	// 3 * x**2
	xA.Exp(xA, big2, nil)
	xA.Mul(xA, big3)

	// 3 * x**2 // xcap
	xB.Exp(xB, big3, nil)
	xB.Mul(xB, big2)
	xB.Div(xB, ecbp1100PolynomialVXCap)

	// (3 * x**2 - 2 * x**3 // xcap)
	out.Sub(xA, xB)

	// // (3 * x**2 - 2 * x**3 // xcap) * height
	out.Mul(out, ecbp1100PolynomialVHeight)

	// xcap ** 2
	xcap2 := new(big.Int).Exp(ecbp1100PolynomialVXCap, big2, nil)

	// (3 * x**2 - 2 * x**3 // xcap) * height // xcap ** 2
	out.Div(out, xcap2)

	// CURVE_FUNCTION_DENOMINATOR + (3 * x**2 - 2 * x**3 // xcap) * height // xcap ** 2
	out.Add(out, ecbp1100PolynomialVCurveFunctionDenominator)
	return out
}

var big2 = big.NewInt(2)
var big3 = big.NewInt(3)

// ecbp1100PolynomialVCurveFunctionDenominator
// CURVE_FUNCTION_DENOMINATOR = 128
var ecbp1100PolynomialVCurveFunctionDenominator = big.NewInt(128)

// ecbp1100PolynomialVXCap
// xcap = 25132 # = floor(8000*pi)
var ecbp1100PolynomialVXCap = big.NewInt(25132)

// ecbp1100PolynomialVAmpl
// ampl = 15
var ecbp1100PolynomialVAmpl = big.NewInt(15)

// ecbp1100PolynomialVHeight
// height = CURVE_FUNCTION_DENOMINATOR * (ampl * 2)
var ecbp1100PolynomialVHeight = new(big.Int).Mul(new(big.Int).Mul(ecbp1100PolynomialVCurveFunctionDenominator, ecbp1100PolynomialVAmpl), big2)

/*
ecbp1100AGSinusoidalA is a sinusoidal function.

OPTION 3: Yet slower takeoff, yet steeper eventual ascent. Has a differentiable ceiling transition.
h(x)=15 sin((x+12000 π)/(8000))+15+1

*/
func ecbp1100AGSinusoidalA(x float64) (antiGravity float64) {
	ampl := float64(15)   // amplitude
	pDiv := float64(8000) // period divisor
	phaseShift := math.Pi * (pDiv * 1.5)
	peakX := math.Pi * pDiv // x value of first sin peak where x > 0
	if x > peakX {
		// Cause the x value to limit to the x value of the first peak of the sin wave (ceiling).
		x = peakX
	}
	return (ampl * math.Sin((x+phaseShift)/pDiv)) + ampl + 1
}

/*
ecbp1100AGExpB is an exponential function with x as a base (and rationalized exponent).

OPTION 2: Slightly slower takeoff, steeper eventual ascent
g(x)=x^(x*0.00002)
*/
func ecbp1100AGExpB(x float64) (antiGravity float64) {
	return math.Pow(x, x*0.00002)
}

/*
ecbp1100AGExpA is an exponential function with x as exponent.

This was (one of?) Vitalik's "original" specs:
> 1.0001 ** (number of seconds between when S1 was received and when S2 was received)
- https://bitcointalk.org/index.php?topic=865169.msg16349234#msg16349234
> gravity(B') = gravity(B) * 0.99 ^ n
- https://blog.ethereum.org/2014/11/25/proof-stake-learned-love-weak-subjectivity/

OPTION 1 (Original ESS)
f(x)=1.0001^(x)
*/
func ecbp1100AGExpA(x float64) (antiGravity float64) {
	return math.Pow(1.0001, x)
}
