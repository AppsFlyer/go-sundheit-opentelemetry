package sundheitotel

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	health "github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"
	"github.com/pkg/errors"
)

type Metric struct {
	Name string
	Tags string
}

func getMetricName(metric string, checkName string, passing bool, classification ...string) Metric {
	classificationTag := ""
	if len(classification) > 0 {
		classificationTag = fmt.Sprintf(",classification=%s", classification[0])
	}

	return Metric{
		Name: metric,
		Tags: fmt.Sprintf("check=%s,check_passing=%s%s", checkName, strconv.FormatBool(passing), classificationTag),
	}
}

func getDurationMetric(checkName string, passing bool, classification ...string) Metric {
	return getMetricName(DurationMetricName, checkName, passing, classification...)
}

func getStatusMetric(checkName string, passing bool, classification ...string) Metric {
	return getMetricName(StatusMetricName, checkName, passing, classification...)
}

func getMetric(metricData string) Metric {
	var reName = regexp.MustCompile(`[^{]*`)
	name := reName.FindStringSubmatch(metricData)
	var reTags = regexp.MustCompile(`check[^}]*`)
	tags := reTags.FindStringSubmatch(metricData)
	return Metric{
		Name: name[0],
		Tags: tags[0],
	}
}

func TestNoCheckNoMetrics(t *testing.T) {
	// Prepare
	output := new(ThreadSafeBuffer)
	ctx := context.Background()
	pusher := installExportPipeline(ctx, t, output)
	listener, err := NewMetricsListener(WithMeter(pusher.Meter(t.Name())))
	require.NoError(t, err)
	h := health.New(health.WithCheckListeners(listener), health.WithHealthListeners(listener))

	// Act
	defer h.DeregisterAll()

	// Assert
	require.Never(t, func() bool {
		return awaitMetric(output)
	}, time.Second*1, time.Millisecond*10, "output needs to be empty")
	if err := pusher.Stop(ctx); err != nil {
		t.Fatalf("stopping push controller: %v", err)
	}
}

func runTestHealthMetrics(t *testing.T, checkName string, passing bool, initiallyPassing bool) {
	// Prepare
	output := new(ThreadSafeBuffer)
	ctx := context.Background()
	pusher := installExportPipeline(ctx, t, output)
	listener, err := NewMetricsListener(WithMeter(pusher.Meter(t.Name())))
	require.NoError(t, err)
	h := health.New(health.WithCheckListeners(listener), health.WithHealthListeners(listener))

	// Act
	registerCheck(h, checkName, passing, initiallyPassing)
	defer h.DeregisterAll()
	awaitOutput(t, awaitMetric, output)

	// Assert
	dataPoints := deserializeOutput(t, output)
	fmt.Println(dataPoints)
	require.GreaterOrEqual(t, len(dataPoints), 3)

	assert.Equal(t, getStatusMetric(checkName, passing), getMetric(dataPoints[0].Name))
	assert.Equal(t, status(passing).asInt64(), dataPoints[0].Last)

	assert.Equal(t, getStatusMetric("all_checks", passing), getMetric(dataPoints[1].Name))
	assert.Equal(t, status(passing).asInt64(), dataPoints[1].Last)

	assert.Equal(t, getDurationMetric(checkName, passing), getMetric(dataPoints[2].Name))
	assert.True(t, 25 <= dataPoints[2].Last)
	if err := pusher.Stop(ctx); err != nil {
		t.Fatalf("stopping push controller: %v", err)
	}
}

func TestHealthMetricsPassing(t *testing.T) {
	runTestHealthMetrics(t, "passing.check", true, false)
}

func TestHealthMetricsFailing(t *testing.T) {
	runTestHealthMetrics(t, "failing.check", false, false)
}

func runTestHealthMetricsWithClassification(t *testing.T, option Option, classification string) {
	// Prepare
	output := new(ThreadSafeBuffer)
	ctx := context.Background()
	pusher := installExportPipeline(ctx, t, output)
	listener, err := NewMetricsListener(option, WithMeter(pusher.Meter(t.Name())))
	require.NoError(t, err)
	h := health.New(health.WithCheckListeners(listener), health.WithHealthListeners(listener))
	checkName := "passing.classification.check"
	passing := true
	initiallyPassing := false

	// Act
	registerCheck(h, checkName, passing, initiallyPassing)
	defer h.DeregisterAll()
	awaitOutput(t, awaitMetric, output)

	// Assert
	dataPoints := deserializeOutput(t, output)
	fmt.Println(dataPoints)
	require.Len(t, dataPoints, 3)
	require.Equal(t, getStatusMetric(checkName, passing, classification), getMetric(dataPoints[0].Name))
	require.Equal(t, status(passing).asInt64(), dataPoints[0].Last)

	require.Equal(t, getStatusMetric("all_checks", passing, classification), getMetric(dataPoints[1].Name))
	require.Equal(t, status(passing).asInt64(), dataPoints[1].Last)

	require.Equal(t, getDurationMetric(checkName, passing, classification), getMetric(dataPoints[2].Name))
	require.True(t, 25 <= dataPoints[2].Last)
	if err := pusher.Stop(ctx); err != nil {
		t.Fatalf("stopping push controller: %v", err)
	}
}

func TestHealthMetricsWithLivenessClassification(t *testing.T) {
	runTestHealthMetricsWithClassification(t, WithLivenessClassification(), "liveness")
}

func TestHealthMetricsWithStartupClassification(t *testing.T) {
	runTestHealthMetricsWithClassification(t, WithStartupClassification(), "startup")
}

func TestHealthMetricsWithReadinessClassification(t *testing.T) {
	runTestHealthMetricsWithClassification(t, WithReadinessClassification(), "readiness")
}

func TestHealthMetricsWithCustomClassification(t *testing.T) {
	runTestHealthMetricsWithClassification(t, WithClassification("demo"), "demo")
}

func registerCheck(h gosundheit.Health, name string, passing bool, initiallyPassing bool) {
	stub := checkStub{
		counts:  0,
		passing: passing,
	}

	_ = h.RegisterCheck(&checks.CustomCheck{
		CheckName: name,
		CheckFunc: stub.run,
	},
		gosundheit.InitialDelay(0),
		gosundheit.ExecutionPeriod(120*time.Minute),
		gosundheit.InitiallyPassing(initiallyPassing),
	)
}

type checkStub struct {
	counts  int64
	passing bool
}

func (c *checkStub) run(_ context.Context) (details interface{}, err error) {
	c.counts++
	time.Sleep(25 * time.Millisecond)
	if c.passing {
		return fmt.Sprintf("%s; i=%d", "success", c.counts), nil
	}

	return fmt.Sprintf("%s; i=%d", "failed", c.counts), errors.New("failed")
}

func awaitMetric(output *ThreadSafeBuffer) bool {
	return strings.Contains(output.String(), StatusMetricName)
}
