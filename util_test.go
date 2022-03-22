package sundheitotel

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

type TestSuite struct {
	suite.Suite
	output *ThreadSafeBuffer
}

func (s *TestSuite) SetupSuite() {
	s.output = new(ThreadSafeBuffer)
	installExportPipeline(context.Background(), s.T(), s.output, time.Millisecond*100)
}

func (s *TestSuite) SetupTest() {
	s.output.buffer.Reset()
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

type Datapoint struct {
	Name string
	Last int64
}

type ThreadSafeBuffer struct {
	buffer bytes.Buffer
	sync.Mutex
}

func (b *ThreadSafeBuffer) Read(p []byte) (n int, err error) {
	b.Lock()
	defer b.Unlock()
	return b.buffer.Read(p)
}
func (b *ThreadSafeBuffer) Write(p []byte) (n int, err error) {
	b.Lock()
	defer b.Unlock()
	return b.buffer.Write(p)
}

func (b *ThreadSafeBuffer) Len() int {
	b.Lock()
	defer b.Unlock()
	return b.buffer.Len()
}

func (b *ThreadSafeBuffer) String() string {
	b.Lock()
	defer b.Unlock()
	return b.buffer.String()
}

func installExportPipeline(ctx context.Context, t *testing.T, writer io.Writer, collectPeriod time.Duration) func() {
	t.Logf("starting exporter pipeline")
	exporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint(), stdoutmetric.WithWriter(writer))
	if err != nil {
		t.Fatalf("creating stdoutmetric exporter: %v", err)
	}

	pusher := controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			exporter,
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(collectPeriod),
	)
	if err = pusher.Start(ctx); err != nil {
		t.Fatalf("starting push controller: %v", err)
	}
	global.SetMeterProvider(pusher)

	return func() {
		if err := pusher.Stop(ctx); err != nil {
			t.Fatalf("stopping push controller: %v", err)
		}
	}
}

type AwaitFunc func(*ThreadSafeBuffer) bool

func (s *TestSuite) AwaitOutput(callBack AwaitFunc) {
	require.Eventually(s.T(), func() bool {
		return callBack(s.output)
	}, time.Second*10, time.Second, "output should not be empty")
}

func (s *TestSuite) deserializeOutput() []Datapoint {
	var res []Datapoint
	// convert output to actual JSON, since otlp stdout exporter is not JSON compatible
	out := strings.ReplaceAll(s.output.String(), "]\n[", ",")
	err := json.Unmarshal([]byte(out), &res)
	require.NoError(s.T(), err)

	sort.Slice(res[:], func(i, j int) bool {
		return res[i].Last < res[j].Last
	})
	return res
}
