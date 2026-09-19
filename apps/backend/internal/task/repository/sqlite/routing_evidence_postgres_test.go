package sqlite

import (
	"context"
	"testing"
	"time"

	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
	"github.com/kandev/kandev/internal/testutil"
)

// TestPostgresRoutingEvidenceRoundTrips is environment-gated by
// KANDEV_TEST_POSTGRES_DSN through PostgresDSNFromEnv.
func TestPostgresRoutingEvidenceRoundTrips(t *testing.T) {
	db := testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t))
	repo, err := NewWithDB(db, db, nil)
	if err != nil {
		t.Fatalf("init postgres schema: %v", err)
	}
	seedPostgresTaskSession(t, repo, "routing-evidence-pg", "routing-evidence-session-pg")
	decision := dynamicruntime.RouteDecision{
		SessionID: "routing-evidence-session-pg", LogicalProfileID: "policy",
		ExecutionProfileID: "strong", Generation: 1, ProfileVersion: 1,
		RouteClass: dynamicruntime.RouteClassStrong, TaskClass: dynamicruntime.TaskClassHighComplexity,
		TaskClassSource: "label",
	}
	state := dynamicruntime.RouteState{
		SessionID: decision.SessionID, LogicalProfileID: decision.LogicalProfileID,
		ExecutionProfileID: decision.ExecutionProfileID, Generation: 1,
		ProfileVersion: 1, Status: "active", UpdatedAt: time.Now().UTC(),
	}
	if err := repo.RecordRouteDecision(context.Background(), decision, state); err != nil {
		t.Fatal(err)
	}
	evidence, err := repo.ListTaskRoutingEvidence(context.Background(), "routing-evidence-pg")
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Attempts) != 1 || evidence.PaidEscalationCount != 1 {
		t.Fatalf("evidence = %#v", evidence)
	}
}
