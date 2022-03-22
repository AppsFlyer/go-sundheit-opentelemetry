package sundheitotel

import (
	"go.opentelemetry.io/otel/metric/global"
)

const (
	// ValAllChecks is the value used for the check tags when tagging all tests
	ValAllChecks       = "all_checks"
	StatusMetricName   = "health/status"
	DurationMetricName = "health/execute_time"
)

var (
	meter             = global.Meter("gosundheit-otel")
	keyCheck          = "check"
	keyCheckPassing   = "check_passing"
	keyClassification = "classification"
	mStatus           = NewInt64Gauge(meter, StatusMetricName)
	mDuration         = NewInt64Gauge(meter, DurationMetricName)
)

type status bool

func (s status) asInt64() int64 {
	if s {
		return 1
	}
	return 0
}
