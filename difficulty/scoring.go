package difficulty

import (
	"math"
	"time"
)

const (
	lamda = 1200.0
)

func Score(since time.Duration, shares int) float64 {
	return math.Exp(since.Seconds() / lamda)
}

func HashRateScore(score float64) float64 {
	n := math.Pow(2, 32)
	return n / lamda * score
}

// HashRateFromDifficulty might be quicker than the other method of going
// diff -> target -> hashes
func HashRateFromDifficulty(diff int, target uint64) uint64 {
	var m uint64 = math.MaxUint64
	rat := ^(^target / uint64(diff))
	return m / (rat * m)
}
