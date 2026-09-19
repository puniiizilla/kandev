package dynamic

import (
	"expvar"
	"strconv"
	"testing"
)

func TestRecordPolicyAttemptUsesBoundedDimensions(t *testing.T) {
	before := expvarInt(policyRouteAttempts, string(RouteClassStrong))
	recordPolicyAttempt(RouteClassStrong, EscalationQualityEscalate)
	after := expvarInt(policyRouteAttempts, string(RouteClassStrong))
	if after != before+1 {
		t.Fatalf("strong attempts = %d, want %d", after, before+1)
	}
	if policyRouteAttempts.Get("unknown") != nil {
		t.Fatal("unknown route class created an unbounded metric key")
	}
}

func expvarInt(metric *expvar.Map, key string) int64 {
	value := metric.Get(key)
	if value == nil {
		return 0
	}
	parsed, _ := strconv.ParseInt(value.String(), 10, 64)
	return parsed
}
