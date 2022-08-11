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
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

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

func installExportPipeline(ctx context.Context, t *testing.T, writer io.Writer) *controller.Controller {
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
		controller.WithCollectPeriod(time.Second*3),
	)
	if err = pusher.Start(ctx); err != nil {
		t.Fatalf("starting push controller: %v", err)
	}
	return pusher
}

type AwaitFunc func(*ThreadSafeBuffer) bool

func awaitOutput(t *testing.T, callBack AwaitFunc, output *ThreadSafeBuffer) {
	require.Eventually(t, func() bool {
		return callBack(output)
	}, time.Second*5, time.Millisecond*10, "output should not be empty")
}

func deserializeOutput(t *testing.T, output *ThreadSafeBuffer) []Datapoint {
	var res []Datapoint
	// convert output to actual JSON, since otlp stdout exporter is not JSON compatible
	out := strings.ReplaceAll(output.String(), "]\n[", ",")
	err := json.Unmarshal([]byte(out), &res)
	require.NoError(t, err)

	sort.Slice(res[:], func(i, j int) bool {
		return res[i].Name > res[j].Name
	})
	return res
}
