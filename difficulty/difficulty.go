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
	pDiff uint64 = 0xffff00000000000
	bDiff uint64 = 0xffffffa68581fc34
)

// ExpectedTarget returns the expected target difficulty a hashrate can obtain
// in a given amount of time.
// Math taken from: https://github.com/WhoSoup/pegnet/wiki/Mining-Probabilities#expected-minimum-difficulty
// Steps follow: https://github.com/WhoSoup/pegnet/wiki/Equations-with-Steps#minimum-difficulty
//		Replace 50 with 1, as we want our best hash to be this, not our 50th
// The expected target is:
//
// Target = 2^64 * (Total Hashes - 1 / Total Hashes)
//

func ExpectedTarget(hashrate uint64, duration time.Duration) uint64 {
	m := new(big.Float).SetUint64(math.MaxUint64)
	hr := new(big.Float).SetUint64(hashrate)
	th := new(big.Float).Mul(hr, big.NewFloat(duration.Seconds()))

	rat := new(big.Float).Quo(th, new(big.Float).Sub(th, big.NewFloat(1)))
	t := new(big.Float).Mul(m, rat)
	u, _ := t.Uint64()
	return u
}

// hashrate / 2^64

func Target(hashrate uint64) uint64 {
	hashPerMinute := new(big.Float).Mul(new(big.Float).SetUint64(hashrate), big.NewFloat(60))
	inv := new(big.Float).Quo(big.NewFloat(1), hashPerMinute)
	inv = inv.Sub(big.NewFloat(1), inv)
	mul := new(big.Float).Mul(new(big.Float).SetUint64(math.MaxUint64), inv)
	u, _ := mul.Uint64()
	return u
}

func BlocksPerTime(hashrate uint64, duration time.Duration) float64 {
	hr := new(big.Float).SetUint64(hashrate)
	dur := big.NewFloat(duration.Seconds())

	num := new(big.Float).Mul(hr, dur)
	res := num.Mul(num, new(big.Float).SetUint64(bDiff))
	f, _ := res.Float64()
	return f
}

// Difficulty returns the difficulty of a given target in relation to pDiff
func Difficulty(target uint64) float64 {
	pDiff := new(big.Float).SetUint64(pDiff)
	fTarget := new(big.Float).SetUint64(target)

	f, _ := pDiff.Quo(pDiff, fTarget).Float64()
	return f
}

// ExpectedHashes computes the number of expected hashes to achieve a given
// target.
// Using https://github.com/WhoSoup/pegnet/wiki/Mining-Probabilities
func ExpectedHashes(min uint64, spot int) float64 {
	minF := big.NewFloat(float64(min))

	// 2^64
	space := big.NewFloat(math.MaxUint64)

	minSpot := big.NewFloat(float64(spot))

	num := new(big.Float).Mul(minSpot, space)
	den := new(big.Float).Sub(space, minF)

	ehr := new(big.Float).Quo(num, den)
	//ehr = ehr.Quo(ehr, big.NewFloat(MiningPeriod))
	f, _ := ehr.Float64()
	return f
}

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
