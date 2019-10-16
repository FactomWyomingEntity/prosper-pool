package pegnet

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pegnetSyncHeight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_pegnet_sync_currentheight",
		Help: "Current synced height of the internal pegnet daemon",
	})
)

var prom sync.Once

func RegisterPrometheus() {
	prom.Do(func() {
		prometheus.MustRegister(pegnetSyncHeight)
	})
}
