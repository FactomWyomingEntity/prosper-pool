package mining

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

// GroupMinerStats has the stats for all miners running from a
// coordinator. It will do aggregation for simple global stats
type GroupMinerStats struct {
	Miners map[uint32]*SingleMinerStats `json:"miners"`
	JobID  int32                        `json:"jobid"`
}

func NewGroupMinerStats(job int32) *GroupMinerStats {
	g := new(GroupMinerStats)
	g.Miners = make(map[uint32]*SingleMinerStats)
	g.JobID = job

	return g
}

// TotalHashPower is the sum of all miner's hashpower
func (g *GroupMinerStats) TotalHashPower() float64 {
	var totalDur time.Duration
	var acc float64
	// Weight by duration
	for _, m := range g.Miners {
		elapsed := m.Stop.Sub(m.Start)
		totalDur += elapsed
		dur := elapsed.Seconds()
		if dur == 0 {
			continue // Divide by 0 catch
		}
		acc += float64(m.TotalHashes) / dur
	}

	return acc
}

// TotalSubmissions
func (g *GroupMinerStats) TotalSubmissions() int {
	var total int
	for _, m := range g.Miners {
		total += m.TotalSubmissions
	}

	return total
}

func (g *GroupMinerStats) AvgHashRatePerMiner() float64 {
	var totalDur time.Duration
	var acc float64
	// Weight by duration
	for _, m := range g.Miners {
		elapsed := m.Stop.Sub(m.Start)
		totalDur += elapsed
		dur := elapsed.Seconds()
		if dur == 0 {
			continue // Divide by 0 catch
		}
		acc += elapsed.Seconds() * (float64(m.TotalHashes) / dur)
	}

	dur := totalDur.Seconds()
	if dur == 0 {
		return 0 // Divide by 0 catch
	}
	return acc / totalDur.Seconds()
}

// AvgDurationPerMiner is the average duration of mining across all miners.
func (g *GroupMinerStats) AvgDurationPerMiner() time.Duration {
	if len(g.Miners) == 0 {
		return 0 // Divide by 0 catch
	}

	var totalDur time.Duration
	// Weight by duration
	for _, m := range g.Miners {
		if m.Start.IsZero() {
			continue
		}
		elapsed := m.Stop.Sub(m.Start)
		totalDur += elapsed
	}

	return totalDur / time.Duration(len(g.Miners))
}

func (g *GroupMinerStats) LogFields() log.Fields {
	f := log.Fields{
		"job":            g.JobID,
		"miners":         len(g.Miners),
		"miner_hashrate": fmt.Sprintf("%s/s", humanize.FormatFloat("", g.AvgHashRatePerMiner())),
		"total_hashrate": fmt.Sprintf("%s/s", humanize.FormatFloat("", g.TotalHashPower())),
		"avg_duration":   fmt.Sprintf("%s", g.AvgDurationPerMiner()),
		"total_submit":   fmt.Sprintf("%d", g.TotalSubmissions()),
	}

	return f
}

// SingleMinerStats is the stats of a single miner
type SingleMinerStats struct {
	ID               uint32 `json:"id"`
	TotalHashes      uint64 `json:"totalhashes"`
	TotalSubmissions int
	BestDifficulty   uint64    `json:"bestdifficulty"`
	Start            time.Time `json:"start"`
	Stop             time.Time `json:"stop"`
}

func NewSingleMinerStats(id uint32) *SingleMinerStats {
	s := new(SingleMinerStats)
	s.ID = id
	return s
}

func (s *SingleMinerStats) NewDifficulty(diff uint64) {
	if diff > s.BestDifficulty {
		s.BestDifficulty = diff
	}
}
