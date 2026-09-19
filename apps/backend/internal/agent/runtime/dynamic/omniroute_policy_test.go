package dynamic

import (
	"testing"

	"github.com/kandev/kandev/internal/agent/runtime/routingerr"
)

func TestPolicyCandidatesFollowTaskClass(t *testing.T) {
	profile := Profile{PolicyKind: PolicyOmniRouteCostFirstV1, Candidates: []Candidate{
		{ID: "strong", Enabled: true, RouteClass: RouteClassStrong},
		{ID: "free", Enabled: true, RouteClass: RouteClassFree},
		{ID: "local", Enabled: true, RouteClass: RouteClassLocal},
		{ID: "cheap", Enabled: true, RouteClass: RouteClassLowCost},
	}}

	tests := []struct {
		name          string
		class         TaskClass
		authorization EscalationReason
		want          []string
	}{
		{name: "simple", class: TaskClassSimple, want: []string{"local", "free"}},
		{name: "medium", class: TaskClassMedium, want: []string{"local", "free", "cheap"}},
		{name: "simple exhausted", class: TaskClassSimple, authorization: EscalationCheapChainExhausted, want: []string{"local", "free", "strong"}},
		{name: "medium quality", class: TaskClassMedium, authorization: EscalationQualityFail, want: []string{"local", "free", "cheap", "strong"}},
		{name: "high risk", class: TaskClassHighRisk, want: []string{"strong", "free", "local", "cheap"}},
		{name: "high complexity", class: TaskClassHighComplexity, want: []string{"strong", "free", "local", "cheap"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyOmniRoutePolicy(profile, tt.class, tt.authorization)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Candidates) != len(tt.want) {
				t.Fatalf("candidates = %v, want %v", candidateIDs(got.Candidates), tt.want)
			}
			for i := range tt.want {
				if got.Candidates[i].ID != tt.want[i] {
					t.Fatalf("candidates = %v, want %v", candidateIDs(got.Candidates), tt.want)
				}
			}
		})
	}
}

func TestPolicyCandidatesFailClosed(t *testing.T) {
	profile := Profile{PolicyKind: PolicyOmniRouteCostFirstV1, Candidates: []Candidate{{ID: "strong", Enabled: true, RouteClass: RouteClassStrong}}}
	if _, err := ApplyOmniRoutePolicy(profile, TaskClassSimple, EscalationReason("manual")); err == nil {
		t.Fatal("unknown escalation reason must fail closed")
	}
	if _, err := ApplyOmniRoutePolicy(profile, TaskClass("urgent"), EscalationNone); err == nil {
		t.Fatal("unknown task class must fail closed")
	}
}

func TestFailureCategoryRemainsDistinct(t *testing.T) {
	tests := []struct {
		code routingerr.Code
		want FailureCategory
	}{
		{routingerr.CodeRateLimited, FailureCategoryRateLimit},
		{routingerr.CodeQuotaLimited, FailureCategoryQuota},
		{routingerr.CodeNetworkUnavailable, FailureCategoryProviderOther},
	}
	for _, tt := range tests {
		if got := ProviderFailureCategory(tt.code); got != tt.want {
			t.Fatalf("ProviderFailureCategory(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
	if got := ClassifiedFailureCategory(&routingerr.Error{Code: routingerr.CodeNetworkUnavailable, RawExcerpt: "i/o timeout"}); got != FailureCategoryTimeout {
		t.Fatalf("timeout category = %q", got)
	}
	if got := ClassifiedFailureCategory(nil); got != FailureCategoryConfiguration {
		t.Fatalf("nil failure category = %q", got)
	}
}

func TestCandidatesAfterNeverWrapsCheapChain(t *testing.T) {
	candidates := []Candidate{{ID: "local"}, {ID: "free"}, {ID: "strong"}}
	got := CandidatesAfter(candidates, "free")
	if len(got) != 1 || got[0].ID != "strong" {
		t.Fatalf("CandidatesAfter = %v, want [strong]", candidateIDs(got))
	}
	if got := CandidatesAfter(candidates, "missing"); got != nil {
		t.Fatalf("missing current candidate returned %v", candidateIDs(got))
	}
}

func candidateIDs(candidates []Candidate) []string {
	ids := make([]string, len(candidates))
	for i := range candidates {
		ids[i] = candidates[i].ID
	}
	return ids
}
