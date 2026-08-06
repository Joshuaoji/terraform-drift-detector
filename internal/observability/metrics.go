package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	scansTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "driftdetect_scans_total",
		Help: "Total number of scans by status and provider",
	}, []string{"status", "provider"})

	scanDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "driftdetect_scan_duration_seconds",
		Help:    "Scan duration in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 12),
	}, []string{"provider"})

	driftsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "driftdetect_drifts_total",
		Help: "Total drift findings by type",
	}, []string{"type"})

	webhookDeliveries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "driftdetect_webhook_deliveries_total",
		Help: "Webhook delivery attempts by status",
	}, []string{"status"})
)

// MetricsHandler serves Prometheus metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RecordScanCompleted records metrics for a completed scan.
func RecordScanCompleted(provider string, duration time.Duration, totalDrifts int) {
	scansTotal.WithLabelValues("completed", provider).Inc()
	scanDuration.WithLabelValues(provider).Observe(duration.Seconds())
	if totalDrifts > 0 {
		driftsTotal.WithLabelValues("total").Add(float64(totalDrifts))
	}
}

// RecordScanFailed records a failed scan.
func RecordScanFailed(provider string) {
	scansTotal.WithLabelValues("failed", provider).Inc()
}

// RecordWebhookDelivery records webhook outcome.
func RecordWebhookDelivery(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	webhookDeliveries.WithLabelValues(status).Inc()
}
