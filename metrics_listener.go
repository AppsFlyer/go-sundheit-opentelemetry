package sundheitotel

import (
	"context"
	"go.opentelemetry.io/otel/metric"
	"sync/atomic"
	"unsafe"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
)

type MetricsListener struct {
	classification   string
	mStatus          asyncint64.Gauge
	mDuration        asyncint64.Gauge
	meter            metric.Meter
	statusResult     int64
	durationResult   int64
	checkName        string
	allStatusResults int64
}

func NewMetricsListener(opts ...Option) (*MetricsListener, error) {
	listener := &MetricsListener{}
	defaults := []Option{WithDefaults()}
	for _, opt := range append(defaults, opts...) {
		opt(listener)
	}
	var err error
	listener.mStatus, err = listener.meter.AsyncInt64().Gauge(StatusMetricName)
	if err != nil {
		return nil, err
	}
	listener.mDuration, err = listener.meter.AsyncInt64().Gauge(DurationMetricName)
	if err != nil {
		return nil, err
	}

	if err := listener.meter.RegisterCallback(
		[]instrument.Asynchronous{
			listener.mStatus,
			listener.mDuration,
		}, listener.metricsCallback,
	); err != nil {
		return nil, err
	}

	return listener, nil
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
	atomic.StoreInt64(&c.allStatusResults, status(allHealthy(results)).asInt64())
}

func (c *MetricsListener) metricsCallback(ctx context.Context) {
	allStatusResults := atomic.LoadInt64(&c.allStatusResults)
	resultsTags := c.defaultTags(ValAllChecks, intStatus(allStatusResults).asBool())
	c.mStatus.Observe(ctx, allStatusResults, resultsTags...)

	checkName := (*string)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&c.checkName))))
	if checkName == nil {
		return
	}
	statusResult := atomic.LoadInt64(&c.statusResult)
	duration := atomic.LoadInt64(&c.durationResult)
	tags := c.defaultTags(*checkName, intStatus(statusResult).asBool())
	c.mStatus.Observe(ctx, statusResult, tags...)
	if duration != 0 {
		c.mDuration.Observe(ctx, duration, tags...)
	}
}

func (c *MetricsListener) recordCheck(name string, result gosundheit.Result) {
	isHealthy := result.IsHealthy()
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&c.checkName)), unsafe.Pointer(&name))
	atomic.StoreInt64(&c.durationResult, result.Duration.Milliseconds())
	atomic.StoreInt64(&c.statusResult, status(isHealthy).asInt64())
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
