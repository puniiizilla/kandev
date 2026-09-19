package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/kandev/kandev/internal/task/models"
)

// @covers AC-TASKS-ARCHIFY-EVIDENCE-002.5
func TestArchitectureEvidenceSchemaPersistsOneCurrentRecord(t *testing.T) {
	repo := newRepoForEntityTests(t)
	var table string
	err := repo.db.GetContext(context.Background(), &table,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'task_architecture_evidence'`)
	if err != nil {
		t.Fatalf("architecture evidence schema missing: %v", err)
	}
}

func TestUpsertArchitectureEvidenceIsBoundedAndIdempotent(t *testing.T) {
	repo := newRepoForEntityTests(t)
	ctx := context.Background()
	workspace := &models.Workspace{ID: "architecture-evidence-workspace", Name: "Architecture evidence"}
	if err := repo.CreateWorkspace(ctx, workspace); err != nil {
		t.Fatal(err)
	}
	task := &models.Task{ID: "architecture-evidence-task", WorkspaceID: workspace.ID, Title: "Architecture evidence"}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	repository := &models.Repository{ID: "architecture-evidence-repository", WorkspaceID: workspace.ID, Name: "Architecture evidence"}
	if err := repo.CreateRepository(ctx, repository); err != nil {
		t.Fatal(err)
	}
	evidence := &models.ArchitectureEvidence{
		TaskID: task.ID, RepositoryID: repository.ID, ArchitecturePath: "docs/archify/ist",
		PolicyVersion: "archify_relevance_v1", Required: true, RelevanceRule: "label",
		State: models.ArchitectureEvidenceBaselinePending, BaseSHA: "base", CreatedAt: time.Now().UTC(),
	}
	if err := repo.UpsertArchitectureEvidence(ctx, evidence, "baseline_started", "actor"); err != nil {
		t.Fatal(err)
	}
	evidence.State = models.ArchitectureEvidenceBaselineReady
	if err := repo.UpsertArchitectureEvidence(ctx, evidence, "captured", "actor"); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetArchitectureEvidence(ctx, task.ID)
	if err != nil || got.State != models.ArchitectureEvidenceBaselineReady || got.Attempt != 1 {
		t.Fatalf("got %#v, err %v", got, err)
	}
	if err := repo.UpsertArchitectureEvidence(ctx, evidence, "baseline_started", "actor"); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetArchitectureEvidence(ctx, task.ID)
	if err != nil || got.Attempt != 2 {
		t.Fatalf("rerun attempt = %#v, err %v", got, err)
	}
	var count int
	if err := repo.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM task_architecture_evidence WHERE task_id = ?`, task.ID); err != nil || count != 1 {
		t.Fatalf("current rows=%d err=%v", count, err)
	}
	evidence.DiagnosticsJSON = string(make([]byte, maxArchitectureEvidenceJSONBytes+1))
	if err := repo.UpsertArchitectureEvidence(ctx, evidence, "oversized", "actor"); err == nil {
		t.Fatal("expected bounded evidence rejection")
	}
}
