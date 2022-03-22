package sundheitotel

import (
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const (
	instrumentationName    = "appsflyer.com/observability"
	instrumentationVersion = "v0.0.1"
)

var (
	meterTest = global.GetMeterProvider().Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(instrumentationVersion),
	)
)

func (s *TestSuite) TestInt64Gauge() {
	g := NewInt64Gauge(meterTest, "test.int64gauge")
	tags := attribute.String("a", "b")
	g.Record(10, tags)
	time.Sleep(time.Second)
	g.Record(-30, tags)
	time.Sleep(time.Second)
	g.Record(50, tags)
	time.Sleep(time.Second)

	s.AwaitOutput(await)
	dataPoints := s.deserializeOutput()
	require.Equal(s.T(), int64(-30), dataPoints[0].Last)
	require.Equal(s.T(), int64(10), dataPoints[1].Last)
	require.Equal(s.T(), int64(50), dataPoints[2].Last)
}

func (s *TestSuite) TestLastValueAggregationNoAttributes() {
	g := NewInt64Gauge(meterTest, "test.lastvalue.gauge")
	for i := 0; i <= 10; i++ {
		g.Record(int64(i))
	}
	s.AwaitOutput(await)

	dataPoints := s.deserializeOutput()
	require.Len(s.T(), dataPoints, 1)
	require.Equal(s.T(), int64(10), dataPoints[0].Last)
}

func (s *TestSuite) TestLastValueAggregationWithAttributes() {
	g := NewInt64Gauge(meterTest, "test.lastvalue.withattributes.gauge")
	tags := attribute.String("a", "b")
	for i := 0; i <= 10; i++ {
		g.Record(int64(i), tags)
	}
	s.AwaitOutput(await)
	dataPoints := s.deserializeOutput()
	require.Len(s.T(), dataPoints, 1)
	require.Equal(s.T(), int64(10), dataPoints[0].Last)
}

func (s *TestSuite) TestMultipleAttributes() {
	tag1 := attribute.String("a", "1")
	tag2 := attribute.String("b", "2")
	g := NewInt64Gauge(meterTest, "test.multipleattributes.gauge")
	for i := 0; i <= 3; i++ {
		g.Record(int64(i), tag1)
		g.Record(int64(i), tag2)
	}
	s.AwaitOutput(await)

	dataPoints := s.deserializeOutput()
	require.Len(s.T(), dataPoints, 2)
	require.Equal(s.T(), int64(3), dataPoints[0].Last)
	require.Equal(s.T(), int64(3), dataPoints[1].Last)
}

func await(output *ThreadSafeBuffer) bool {
	return output.Len() > 0
}
