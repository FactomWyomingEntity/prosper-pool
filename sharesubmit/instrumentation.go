package sharesubmit

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	emaDifficulty = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_submit_difficulty_cutoff_ema_diff",
		Help: "Cutoff minimum difficulty",
	})
	cutoffMinimumDifficulty = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_submit_difficulty_cutoff_min_diff",
		Help: "Cutoff minimum difficulty",
	})
	cutoffMinimumIndex = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_submit_difficulty_cutoff_min_index",
		Help: "Cutoff minimum index",
	})
	lastGradedDifficulty = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_submit_difficulty_last_graded_diff",
		Help: "Last graded difficulty",
	})
	lastGradedIndex = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_submit_difficulty_last_graded_index",
		Help: "Last graded index",
	})
)

var prom sync.Once

func RegisterPrometheus() {
	prom.Do(func() {
		prometheus.MustRegister(lastGradedDifficulty)
		prometheus.MustRegister(lastGradedIndex)
		prometheus.MustRegister(cutoffMinimumIndex)
		prometheus.MustRegister(cutoffMinimumDifficulty)
		prometheus.MustRegister(emaDifficulty)
	})
}
