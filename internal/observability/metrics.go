package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	GaugeVMSyncing = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "virtual_machine_syncing_vms",
			Help: "Number of virtual machines currently syncing",
		},
	)
)
