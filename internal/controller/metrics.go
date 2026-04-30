// Copyright 2026 Red Hat Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	BuildsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bob",
			Name:      "builds_total",
			Help:      "Total number of builds started, partitioned by namespace and result.",
		},
		[]string{"namespace", "result"},
	)

	BuildDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bob",
			Name:      "build_duration_seconds",
			Help:      "Duration of builds from creation to completion.",
			Buckets:   prometheus.ExponentialBuckets(30, 2, 10), // 30s to ~8.5h
		},
		[]string{"namespace", "architecture"},
	)

	ActiveBuilds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "bob",
			Name:      "active_builds",
			Help:      "Number of currently running builds.",
		},
		[]string{"namespace"},
	)

	ToolchainBuildsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bob",
			Name:      "toolchain_builds_total",
			Help:      "Total number of toolchain image builds, partitioned by result.",
		},
		[]string{"namespace", "result"},
	)

	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bob",
			Name:      "reconcile_errors_total",
			Help:      "Total number of reconciliation errors by controller.",
		},
		[]string{"controller"},
	)

	CachePVCsManaged = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "bob",
			Name:      "cache_pvcs_managed",
			Help:      "Number of cache PVCs currently managed.",
		},
		[]string{"namespace"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		BuildsTotal,
		BuildDurationSeconds,
		ActiveBuilds,
		ToolchainBuildsTotal,
		ReconcileErrors,
		CachePVCsManaged,
	)
}
