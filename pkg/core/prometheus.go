package core

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics for monitoring service.
var (
	//blockHeight prometheus metric.
	blockHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current index of processed block",
			Name:      "current_block_height",
			Namespace: "saiya",
		},
	)
	//persistedHeight prometheus metric.
	persistedHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current persisted block count",
			Name:      "current_persisted_height",
			Namespace: "saiya",
		},
	)
	//headerHeight prometheus metric.
	headerHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current header height",
			Name:      "current_header_height",
			Namespace: "saiya",
		},
	)
)

func init() {
	prometheus.MustRegister(
		blockHeight,
		persistedHeight,
		headerHeight,
	)
}

func updatePersistedHeightMetric(pHeight uint32) {
	persistedHeight.Set(float64(pHeight))
}

func updateHeaderHeightMetric(hHeight int) {
	headerHeight.Set(float64(hHeight))
}

func updateBlockHeightMetric(bHeight uint32) {
	blockHeight.Set(float64(bHeight))
}
