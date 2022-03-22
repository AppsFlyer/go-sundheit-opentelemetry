package sundheitotel

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// nolint: structcheck
type gauge struct {
	values *sync.Map
	name   string
}

type Int64Gauge gauge

func NewInt64Gauge(meter metric.Meter, name string) Int64Gauge {
	g := Int64Gauge{values: &sync.Map{}, name: name}
	g.registerCallback(meter, name)
	return g
}

func (g *Int64Gauge) Record(value int64, tags ...attribute.KeyValue) {
	serializedTags, err := serializeAttributes(tags)
	if err != nil {
		// metric will be recorded with no tags
		serializedTags = ""
	}

	g.values.Store(serializedTags, value)
}

func (g *Int64Gauge) registerCallback(meter metric.Meter, name string) {
	metric.Must(meter).NewInt64GaugeObserver(name, func(ctx context.Context, result metric.Int64ObserverResult) {
		g.values.Range(func(tags, value interface{}) bool {
			attributes, err := deserializeAttributes(tags.(string))
			if err != nil {
				return false
			}
			result.Observe(value.(int64), attributes...)
			g.values.Delete(tags)
			return true
		})
	})
}
