package sundheitotel

import "go.opentelemetry.io/otel/metric/global"

const (
	// ValAllChecks is the value used for the check tags when tagging all tests
	ValAllChecks       = "all_checks"
	StatusMetricName   = "health/status"
	DurationMetricName = "health/execute_time"
)

var (
	defaultMeter      = global.Meter("gosundheit-otel")
	keyCheck          = "check"
	keyCheckPassing   = "check_passing"
	keyClassification = "classification"
)

type status bool

func (s status) asInt64() int64 {
	if s {
		return 1
	}
	return 0
}

type intStatus int64

func (i intStatus) asBool() bool {
	return i == 1
}
