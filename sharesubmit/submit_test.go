package sharesubmit

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/FactomWyomingEntity/prosper-pool/difficulty"
)

func TestComputeEMA(t *testing.T) {
	type vec struct {
		prev    uint64
		current uint64
		exp     uint64
		n       int
	}

	vecs := []vec{
		{prev: 0, current: 10, exp: 10, n: 4},
		{prev: 18446024746824600000, current: 18446222197702600000, exp: 18446035419845032000, n: 36},
		{prev: 18443785558926500000, current: 18442051839271800000, exp: 18443691844350500000, n: 36},
		{prev: 21412451, current: 123, exp: 20255027, n: 36},
	}

	for _, v := range vecs {
		f := ComputeEMA(v.current, v.prev, v.n)
		var diff uint64
		if f > v.exp {
			diff = f - v.exp
		} else {
			diff = v.exp - f
		}
		pDiff := float64(diff) / float64(v.exp)
		if f != v.exp && pDiff > 0.001 {
			t.Errorf("exp %d, found %d, diff %d", v.exp, f, diff)
		}
	}
}

func TestSoftMaxSort(t *testing.T) {
	t.Run("disabled softmax", func(t *testing.T) {
		s := new(Submitter)
		s.resetJobState()
		for i := 0; i < 100; i++ {
			u := rand.Uint64()
			if !s.softMax(u) {
				t.Error("Softmax is disabled, yet it rejected a share")
			}
		}
	})

	t.Run("enabled softmax", func(t *testing.T) {
		s := new(Submitter)
		s.configuration.SoftMaxLimit = 50
		s.resetJobState()
		for i := 0; i < 5000; i++ {
			u := rand.Uint64()
			keep := u > s.jobState.diffList[len(s.jobState.diffList)-1]
			acc := s.softMax(u)
			if !acc && keep {
				t.Error("Should have kept")
			}
		}
		s.resetJobState()
		for i := 0; i < s.configuration.SoftMaxLimit; i++ {
			if !s.softMax(rand.Uint64()) {
				t.Error("should accept")
			}
		}
	})
}

func TestSoftMaxEfficiency(t *testing.T) {
	// 247/1941 :: 11950000000/12000000000 :: 99.583
	// Only submit 247 of 1941 possible submissions. Not bad for being cheap.
	t.Run("10x the net hashrate", func(t *testing.T) {
		if true {
			return // This test takes so long to run.
		}
		s := new(Submitter)
		s.configuration.SoftMaxLimit = 50
		s.resetJobState()

		netTotalHashes := uint64(1e7 * 120) // 100mh/s
		ourHashes := uint64(netTotalHashes * 10)
		total := 0
		softMaxTotal := 0

		//  A ema-ish amount
		floor := difficulty.ExpectedMinimumTarget(netTotalHashes, 200)
		for i := uint64(0); i < ourHashes; i++ {
			u := rand.Uint64()
			if u > floor {
				total++
				if s.softMax(u) {
					softMaxTotal++
				}
			}
			if i%(1e7*5) == 0 {
				fmt.Printf("%d/%d :: %d/%d :: %.3f\n", softMaxTotal, total, i, ourHashes, 100*float64(i)/float64(ourHashes))
			}
		}

		fmt.Println(total, softMaxTotal)
	})
}
