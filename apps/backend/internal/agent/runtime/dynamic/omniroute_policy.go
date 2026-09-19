package dynamic

import (
	"fmt"
	"strings"

	"github.com/kandev/kandev/internal/agent/runtime/routingerr"
)

const PolicyOmniRouteCostFirstV1 = "omniroute_cost_first_v1"

type RouteClass string

const (
	RouteClassLocal   RouteClass = "local"
	RouteClassFree    RouteClass = "free"
	RouteClassLowCost RouteClass = "low_cost"
	RouteClassStrong  RouteClass = "strong"
)

type EscalationReason string

const (
	EscalationNone                EscalationReason = ""
	EscalationCheapChainExhausted EscalationReason = "cheap_chain_exhausted"
	EscalationQualityFail         EscalationReason = "quality_fail"
	EscalationQualityEscalate     EscalationReason = "quality_escalate"
)

type FailureCategory string

const (
	FailureCategoryRateLimit     FailureCategory = "rate_limit"
	FailureCategoryQuota         FailureCategory = "quota"
	FailureCategoryTimeout       FailureCategory = "timeout"
	FailureCategoryProviderOther FailureCategory = "provider_other"
	FailureCategoryQualityGate   FailureCategory = "quality_gate"
	FailureCategoryConfiguration FailureCategory = "configuration"
)

// ApplyOmniRoutePolicy limits Kandev to choosing complete configured profiles.
// Provider and model selection remain inside OmniRoute.
func ApplyOmniRoutePolicy(profile Profile, taskClass TaskClass, authorization EscalationReason) (Profile, error) {
	if profile.PolicyKind != PolicyOmniRouteCostFirstV1 {
		return profile, nil
	}
	if !validEscalationReason(authorization) {
		return Profile{}, fmt.Errorf("unsupported escalation reason %q", authorization)
	}
	allowed, err := allowedRouteClasses(taskClass, authorization)
	if err != nil {
		return Profile{}, err
	}
	filtered := profile
	filtered.TaskClass = taskClass
	filtered.EscalationReason = authorization
	filtered.Candidates = make([]Candidate, 0, len(profile.Candidates))
	if taskClass == TaskClassHighRisk || taskClass == TaskClassHighComplexity {
		for _, candidate := range profile.Candidates {
			if _, ok := allowed[candidate.RouteClass]; ok {
				filtered.Candidates = append(filtered.Candidates, candidate)
			}
		}
		return filtered, nil
	}
	for _, routeClass := range []RouteClass{RouteClassLocal, RouteClassFree, RouteClassLowCost, RouteClassStrong} {
		if _, ok := allowed[routeClass]; !ok {
			continue
		}
		for _, candidate := range profile.Candidates {
			if candidate.RouteClass == routeClass {
				filtered.Candidates = append(filtered.Candidates, candidate)
			}
		}
	}
	return filtered, nil
}

func allowedRouteClasses(taskClass TaskClass, authorization EscalationReason) (map[RouteClass]struct{}, error) {
	allowed := map[RouteClass]struct{}{RouteClassLocal: {}, RouteClassFree: {}}
	switch taskClass {
	case TaskClassSimple:
	case TaskClassMedium:
		allowed[RouteClassLowCost] = struct{}{}
	case TaskClassHighRisk, TaskClassHighComplexity:
		allowed[RouteClassLowCost] = struct{}{}
		allowed[RouteClassStrong] = struct{}{}
	default:
		return nil, fmt.Errorf("unsupported task class %q", taskClass)
	}
	if authorization != EscalationNone {
		allowed[RouteClassStrong] = struct{}{}
	}
	return allowed, nil
}

func validEscalationReason(reason EscalationReason) bool {
	switch reason {
	case EscalationNone, EscalationCheapChainExhausted, EscalationQualityFail, EscalationQualityEscalate:
		return true
	default:
		return false
	}
}

func ProviderFailureCategory(code routingerr.Code) FailureCategory {
	switch code {
	case routingerr.CodeRateLimited:
		return FailureCategoryRateLimit
	case routingerr.CodeQuotaLimited:
		return FailureCategoryQuota
	default:
		return FailureCategoryProviderOther
	}
}

func ClassifiedFailureCategory(failure *routingerr.Error) FailureCategory {
	if failure == nil {
		return FailureCategoryConfiguration
	}
	signal := strings.ToLower(failure.ClassifierRule + " " + failure.RawExcerpt)
	if strings.Contains(signal, "timeout") || strings.Contains(signal, "timed out") || strings.Contains(signal, "etimedout") {
		return FailureCategoryTimeout
	}
	return ProviderFailureCategory(failure.Code)
}

func CandidatesAfter(candidates []Candidate, currentID string) []Candidate {
	for i := range candidates {
		if candidates[i].ID == currentID {
			return append([]Candidate(nil), candidates[i+1:]...)
		}
	}
	return nil
}
