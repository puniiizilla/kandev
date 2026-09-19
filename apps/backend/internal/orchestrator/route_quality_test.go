package orchestrator

import (
	"testing"

	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
)

func TestValidateRouteQualityResultFailsClosed(t *testing.T) {
	valid := dynamicruntime.RouteQualityResult{
		TaskID: "task", SessionID: "session", RouteAttemptID: "attempt",
		Result: dynamicruntime.QualityFail, Source: dynamicruntime.QualitySourceReview,
		SourceReference: "review-1",
	}
	if err := validateRouteQualityResult(valid); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}
	invalid := []dynamicruntime.RouteQualityResult{
		{TaskID: "task", SessionID: "session", Result: dynamicruntime.QualityFail, Source: dynamicruntime.QualitySourceReview, SourceReference: "review-1"},
		{TaskID: "task", SessionID: "session", RouteAttemptID: "attempt", Result: "approve", Source: dynamicruntime.QualitySourceReview, SourceReference: "review-1"},
		{TaskID: "task", SessionID: "session", RouteAttemptID: "attempt", Result: dynamicruntime.QualityFail, Source: "agent", SourceReference: "review-1"},
		{TaskID: "task", SessionID: "session", RouteAttemptID: "attempt", Result: dynamicruntime.QualityFail, Source: dynamicruntime.QualitySourceReview},
	}
	for i, result := range invalid {
		if err := validateRouteQualityResult(result); err == nil {
			t.Fatalf("invalid result %d accepted", i)
		}
	}
}
