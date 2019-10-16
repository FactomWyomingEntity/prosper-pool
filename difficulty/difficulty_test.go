package difficulty_test

import (
	"math"
	"testing"

	. "github.com/FactomWyomingEntity/private-pool/difficulty"
)

const (
	K uint64 = 1000

	Minute uint64 = 60
)

//func TestExpectedMinimumTarget(t *testing.T) {
//	fmt.Printf("%x\n", TargetI(100*K))
//	fmt.Printf("%x\n", ExpectedTarget(100*K, time.Minute/30))
//
//	//fmt.Println(BlocksPerTime(100*K, time.Minute*9))
//
//	fmt.Printf("%x\n", ExpectedMinimumTarget(100*K*MiningPeriod, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*2*Minute, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*Minute, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*MiningPeriod, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(100*K*MiningPeriod, 1))
//}

func TestDifficulty(t *testing.T) {
	t.Run("hashrate doubling", func(t *testing.T) {
		amt := uint64(40)
		start := uint64(10) // First few have too much error
		d := make([]float64, amt)
		for i := start; i < amt; i++ {
			target := TargetI(uint64(math.Pow(float64(2), float64(i))))
			d[i] = Difficulty(target, PDiff)
		}

		for i := start + 1; i < amt; i++ {
			if d[i]/d[i-1] != 2 {
				t.Errorf("expect ratio of 2, found %f", d[i]/d[i-1])
			}
		}
	})
}

func TestTotalHashes(t *testing.T) {

}
