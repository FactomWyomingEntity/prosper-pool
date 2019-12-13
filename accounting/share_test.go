package accounting_test

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/shopspring/decimal"

	. "github.com/FactomWyomingEntity/prosper-pool/accounting"
)

func TestShareMap(t *testing.T) {
	m := NewShareMap()
	m.AddShare("test-user-1", Share{Difficulty: 10})
	m.AddShare("test-user-1", Share{Difficulty: 10})
	m.AddShare("test-user-2", Share{Difficulty: 10})
	m.AddShare("test-user-3", Share{Difficulty: 10})

	if m.TotalDiff != 40 {
		t.Errorf("expect total diff of 40, found %.2f", m.TotalDiff)
	}

	if m.Sums["test-user-1"].TotalDifficulty != 20 {
		t.Errorf("expect total diff of 20, found %.2f", m.Sums["test-user-1"].TotalDifficulty)
	}

	if m.Sums["test-user-1"].TotalShares != 2 {
		t.Errorf("expect total shares of 2, found %d", m.Sums["test-user-1"].TotalShares)
	}
}

func TestPayouts_TakePoolCut(t *testing.T) {
	t.Run("vectored", func(t *testing.T) {
		type tVec struct {
			Rate      string
			Reward    int64
			Remaining int64
			Cut       int64
		}

		vecs := []tVec{
			{Rate: "0", Reward: 10 * 1e8, Remaining: 10 * 1e8, Cut: 0},             // 0%
			{Rate: "0.01", Reward: 1 * 1e8, Remaining: 1*1e8 - 1*1e6, Cut: 1e6},    // 1%
			{Rate: "0.05", Reward: 500 * 1e8, Remaining: 475 * 1e8, Cut: 25 * 1e8}, // 5%
			{Rate: "0.10", Reward: 500 * 1e8, Remaining: 450 * 1e8, Cut: 50 * 1e8}, // 10%
			{Rate: "1", Reward: 500 * 1e8, Remaining: 0, Cut: 500 * 1e8},           // 100%
		}

		for _, v := range vecs {
			r, err := decimal.NewFromString(v.Rate)
			if err != nil {
				t.Error(err)
			}
			pays := OwedPayouts{
				PoolFeeRate: r,
			}

			remain := pays.TakePoolCut(v.Reward)
			if remain != v.Remaining {
				t.Errorf("exp %d remain, found %d", v.Remaining, remain)
			}
			if pays.PoolFee != v.Cut {
				t.Errorf("exp %d cut, found %d", v.Cut, pays.PoolFee)
			}
		}
	})

	t.Run("float approximations", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			rate := TruncateTo4(rand.Float64())
			reward := rand.Int63() % (1e5 * 1e8) // 100K max
			r := decimal.NewFromFloat(rate)

			cutF := decimal.NewFromFloat(float64(reward)).Mul(r).IntPart()
			remainingF := reward - cutF

			pays := OwedPayouts{
				PoolFeeRate: r,
			}
			remainingI := pays.TakePoolCut(reward)

			if remainingF < 0 || remainingI < 0 {
				t.Errorf("less than 0 remains")
			}

			// 0.01% tolerance
			tolerance := int64(0.0001 * float64(reward))

			if abs(pays.PoolFee-cutF) > tolerance {
				t.Errorf("reward %d, rate %.2f,exp cut as %d, found %d", reward, rate, cutF, pays.PoolFee)
			}

			if abs(remainingI-remainingF) > tolerance {
				t.Errorf("reward %d, rate %.2f, exp remain as %d, found %d", reward, rate, remainingF, remainingI)
			}
		}

	})
}

func TestNewPayout(t *testing.T) {
	// Just testing the float -> int math and proportions
	t.Run("ensure props add to 100", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			users := rand.Int() % 100
			if users == 0 {
				users = 1
			}
			pays := NewPayout(Reward{
				JobID:      100,
				PoolReward: rand.Int63() % (1e6 * 1e8), // 100K max PEG
				Winning:    10,
				Graded:     15,
			}, randomRate(),
				*randomShareMap(100, users))

			var totalProp decimal.Decimal
			var totalPay int64
			for _, payouts := range pays.UserPayouts {
				totalPay += payouts.Payout
				totalProp = totalProp.Add(payouts.Proportion)
			}

			diff := pays.Reward.PoolReward - (totalPay + pays.PoolFee)
			diffP := decimal.New(diff, 0).Div(decimal.New(pays.Reward.PoolReward, 0))
			diffP = diffP.Mul(decimal.New(100, 0))
			if totalPay+pays.PoolFee != pays.Reward.PoolReward && diffP.GreaterThan(decimal.NewFromFloat(0.0001)) {
				t.Errorf("exp total payouts to be %d, found %d. %s%% off", pays.Reward.PoolReward, totalPay+pays.PoolFee, diffP.String())
			}

			// If > 1 || < 0.99990
			min, _ := decimal.NewFromString("0.99990")
			if totalProp.GreaterThan(decimal.New(1, 0)) || totalProp.LessThan(min) {
				t.Errorf("props add to %s, not %d", totalProp, 1)
			}

			// More than 1 PEG as dust is a problem
			if pays.Dust > 1e8 {
				t.Errorf("[%.8f] should have 0 dust, found %d, or %.8f PEG", float64(pays.PoolReward)/1e8, pays.Dust, float64(pays.Dust)/1e8)
			}

			dustP := decimal.New(pays.Dust, 0).Div(decimal.New(pays.Reward.PoolReward, 0))
			if dustP.GreaterThan(decimal.NewFromFloat(0.00001)) {
				t.Error("More than 0.001% in dust")
			}
		}
	})

	t.Run("test the 0 user case", func(t *testing.T) {
		// Idk how we could have 0 users, and a winner, but things should not panic
		pays := NewPayout(Reward{
			JobID:      100,
			PoolReward: rand.Int63() % (1e6 * 1e8), // 100K max PEG
			Winning:    10,
			Graded:     15,
		}, randomRate(),
			*randomShareMap(100, 0))

		if pays.PoolFee == 0 {
			t.Errorf("pool always takes a cut")
		}
		if pays.Dust+pays.PoolFee != pays.Reward.PoolReward {
			t.Errorf("dust + pool cut should equal reward")
		}
	})
}

func TestInsertTarget(t *testing.T) {
	var a [TargetsKept]uint64
	for i := 0; i < 10000; i++ {
		InsertTarget(rand.Uint64(), &a, i)
	}

	if !sort.SliceIsSorted(a, func(i, j int) bool { return a[i] > a[j] }) {
		t.Errorf("Not sorted")
	}

	if a[0] < a[19] {
		t.Errorf("not sorted")
	}
}

func randomShareMap(jobid int32, users int) *ShareMap {
	s := NewShareMap()
	for i := 0; i < users; i++ {
		buf := make([]byte, 8)
		rand.Read(buf)
		s.AddShare(fmt.Sprintf("%x", buf), Share{
			JobID:      jobid,
			Difficulty: rand.Float64() * 20,
			Accepted:   false,
			MinerID:    fmt.Sprintf("%x", buf),
			UserID:     fmt.Sprintf("%x", buf),
		})
	}

	return s
}

func randomRate() decimal.Decimal {
	return decimal.NewFromFloat(rand.Float64())
}

func abs(v int64) int64 {
	if v < 0 {
		return v * -1
	}
	return v
}
