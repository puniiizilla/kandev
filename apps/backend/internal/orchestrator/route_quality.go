package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
)

type routeQualityRecorder interface {
	RecordRouteQualityResult(context.Context, dynamicruntime.RouteQualityResult) error
}

func (s *Service) RecordRouteQualityResult(ctx context.Context, result dynamicruntime.RouteQualityResult) (*RouteActionResult, error) {
	if err := validateRouteQualityResult(result); err != nil {
		return nil, err
	}
	recorder, ok := s.repo.(routeQualityRecorder)
	if !ok {
		return nil, errors.New("route quality persistence is not configured")
	}
	if result.Result == dynamicruntime.QualityPass {
		if err := recorder.RecordRouteQualityResult(ctx, result); err != nil {
			return nil, err
		}
		return nil, nil
	}
	session, err := s.repo.GetTaskSession(ctx, result.SessionID)
	if err != nil {
		return nil, err
	}
	action := RouteActionTryNext
	reason := dynamicruntime.EscalationQualityFail
	switch result.Result {
	case dynamicruntime.QualityRetry:
		action = RouteActionRetry
		reason = dynamicruntime.EscalationNone
	case dynamicruntime.QualityEscalate:
		reason = dynamicruntime.EscalationQualityEscalate
	}
	request := RouteActionRequest{
		SessionID: result.SessionID, Action: action,
		ExpectedGeneration: session.RouteGeneration, EscalationReason: reason,
	}
	if s.routeActionHandler == nil {
		return nil, errors.New("dynamic route actions are not configured")
	}
	if err := s.authorizeSession(ctx, result.SessionID); err != nil {
		return nil, err
	}
	releaseRouteAction := s.acquireRouteActionOperationLock(result.SessionID)
	defer releaseRouteAction()
	if err := s.rejectRouteActionDuringActiveTurn(ctx, result.SessionID); err != nil {
		return nil, err
	}
	if err := recorder.RecordRouteQualityResult(ctx, result); err != nil {
		return nil, err
	}
	return s.routeActionHandler(ctx, request)
}

func validateRouteQualityResult(result dynamicruntime.RouteQualityResult) error {
	if result.TaskID == "" || result.SessionID == "" || result.RouteAttemptID == "" {
		return errors.New("task_id, session_id, and route_attempt_id are required")
	}
	switch result.Result {
	case dynamicruntime.QualityPass, dynamicruntime.QualityFail, dynamicruntime.QualityRetry, dynamicruntime.QualityEscalate:
	default:
		return fmt.Errorf("unsupported quality result %q", result.Result)
	}
	switch result.Source {
	case dynamicruntime.QualitySourceWorkflow, dynamicruntime.QualitySourceReview,
		dynamicruntime.QualitySourceVerification, dynamicruntime.QualitySourceOperator:
	default:
		return fmt.Errorf("unsupported quality source %q", result.Source)
	}
	if strings.TrimSpace(result.SourceReference) == "" || len(result.SourceReference) > 200 {
		return errors.New("source_reference must contain 1 to 200 bytes")
	}
	return nil
}
