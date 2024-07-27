/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

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
