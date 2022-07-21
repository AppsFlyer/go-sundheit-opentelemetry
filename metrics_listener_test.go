package sundheitotel

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"regexp"
	"strconv"
	"strings"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	health "github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"
	"github.com/pkg/errors"
)

const (
	successMsg = "success"
	failedMsg  = "failed"

	failingCheckName = "failing.check"
	passingCheckName = "passing.check"
	statusCheckName  = "all_checks"
	sleepTime        = 25
)

type Metric struct {
	Name string
	Tags string
}

func (s *TestSuite) getMetricName(metric string, checkName string, passing bool, classification ...string) Metric {
	classificationTag := ""
	if len(classification) > 0 {
		classificationTag = fmt.Sprintf(",classification=%s", classification[0])
	}

	return Metric{
		Name: metric,
		Tags: fmt.Sprintf("check=%s,check_passing=%s%s", checkName, strconv.FormatBool(passing), classificationTag),
	}
}

func (s *TestSuite) getDurationMetric(checkName string, passing bool, classification ...string) Metric {
	return s.getMetricName(DurationMetricName, checkName, passing, classification...)
}

func (s *TestSuite) getStatusMetric(checkName string, passing bool, classification ...string) Metric {
	return s.getMetricName(StatusMetricName, checkName, passing, classification...)
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

func (s *TestSuite) runTestHealthMetrics(checkName string, passing bool, initiallyPassing bool) {
	// Prepare
	listener, err := NewMetricsListener()
	s.Require().NoError(err)
	h := health.New(health.WithCheckListeners(listener), health.WithHealthListeners(listener))

	// Act
	registerCheck(h, checkName, passing, initiallyPassing)
	defer h.DeregisterAll()
	s.AwaitOutput(awaitMetric)

	// Assert
	dataPoints := s.deserializeOutput()
	require.Len(s.T(), dataPoints, 3)
	require.Equal(s.T(), s.getDurationMetric(checkName, initiallyPassing), getMetric(dataPoints[0].Name))
	require.Equal(s.T(), status(initiallyPassing).asInt64(), dataPoints[0].Last)

	require.Equal(s.T(), s.getStatusMetric(statusCheckName, passing), getMetric(dataPoints[1].Name))
	require.Equal(s.T(), status(passing).asInt64(), dataPoints[1].Last)

	require.Equal(s.T(), s.getDurationMetric(checkName, passing), getMetric(dataPoints[2].Name))
	require.True(s.T(), sleepTime <= dataPoints[2].Last)
}

func (s *TestSuite) TestHealthMetricsPassing() {
	s.runTestHealthMetrics(passingCheckName, true, false)
}

func (s *TestSuite) TestHealthMetricsFailing() {
	s.runTestHealthMetrics(failingCheckName, false, false)
}

func (s *TestSuite) runTestHealthMetricsWithClassification(option Option, classification string) {
	listener, err := NewMetricsListener(option)
	s.Require().NoError(err)
	h := health.New(health.WithCheckListeners(listener), health.WithHealthListeners(listener))
	checkName := passingCheckName
	passing := true
	initiallyPassing := false
	registerCheck(h, checkName, passing, initiallyPassing)
	defer h.DeregisterAll()

	s.AwaitOutput(awaitMetric)
	dataPoints := s.deserializeOutput()

	require.Len(s.T(), dataPoints, 3)
	require.Equal(s.T(), s.getDurationMetric(checkName, initiallyPassing, classification), getMetric(dataPoints[0].Name))
	require.Equal(s.T(), status(initiallyPassing).asInt64(), dataPoints[0].Last)

	require.Equal(s.T(), s.getStatusMetric(statusCheckName, passing, classification), getMetric(dataPoints[1].Name))
	require.Equal(s.T(), status(passing).asInt64(), dataPoints[1].Last)

	require.Equal(s.T(), s.getDurationMetric(checkName, passing, classification), getMetric(dataPoints[2].Name))
	require.True(s.T(), sleepTime <= dataPoints[2].Last)
}

func (s *TestSuite) TestHealthMetricsWithLivenessClassification() {
	s.runTestHealthMetricsWithClassification(WithLivenessClassification(), "liveness")
}

func (s *TestSuite) TestHealthMetricsWithStartupClassification() {
	s.runTestHealthMetricsWithClassification(WithStartupClassification(), "startup")
}

func (s *TestSuite) TestHealthMetricsWithReadinessClassification() {
	s.runTestHealthMetricsWithClassification(WithReadinessClassification(), "readiness")
}

func (s *TestSuite) TestHealthMetricsWithCustomClassification() {
	s.runTestHealthMetricsWithClassification(WithClassification("demo"), "demo")
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
	time.Sleep(sleepTime * time.Millisecond)
	if c.passing {
		return fmt.Sprintf("%s; i=%d", successMsg, c.counts), nil
	}

	return fmt.Sprintf("%s; i=%d", failedMsg, c.counts), errors.New(failedMsg)
}

func awaitMetric(output *ThreadSafeBuffer) bool {
	return strings.Contains(output.String(), StatusMetricName)
}
