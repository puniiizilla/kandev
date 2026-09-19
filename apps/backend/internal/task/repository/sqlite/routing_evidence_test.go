package sqlite

import (
	"context"
	"testing"
	"time"

	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
	"github.com/kandev/kandev/internal/agent/runtime/routingerr"
	"github.com/kandev/kandev/internal/task/models"
)

func TestRoutingEvidenceSchemaContainsAuditColumns(t *testing.T) {
	repo := newRepoForSessionTests(t)
	rows, err := repo.db.Queryx(`PRAGMA table_info(dynamic_route_attempts)`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	want := map[string]bool{
		"task_id": false, "route_class": false, "task_class": false,
		"task_class_source": false, "attempt_ordinal": false,
		"escalation_reason": false, "failure_category": false,
		"quality_result": false, "quality_source": false,
		"usage_event_id": false, "latency_ms": false,
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("dynamic_route_attempts missing %s", name)
		}
	}
}

func TestRoutingEvidenceJoinsUsageAndCountsStrongAttemptsOnce(t *testing.T) {
	repo := newRepoForSessionTests(t)
	ctx := context.Background()
	if err := repo.CreateTask(ctx, &models.Task{ID: "evidence-task", Title: "Evidence"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateTaskSession(ctx, &models.TaskSession{ID: "evidence-session", TaskID: "evidence-task", State: models.TaskSessionStateStarting}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-2 * time.Second)
	decision := dynamicruntime.RouteDecision{
		SessionID: "evidence-session", LogicalProfileID: "policy", ExecutionProfileID: "omniroute-strong",
		Generation: 1, ProfileVersion: 3, RouteClass: dynamicruntime.RouteClassStrong,
		TaskClass: dynamicruntime.TaskClassHighRisk, TaskClassSource: "label",
		EscalationReason: dynamicruntime.EscalationQualityEscalate,
	}
	state := dynamicruntime.RouteState{SessionID: decision.SessionID, LogicalProfileID: decision.LogicalProfileID, ExecutionProfileID: decision.ExecutionProfileID, Generation: 1, ProfileVersion: 3, Status: "active", UpdatedAt: started}
	if err := repo.RecordRouteDecision(ctx, decision, state); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateTaskUsageEvent(ctx, &models.TaskUsageEvent{
		UsageEventID: "usage-evidence", TaskID: "evidence-task", SessionID: "evidence-session",
		AgentProfileID: "omniroute-strong", AgentType: "opencode", Provider: "effective-provider", Model: "effective-model",
		TokensIn: 10, TokensTotal: 15, CostSubcents: 42, CostSource: "provider_reported",
		ContractVersion: 1, OccurredAt: started.Add(1500 * time.Millisecond), CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.ListTaskRoutingEvidence(ctx, "evidence-task")
	if err != nil {
		t.Fatal(err)
	}
	if got.PaidEscalationCount != 1 || len(got.Attempts) != 1 {
		t.Fatalf("evidence = %#v", got)
	}
	attempt := got.Attempts[0]
	if attempt.RouteClass != dynamicruntime.RouteClassStrong || attempt.Provider == nil || *attempt.Provider != "effective-provider" || attempt.Model == nil || *attempt.Model != "effective-model" {
		t.Fatalf("attribution = %#v", attempt)
	}
	if attempt.TokensIn == nil || *attempt.TokensIn != 10 || attempt.ReportedCost == nil || *attempt.ReportedCost != 42 || attempt.LatencyMS == nil || *attempt.LatencyMS != 1500 {
		t.Fatalf("usage linkage = %#v", attempt)
	}
	if err := repo.CreateTaskUsageEvent(ctx, &models.TaskUsageEvent{UsageEventID: "usage-evidence", TaskID: "evidence-task", SessionID: "evidence-session", CostSource: "provider_reported", ContractVersion: 1, OccurredAt: time.Now().UTC(), CreatedAt: time.Now().UTC()}); err == nil {
		t.Fatal("usage replay unexpectedly succeeded")
	}
	replayed, err := repo.ListTaskRoutingEvidence(ctx, "evidence-task")
	if err != nil || replayed.PaidEscalationCount != 1 || len(replayed.Attempts) != 1 {
		t.Fatalf("replayed evidence = %#v, err = %v", replayed, err)
	}
}

func TestProviderFailureCategoryIsPersistedOnFailedAttempt(t *testing.T) {
	repo := newRepoForSessionTests(t)
	ctx := context.Background()
	if err := repo.CreateTask(ctx, &models.Task{ID: "failure-task", Title: "Failure"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateTaskSession(ctx, &models.TaskSession{ID: "failure-session", TaskID: "failure-task", State: models.TaskSessionStateStarting}); err != nil {
		t.Fatal(err)
	}
	engine := dynamicruntime.NewEngine(dynamicruntime.WithPersistence(repo), dynamicruntime.WithStateLoader(repo))
	profile := dynamicruntime.Profile{ID: "policy", PolicyKind: dynamicruntime.PolicyOmniRouteCostFirstV1, TaskClass: dynamicruntime.TaskClassSimple, TaskClassSource: "default", Candidates: []dynamicruntime.Candidate{
		{ID: "local", Enabled: true, BindingKey: "local", RouteClass: dynamicruntime.RouteClassLocal, Rules: map[string]dynamicruntime.Action{"on_provider_error": dynamicruntime.ActionTryNext}},
		{ID: "free", Enabled: true, BindingKey: "free", RouteClass: dynamicruntime.RouteClassFree},
	}}
	if _, err := engine.SelectContext(ctx, "failure-session", profile, 0, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.ApplyFailureContext(ctx, "failure-session", profile, 1, "local", &routingerr.Error{Code: routingerr.CodeRateLimited, Class: routingerr.ClassTransient, FallbackAllowed: true}); err != nil {
		t.Fatal(err)
	}
	evidence, err := repo.ListTaskRoutingEvidence(ctx, "failure-task")
	if err != nil || len(evidence.Attempts) != 2 {
		t.Fatalf("evidence = %#v, err = %v", evidence, err)
	}
	if evidence.Attempts[0].FailureCategory != dynamicruntime.FailureCategoryRateLimit {
		t.Fatalf("failure category = %q", evidence.Attempts[0].FailureCategory)
	}
}

func TestPolicyFlowPersistsLocalFreeLowCostStrongAcrossRestart(t *testing.T) {
	repo := newRepoForSessionTests(t)
	ctx := context.Background()
	if err := repo.CreateTask(ctx, &models.Task{ID: "flow-task", Title: "Flow"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateTaskSession(ctx, &models.TaskSession{ID: "flow-session", TaskID: "flow-task", State: models.TaskSessionStateWaitingForInput}); err != nil {
		t.Fatal(err)
	}
	base := dynamicruntime.Profile{ID: "policy", PolicyKind: dynamicruntime.PolicyOmniRouteCostFirstV1, Candidates: []dynamicruntime.Candidate{
		{ID: "strong", Enabled: true, BindingKey: "strong", RouteClass: dynamicruntime.RouteClassStrong},
		{ID: "cheap", Enabled: true, BindingKey: "cheap", RouteClass: dynamicruntime.RouteClassLowCost, Rules: map[string]dynamicruntime.Action{"on_provider_error": dynamicruntime.ActionTryNext}},
		{ID: "free", Enabled: true, BindingKey: "free", RouteClass: dynamicruntime.RouteClassFree, Rules: map[string]dynamicruntime.Action{"on_provider_error": dynamicruntime.ActionTryNext}},
		{ID: "local", Enabled: true, BindingKey: "local", RouteClass: dynamicruntime.RouteClassLocal, Rules: map[string]dynamicruntime.Action{"on_provider_error": dynamicruntime.ActionTryNext}},
	}}
	profile, err := dynamicruntime.ApplyOmniRoutePolicy(base, dynamicruntime.TaskClassMedium, dynamicruntime.EscalationNone)
	if err != nil {
		t.Fatal(err)
	}
	profile.TaskClassSource = "label"
	engine := dynamicruntime.NewEngine(dynamicruntime.WithPersistence(repo), dynamicruntime.WithStateLoader(repo))
	failure := &routingerr.Error{Code: routingerr.CodeRateLimited, Class: routingerr.ClassTransient, FallbackAllowed: true}
	selected, err := engine.SelectContext(ctx, "flow-session", profile, 0, "")
	if err != nil || selected.ExecutionProfileID != "local" {
		t.Fatalf("local = %#v, %v", selected, err)
	}
	recordFlowUsage(t, repo, "usage-local", "local-provider", "local-model", 1)
	selected, err = engine.ApplyFailureContext(ctx, "flow-session", base, 1, "local", failure)
	if err != nil || selected.ExecutionProfileID != "free" {
		t.Fatalf("free = %#v, %v", selected, err)
	}
	recordFlowUsage(t, repo, "usage-free", "free-provider", "free-model", 2)
	selected, err = engine.ApplyFailureContext(ctx, "flow-session", base, 2, "free", failure)
	if err != nil || selected.ExecutionProfileID != "cheap" {
		t.Fatalf("low cost = %#v, %v", selected, err)
	}
	recordFlowUsage(t, repo, "usage-cheap", "cheap-provider", "cheap-model", 3)

	attempts, err := repo.ListRouteAttempts(ctx, "flow-session")
	if err != nil || len(attempts) != 3 {
		t.Fatalf("attempts = %#v, %v", attempts, err)
	}
	if err := repo.RecordRouteQualityResult(ctx, dynamicruntime.RouteQualityResult{
		TaskID: "flow-task", SessionID: "flow-session", RouteAttemptID: attempts[2].ID,
		Result: dynamicruntime.QualityEscalate, Source: dynamicruntime.QualitySourceVerification,
		SourceReference: "verification-1",
	}); err != nil {
		t.Fatal(err)
	}

	engine = dynamicruntime.NewEngine(dynamicruntime.WithPersistence(repo), dynamicruntime.WithStateLoader(repo))
	escalated, err := dynamicruntime.ApplyOmniRoutePolicy(base, dynamicruntime.TaskClassMedium, dynamicruntime.EscalationQualityEscalate)
	if err != nil {
		t.Fatal(err)
	}
	escalated.TaskClassSource = "label"
	escalated.Candidates = dynamicruntime.CandidatesAfter(escalated.Candidates, "cheap")
	selected, err = engine.SelectContextWithReason(ctx, "flow-session", escalated, 3, "cheap", string(dynamicruntime.EscalationQualityEscalate))
	if err != nil || selected.ExecutionProfileID != "strong" {
		t.Fatalf("strong = %#v, %v", selected, err)
	}
	recordFlowUsage(t, repo, "usage-strong", "paid-provider", "paid-model", 4)

	evidence, err := repo.ListTaskRoutingEvidence(ctx, "flow-task")
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Attempts) != 4 || evidence.PaidEscalationCount != 1 {
		t.Fatalf("evidence = %#v", evidence)
	}
	wantClasses := []dynamicruntime.RouteClass{dynamicruntime.RouteClassLocal, dynamicruntime.RouteClassFree, dynamicruntime.RouteClassLowCost, dynamicruntime.RouteClassStrong}
	for i, want := range wantClasses {
		if evidence.Attempts[i].RouteClass != want || evidence.Attempts[i].UsageEventID == nil || evidence.Attempts[i].TaskClass != dynamicruntime.TaskClassMedium || evidence.Attempts[i].TaskClassSource != "label" {
			t.Fatalf("attempt %d = %#v", i, evidence.Attempts[i])
		}
	}
}

func recordFlowUsage(t *testing.T, repo *Repository, eventID, provider, model string, ordinal int64) {
	t.Helper()
	now := time.Now().UTC()
	if err := repo.CreateTaskUsageEvent(context.Background(), &models.TaskUsageEvent{
		UsageEventID: eventID, TaskID: "flow-task", SessionID: "flow-session",
		AgentProfileID: model, AgentType: "opencode", Provider: provider, Model: model,
		TokensIn: ordinal, TokensTotal: ordinal, CostSubcents: ordinal, CostSource: "provider_reported",
		ContractVersion: 1, OccurredAt: now, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}
