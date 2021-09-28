package dola

import "time"

type Metric int

const (
	// Submit order metrics.
	SubmitOrderMetric Metric = iota
	SubmitOrderLatencyMetric
	SubmitOrderErrorMetric
	// Submit bulk orders metrics.
	SubmitBulkOrderMetric
	SubmitBulkOrderLatencyMetric
	// Modify order metrics.
	ModifyOrderMetric
	ModifyOrderLatencyMetric
	ModifyOrderErrorMetric
	// Cancel order metrics.
	CancelOrderMetric
	CancelOrderLatencyMetric
	CancelOrderErrorMetric
	// Cancel all orders metrics.
	CancelAllOrdersMetric
	CancelAllOrdersLatencyMetric
	CancelAllOrdersErrorMetric
	// Get Active orders metrics.
	GetActiveOrdersMetric
	GetActiveOrdersLatencyMetric
	GetActiveOrdersErrorMetric
	// this should always be the last one.
	MaxMetrics
)

type Reporter interface {
	// Event metrics is a single occurrence
	Event(m Metric, labels ...string)
	// Latency metrics provides visibility over latencies
	Latency(m Metric, d time.Duration, labels ...string)
	// Value metrics provide visibility over arbitrary values
	Value(m Metric, v float64, labels ...string)
}
