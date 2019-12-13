package main

import (
	crand "crypto/rand"
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/FactomWyomingEntity/prosper-pool/accounting"
	"github.com/FactomWyomingEntity/prosper-pool/difficulty"
)

var _ = crand.Int

type Vector struct {
	TargetHashrate float64
	Duration       time.Duration
}

var Vectors = []Vector{
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
	{9000, time.Second * 60},
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var writer *csv.Writer
	filename := "data.csv"
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		writer = csv.NewWriter(file)
		panicErr(writer.Write([]string{"ID", "actual hr", "duration(s)", "avghr", "last", "weightedavg", "kept"}))
	} else {
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0777)
		panicErr(err)
		defer file.Close()
		writer = csv.NewWriter(file)
	}

	var wl sync.Mutex
	var wg sync.WaitGroup
	for i := range Vectors {
		wg.Add(1)
		go func() {
			now := time.Now()
			sums := accounting.NewShareMap()
			computeShares(sums, "key", Vectors[i].TargetHashrate, Vectors[i].Duration)
			wl.Lock()
			panicErr(writer.Write([]string{
				fmt.Sprintf("%d", i),
				fmt.Sprintf("%.2f", Vectors[i].TargetHashrate),
				fmt.Sprintf("%.2f", Vectors[i].Duration.Seconds()),
				fmt.Sprintf("%.2f", sums.Sums["key"].AverageHashrate()),
				fmt.Sprintf("%.2f", sums.Sums["key"].LastHashrate()),
				fmt.Sprintf("%.2f", sums.Sums["key"].WeightedAverageHashrate()),
				fmt.Sprintf("%d", accounting.TargetsKept),
			}))
			wl.Unlock()
			wg.Done()
			fmt.Printf("%d finished in %s\n", i, time.Since(now))
		}()
	}
	wg.Wait()
	writer.Flush()

}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func computeShares(shareMap *accounting.ShareMap, key string, hashrate float64, duration time.Duration) {
	for i := float64(0); i < hashrate*duration.Seconds(); i++ {
		//x := make([]byte, 8)
		//_, _ = crand.Read(x)
		//target := binary.BigEndian.Uint64(x)
		target := rand.Uint64()
		if target > difficulty.PDiff {
			shareMap.AddShare(key, accounting.Share{
				JobID:      0,
				Nonce:      []byte{},
				Difficulty: 0,
				Target:     target,
				Accepted:   true,
				MinerID:    "100K",
				UserID:     "100K",
			})
		}
	}
	shareMap.Sums[key].FirstShare = time.Now().Add(-1 * duration)
	shareMap.Sums[key].LastShare = shareMap.Sums[key].FirstShare.Add(duration)
}
