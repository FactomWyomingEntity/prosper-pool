package difficulty

import (
	"math"
	"math/big"
	"time"

	"github.com/pegnet/pegnet/modules/grader"
)

const (
	MiningPeriodDuration = 480 * time.Second
	MiningPeriodSeconds  = 480 // in seconds

	// Roughly 32,000 h/s for 2s
	PDiff uint64 = 0xffff000000000000

	// TODO: Maybe define a BDiff?
	BDiff uint64 = 0xffffffa68581fc34
)

var (
	BigMaxUint64  = new(big.Int).SetUint64(math.MaxUint64)
	BigMaxUint64F = new(big.Float).SetUint64(math.MaxUint64)

	oneF = big.NewFloat(1)
)

// TODO: Type up all equations on a pdf, not in the comments. It's ugly here.

// TotalHashes returns the estimated number of hashes done to obtain the given
// target. This is an estimate of the average case, and can be used to determine
// the estimated hashrate.
//
//	TH = 2^64 / (2^64 - Target)
//  In Python: m = np.uint64((2**64)-1); 1.0*m/(m-np.uint64(target))
//
func TotalHashes(target uint64) *big.Int {
	tF := new(big.Float).SetUint64(target)
	den := new(big.Float).Sub(BigMaxUint64F, tF)
	quo := new(big.Float).Quo(BigMaxUint64F, den)
	resI, _ := quo.Int(nil)

	return resI
}

// TargetFromHashRate returns the target given a hashrate and a duration, where
// the rate is in hashes per second.
func TargetFromHashRate(rate float64, duration time.Duration) uint64 {
	total := new(big.Float).Mul(big.NewFloat(rate), big.NewFloat(duration.Seconds()))
	i, _ := total.Int(nil)
	return TargetFromHashes(i)
}

func TargetI(totalHashes uint64) uint64 {
	return TargetFromHashes(new(big.Int).SetUint64(totalHashes))
}

// TargetFromHashes returns the expected target for a given number of hashes.
func TargetFromHashes(totalHashes *big.Int) uint64 {
	th := new(big.Float).SetInt(totalHashes)
	sub1 := new(big.Float).Sub(th, oneF)
	num := new(big.Float).Mul(BigMaxUint64F, sub1)
	quo := new(big.Float).Quo(num, th)
	target, _ := quo.Uint64()
	return target
}

// DifficultyFromTarget returns the target difficulty based off of `one`
func DifficultyFromTarget(target, one uint64) float64 {
	num := new(big.Float).SetUint64(^one)
	den := new(big.Float).SetUint64(^target)
	quo := num.Quo(num, den)
	diff, _ := quo.Float64()
	return diff
}

func TargetFromDifficulty(difficulty float64, one uint64) uint64 {
	d := big.NewFloat(difficulty)
	t := new(big.Float).Quo(new(big.Float).SetUint64(^one), d)
	target, _ := t.Uint64()
	return ^target
}

// -- TODO: Check below the line

// ExpectedMinimumTarget will report what minimum target we would expect given
// a number of hashes and a given position.
// Using https://github.com/WhoSoup/pegnet/wiki/Mining-Probabilities#expected-minimum-difficulty
func ExpectedMinimumTarget(numHashes uint64, spot int) uint64 {
	// 2^64
	space := new(big.Int).SetUint64(math.MaxUint64)
	ehrF := new(big.Int).SetUint64(numHashes)
	spotF := new(big.Int).Sub(ehrF, big.NewInt(int64(spot)))
	num := new(big.Int).Mul(space, spotF)

	den := ehrF

	expMin := new(big.Int).Quo(num, den)
	return expMin.Uint64()
}

// -----
// Taken from pegnet. Might be able to consolidate it a bit
// -----

// CalculateMinimumDifficultyFromOPRs that we should submit for a block. Uses
// the 50th (or last) graded opr for the estimation
//	Params:
//		oprs 		Sorted by difficulty, must be > 0 oprs
//		cutoff 		Is 1 based, not 0. So cutoff 50 is at index 49.
func CalculateMinimumDifficultyFromOPRs(oprs []*grader.GradingOPR, cutoff int) uint64 {
	if len(oprs) == 0 {
		return 0
	}
	var min *grader.GradingOPR
	var spot = 0
	// grab the least difficult in the top 50
	if len(oprs) >= 50 {
		// Oprs is indexed at 0, where the math (spot) is indexed at 1.
		min = oprs[49]
		spot = 50
	} else {
		min = oprs[len(oprs)-1]
		spot = len(oprs)
	}

	// minDiff is our number to create a tolerance around
	minDiff := min.SelfReportedDifficulty

	return CalculateMinimumDifficulty(spot, minDiff, cutoff)
}

// CalculateMinimumDifficulty
//		spot		The index of the difficulty param in the list. Sorted by difficulty
//		difficulty	The difficulty at index 'spot'
//		cutoff		The targeted index difficulty estimate
func CalculateMinimumDifficulty(spot int, difficulty uint64, cutoff int) uint64 {
	// Calculate the effective hash rate of the network in hashes/s
	hashrate := EffectiveHashRate(difficulty, spot, MiningPeriodSeconds)

	// Given that hashrate, aim to be above the cutoff
	floor := ExpectedMinimumDifficulty(hashrate, cutoff)
	return floor
}

// The effective hashrate of the network given the difficulty of the 50th opr
// sorted by difficulty.
// Using https://github.com/WhoSoup/pegnet/wiki/Mining-Probabilities
func EffectiveHashRate(min uint64, spot int, seconds float64) float64 {
	minF := big.NewFloat(float64(min))

	// 2^64
	space := big.NewFloat(math.MaxUint64)

	// Assume min is the 50th spot
	minSpot := big.NewFloat(float64(spot))

	num := new(big.Float).Mul(minSpot, space)
	den := new(big.Float).Sub(space, minF)

	ehr := new(big.Float).Quo(num, den)
	ehr = ehr.Quo(ehr, big.NewFloat(seconds))
	f, _ := ehr.Float64()
	return f
}

// ExpectedMinimumDifficulty will report what minimum difficulty we would expect given a hashrate for
// a given position.
// Using https://github.com/WhoSoup/pegnet/wiki/Mining-Probabilities#expected-minimum-difficulty
func ExpectedMinimumDifficulty(hashrate float64, spot int) uint64 {
	// 2^64
	space := big.NewFloat(math.MaxUint64)
	ehrF := big.NewFloat(hashrate * MiningPeriodSeconds)
	spotF := new(big.Float).Sub(ehrF, big.NewFloat(float64(spot)))
	num := new(big.Float).Mul(space, spotF)

	den := ehrF

	expMin := new(big.Float).Quo(num, den)
	f, _ := expMin.Float64()
	return uint64(f)
}
