package orchestrator

import (
	"context"
	"errors"
	"testing"

	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

type architectureGateStub struct {
	err   error
	calls int
}

func (g *architectureGateStub) PrepareArchitectureBaseline(context.Context, string) error {
	return g.err
}
func (g *architectureGateStub) RequireArchitectureEvidence(context.Context, string) error {
	g.calls++
	return g.err
}

func TestArchitectureEvidenceGateCoversReviewAndApproval(t *testing.T) {
	blocked := errors.New("not ready")
	gate := &architectureGateStub{err: blocked}
	service := &Service{architectureEvidenceGate: gate}
	for _, stage := range []wfmodels.StageType{wfmodels.StageTypeReview, wfmodels.StageTypeApproval} {
		if !errors.Is(service.preflightArchitectureEvidence(context.Background(), "task", &wfmodels.WorkflowStep{StageType: stage}), blocked) {
			t.Fatalf("stage %s did not block", stage)
		}
	}
	if err := service.preflightArchitectureEvidence(context.Background(), "task", &wfmodels.WorkflowStep{StageType: wfmodels.StageTypeWork}); err != nil {
		t.Fatal(err)
	}
	if gate.calls != 2 {
		t.Fatalf("calls=%d", gate.calls)
	}
}
