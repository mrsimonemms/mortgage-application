package observability

import "github.com/prometheus/client_golang/prometheus"

const (
	metricNamespace = "mortgage"
	metricSubsystem = "applications"

	labelScenario = "scenario"
	labelVersion  = "version"
	labelOutcome  = "outcome"
)

var (
	ApplicationsStartedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "started_total",
			Help:      "Total number of mortgage applications started",
		},
		[]string{labelScenario, labelVersion},
	)

	ApplicationsCompletedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "completed_total",
			Help:      "Total number of mortgage applications completed by outcome",
		},
		[]string{labelScenario, labelVersion, labelOutcome},
	)

	ApplicationsCompensatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "compensated_total",
			Help:      "Total number of mortgage applications that triggered compensation",
		},
		[]string{labelScenario, labelVersion},
	)

	ApplicationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "duration_seconds",
			Help:      "Duration of mortgage application workflow execution",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{labelScenario, labelVersion, labelOutcome},
	)
)

func init() {
	prometheus.MustRegister(
		ApplicationsStartedTotal,
		ApplicationsCompletedTotal,
		ApplicationsCompensatedTotal,
		ApplicationDurationSeconds,
	)
}
