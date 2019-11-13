package difficulty_test

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	. "github.com/FactomWyomingEntity/prosper-pool/difficulty"
)

var _ = crand.Read

const (
	K float64 = 1000

	Minute uint64 = 60
)

//func TestExpectedMinimumTarget(t *testing.T) {
//	fmt.Printf("%x\n", TargetI(100*K))
//	fmt.Printf("%x\n", ExpectedTarget(100*K, time.Minute/30))
//
//	//fmt.Println(BlocksPerTime(100*K, time.Minute*9))
//
//	fmt.Printf("%x\n", ExpectedMinimumTarget(100*K*MiningPeriodSeconds, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*2*Minute, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*Minute, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*MiningPeriodSeconds, 1))
//	fmt.Printf("%x\n", ExpectedMinimumTarget(100*K*MiningPeriodSeconds, 1))
//}

func TestPDiffProperties(t *testing.T) {
	total := TotalHashes(PDiff)
	tF := new(big.Float).SetInt(total)

	stat := func(dur time.Duration) {
		fmt.Printf("%.2f h/s for %s\n",
			new(big.Float).Quo(tF, big.NewFloat(dur.Seconds())), dur)
	}
	stat(time.Second)
	stat(time.Second * 2)
	stat(time.Second * 5)
	stat(time.Minute)
}

func TestDifficulty(t *testing.T) {
	t.Run("hashrate doubling using estimates", func(t *testing.T) {
		amt := uint64(40)
		start := uint64(10) // First few have too much error
		d := make([]float64, amt)
		for i := start; i < amt; i++ {
			target := TargetI(uint64(math.Pow(float64(2), float64(i))))
			d[i] = DifficultyFromTarget(target, PDiff)
		}

		for i := start + 1; i < amt; i++ {
			if d[i]/d[i-1] != 2 {
				t.Errorf("expect ratio of 2, found %f", d[i]/d[i-1])
			}
		}
	})

	// This is a bit all over the place
	t.Run("hashrate doubling using rand generator", func(t *testing.T) {
		hashrate := 25 * K
		dur := time.Second * 50
		amt := 5

		d := make([]float64, amt)

		for i := float64(0); i < float64(amt); i++ {
			hr := math.Pow(float64(2), float64(i)) * hashrate
			fmt.Println(hr)
			d[int(i)] = DifficultyFromTarget(bestHash(hr, dur), PDiff)
		}

		for i := 1; i < amt; i++ {
			if d[i]/d[i-1] != 2 {
				t.Errorf("expect ratio of 2, found %f", d[i]/d[i-1])
			}
		}
	})
}

func TestTargetFromDifficulty(t *testing.T) {
	t.Run("test one", func(t *testing.T) {
		if TargetFromDifficulty(1, PDiff) != PDiff {
			t.Error("diff 1 should be pDiff")
		}
	})

	t.Run("test random", func(t *testing.T) {
		testDiff := func(from uint64) {
			for i := 0; i < 1000; i++ {
				tar := rand.Uint64() | from
				d := DifficultyFromTarget(tar, PDiff)
				nT := TargetFromDifficulty(d, PDiff)
				diff := nT - tar
				if tar > nT {
					diff = tar - nT
				}

				if nT != tar && diff > 1 {
					t.Errorf("exp %d, found %d. Diff %d", tar, nT, diff)
				}
			}
		}
		// From pDiff
		testDiff(PDiff)
		// From higer
		testDiff(0xffffff0000000000)
	})
}

func TestTotalHashes(t *testing.T) {
	tar := Target(18446490372542540769)
	fmt.Println(tar.HashRateFromCustom(time.Second*2 + time.Millisecond*680))
}

func bestHash(hashrate float64, duration time.Duration) uint64 {
	var best uint64
	for i := float64(0); i < hashrate*duration.Seconds(); i++ {
		x := make([]byte, 8)
		_, _ = crand.Read(x)
		target := binary.BigEndian.Uint64(x)
		//target := rand.Uint64()
		if target > best {
			best = target
		}
	}
	return best
}
