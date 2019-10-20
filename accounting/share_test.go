package accounting_test

import (
	"fmt"
	"math/rand"
	"testing"

	. "github.com/FactomWyomingEntity/private-pool/accounting"
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
			Rate      int64
			Reward    int64
			Remaining int64
			Cut       int64
		}

		vecs := []tVec{
			{Rate: 0, Reward: 10 * 1e8, Remaining: 10 * 1e8, Cut: 0},             // 0%
			{Rate: 100, Reward: 1 * 1e8, Remaining: 1*1e8 - 1*1e6, Cut: 1e6},     // 1%
			{Rate: 500, Reward: 500 * 1e8, Remaining: 475 * 1e8, Cut: 25 * 1e8},  // 5%
			{Rate: 1000, Reward: 500 * 1e8, Remaining: 450 * 1e8, Cut: 50 * 1e8}, // 10%
			{Rate: 10000, Reward: 500 * 1e8, Remaining: 0, Cut: 500 * 1e8},       // 100%
		}

		for _, v := range vecs {
			pays := Payouts{
				PoolFeeRate: v.Rate,
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
			cutF := int64(float64(reward) * rate)
			remainingF := reward - cutF

			pays := Payouts{
				PoolFeeRate: int64(1e4 * rate),
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
				JobID:      "test",
				PoolReward: rand.Int63() % (1e6 * 1e8), // 100K max PEG
				Winning:    10,
				Graded:     15,
			}, randomRate(),
				*randomShareMap("test", users))

			var totalProp int64
			var totalPay int64
			for _, payouts := range pays.UserPayouts {
				totalPay += payouts.Payout
				totalProp += payouts.Proportion
			}

			if totalPay+pays.PoolFee != pays.Reward.PoolReward {
				t.Errorf("exp total payouts to be %d, found %d", pays.Reward.PoolReward, totalPay+pays.PoolFee)
			}
			if totalProp != 100*100 {
				t.Errorf("props add to %d, not %d", totalProp, 100*100)
			}
			if pays.Dust != 0 {
				t.Errorf("should have 0 dust")
			}
		}
	})

	t.Run("test the 0 user case", func(t *testing.T) {
		// Idk how we could have 0 users, and a winner, but things should not panic
		pays := NewPayout(Reward{
			JobID:      "test",
			PoolReward: rand.Int63() % (1e6 * 1e8), // 100K max PEG
			Winning:    10,
			Graded:     15,
		}, randomRate(),
			*randomShareMap("test", 0))

		if pays.PoolFee == 0 {
			t.Errorf("pool always takes a cut")
		}
		if pays.Dust+pays.PoolFee != pays.Reward.PoolReward {
			t.Errorf("dust + pool cut should equal reward")
		}
	})
}

func randomShareMap(jobid string, users int) *ShareMap {
	s := NewShareMap()
	for i := 0; i < users; i++ {
		buf := make([]byte, 8)
		s.AddShare(fmt.Sprintf("%x", buf), Share{
			JobID:      jobid,
			Difficulty: rand.Float64() * 100,
			Accepted:   false,
			MinerID:    fmt.Sprintf("%x", buf),
			UserID:     fmt.Sprintf("%x", buf),
		})
	}

	return s
}

func randomRate() int64 {
	rate := TruncateTo4(rand.Float64())
	return int64(10000 * rate)
}

func abs(v int64) int64 {
	if v < 0 {
		return v * -1
	}
	return v
}
