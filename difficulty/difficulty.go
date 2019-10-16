package difficulty

import (
	"math"
	"math/big"
	"time"
)

const (
	MiningPeriod = 480 // in seconds

	//             0x00000000
	//             0x00000000FFFF
	//             0xffffc813713da191
	PDiff uint64 = 0xffff00000000000
	BDiff uint64 = 0xffffffa68581fc34
)

var (
	BigMaxUint64  = new(big.Int).SetUint64(math.MaxUint64)
	BigMaxUint64F = new(big.Float).SetUint64(math.MaxUint64)

	oneF = big.NewFloat(1)
)

// TotalHashes returns the estimated number of hashes done to obtain the given
// target. This is an estimate of the average case, and can be used to determine
// the estimated hashrate.
//
//	TH = 2^64 * (1 - (1 / target))
//
func TotalHashes(target uint64) uint64 {
	inv := new(big.Float).Quo(oneF, new(big.Float).SetUint64(target))
	sub := new(big.Float).Sub(oneF, inv)
	res := new(big.Float).Mul(BigMaxUint64F, sub)
	u, _ := res.Uint64()
	return u
}

// TargetFromHashRate returns the target given a hashrate and a duration, where
// the rate is in hashes per second.
func TargetFromHashRate(rate float64, duration time.Duration) *big.Int {
	total := new(big.Float).Mul(big.NewFloat(rate), big.NewFloat(duration.Seconds()))
	i, _ := total.Int(nil)
	return i
}

func TargetI(totalHashes uint64) uint64 {
	return Target(new(big.Int).SetUint64(totalHashes))
}

// Target returns the expected target for a given number of hashes.
func Target(totalHashes *big.Int) uint64 {
	th := new(big.Float).SetInt(totalHashes)
	sub1 := new(big.Float).Sub(th, oneF)
	num := new(big.Float).Mul(BigMaxUint64F, sub1)
	quo := new(big.Float).Quo(num, th)
	target, _ := quo.Uint64()
	return target
}

// Difficulty returns the target difficulty based off of `one`
func Difficulty(target, one uint64) float64 {
	num := new(big.Float).SetUint64(^one)
	den := new(big.Float).SetUint64(^target)
	quo := num.Quo(num, den)
	diff, _ := quo.Float64()
	return diff
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
