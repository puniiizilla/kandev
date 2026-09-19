package dynamic

import "expvar"

var (
	policyRouteAttempts = expvar.NewMap("dynamic_policy_route_attempts_total")
	policyEscalations   = expvar.NewMap("dynamic_policy_escalations_total")
)

func recordPolicyAttempt(routeClass RouteClass, reason EscalationReason) {
	switch routeClass {
	case RouteClassLocal, RouteClassFree, RouteClassLowCost, RouteClassStrong:
		policyRouteAttempts.Add(string(routeClass), 1)
	default:
		return
	}
	switch reason {
	case EscalationCheapChainExhausted, EscalationQualityFail, EscalationQualityEscalate:
		policyEscalations.Add(string(reason), 1)
	}
}
