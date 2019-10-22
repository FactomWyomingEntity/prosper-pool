package sharesubmit_test

import (
	"testing"

	"github.com/FactomWyomingEntity/private-pool/sharesubmit"
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
		f := sharesubmit.ComputeEMA(v.current, v.prev, v.n)
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
