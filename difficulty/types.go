package difficulty

import (
	"math/big"
	"time"
)

// Using this package is kinda annoying. You have to do:
// TargetFromX and DifficultyFromX, and keep remembering what can
// be translated to what. It's easier to go:
// 		var target Target
//		diff := target.Difficulty()
//
// The type conversions are much easier and the exported functions are less.

type Target uint64

func (t Target) DifficultyP() Difficulty {
	return t.Difficulty(PDiff)
}

func (t Target) Difficulty(base uint64) Difficulty {
	return Difficulty(DifficultyFromTarget(t.Uint64(), base))
}

func (t Target) HashRate() float64 {
	return t.HashRateFromCustom(MiningPeriodDuration)
}

func (t Target) HashRateFromCustom(dur time.Duration) float64 {
	totalHashes := TotalHashes(t.Uint64())
	th := new(big.Float).SetInt(totalHashes)
	hashrate := th.Quo(th, big.NewFloat(dur.Seconds()))
	rate, _ := hashrate.Float64()

	return rate
}

func (t Target) Uint64() uint64 {
	return uint64(t)
}

type Difficulty float64

func (d Difficulty) Float64() float64 {
	return float64(d)
}

func (d Difficulty) TargetP(base uint64) Target {
	return d.Target(PDiff)
}

func (d Difficulty) Target(base uint64) Target {
	return Target(TargetFromDifficulty(d.Float64(), base))
}

func (d Difficulty) HashRate(base uint64) float64 {
	return d.HashRateFromCustom(MiningPeriodDuration, base)
}

func (d Difficulty) HashRateFromCustom(dur time.Duration, base uint64) float64 {
	t := d.Target(base)
	return t.HashRateFromCustom(dur)
}
