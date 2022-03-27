package sundheitotel

import (
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"go.opentelemetry.io/otel/attribute"
)

type MetricsListener struct {
	classification string
	gauge          Int64Gauge
}

func NewMetricsListener(opts ...Option) *MetricsListener {
	listener := &MetricsListener{}

	for _, opt := range append(opts, WithDefaults()) {
		opt(listener)
	}

	return listener
}

func (c *MetricsListener) OnCheckRegistered(name string, result gosundheit.Result) {
	c.recordCheck(name, result)
}

func (c *MetricsListener) OnCheckStarted(_ string) {
}

func (c *MetricsListener) OnCheckCompleted(name string, result gosundheit.Result) {
	c.recordCheck(name, result)
}

func (c *MetricsListener) OnResultsUpdated(results map[string]gosundheit.Result) {
	allHealthy := allHealthy(results)
	tags := c.defaultTags(ValAllChecks, allHealthy)
	mStatus.Record(status(allHealthy).asInt64(), tags...)
}

func (c *MetricsListener) recordCheck(name string, result gosundheit.Result) {
	duration := int64(result.Duration) / int64(time.Millisecond)
	tags := c.defaultTags(name, result.IsHealthy())
	mDuration.Record(duration, tags...)
}

func (c *MetricsListener) defaultTags(checkName string, isPassing bool) []attribute.KeyValue {
	tags := []attribute.KeyValue{
		attribute.String(keyCheck, checkName),
		attribute.Bool(keyCheckPassing, isPassing),
	}
	if c.classification != "" {
		tags = append(tags, attribute.String(keyClassification, c.classification))
	}
	return tags
}
