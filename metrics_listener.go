package sundheitotel

import (
	"context"
	gosundheit "github.com/AppsFlyer/go-sundheit"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"sync/atomic"
	"unsafe"
)

type MetricsListener struct {
	classification string
	mStatus        asyncint64.Gauge
	mDuration      asyncint64.Gauge
	statusResult   int64
	durationResult int64
	tags           []attribute.KeyValue
}

func NewMetricsListener(opts ...Option) (*MetricsListener, error) {
	mStatus, err := meter.AsyncInt64().Gauge(StatusMetricName)
	if err != nil {
		return nil, err
	}
	mDuration, err := meter.AsyncInt64().Gauge(DurationMetricName)
	if err != nil {
		return nil, err
	}

	listener := &MetricsListener{
		mStatus:   mStatus,
		mDuration: mDuration,
	}

	if err := meter.RegisterCallback(
		[]instrument.Asynchronous{
			mStatus,
		}, listener.statusCallback,
	); err != nil {
		return nil, err
	}

	if err := meter.RegisterCallback(
		[]instrument.Asynchronous{
			mDuration,
		}, listener.durationCallback,
	); err != nil {
		return nil, err
	}

	for _, opt := range append(opts, WithDefaults()) {
		opt(listener)
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
	allHealthy := allHealthy(results)
	atomic.StoreInt64(&c.statusResult, status(allHealthy).asInt64())
}

func (c *MetricsListener) statusCallback(ctx context.Context) {
	atomic.LoadInt64(&c.statusResult)
	c.mStatus.Observe(ctx, c.statusResult, c.defaultTags(ValAllChecks, intStatus(c.statusResult).asBool())...)
}

func (c *MetricsListener) durationCallback(ctx context.Context) {
	atomic.LoadInt64(&c.durationResult)
	c.mDuration.Observe(ctx, c.durationResult, c.defaultTags(ValAllChecks, intStatus(c.durationResult).asBool())...)

}

func (c *MetricsListener) recordCheck(name string, result gosundheit.Result) {
	isHealthy := result.IsHealthy()
	key := unsafe.Pointer(&c.tags)
	tags := c.defaultTags(name, isHealthy)
	atomic.StorePointer(&key, unsafe.Pointer(&tags))
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
