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

func (t Target) Difficulty() Difficulty {
	return Difficulty(DifficultyFromTarget(t.Uint64(), PDiff))
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

func (d Difficulty) Target() Target {
	return Target(TargetFromDifficulty(d.Float64(), PDiff))
}

func (d Difficulty) HashRate() float64 {
	return d.HashRateFromCustom(MiningPeriodDuration)
}

func (d Difficulty) HashRateFromCustom(dur time.Duration) float64 {
	t := d.Target()
	return t.HashRateFromCustom(dur)
}
