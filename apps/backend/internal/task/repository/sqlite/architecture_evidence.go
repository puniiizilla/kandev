package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kandev/kandev/internal/db/dialect"
	"github.com/kandev/kandev/internal/task/models"
)

const maxArchitectureEvidenceJSONBytes = 64 << 10

func (r *Repository) UpsertArchitectureEvidence(ctx context.Context, value *models.ArchitectureEvidence, reason, actorID string) error {
	if value == nil || value.TaskID == "" || value.RepositoryID == "" || value.ArchitecturePath == "" {
		return errors.New("architecture evidence identity is required")
	}
	for _, body := range []string{value.AffectedDiagramsJSON, value.DiffSummaryJSON, value.DiagnosticsJSON, value.BaselineReceiptJSON, value.AfterReceiptJSON, value.DeltaReceiptJSON} {
		if len(body) > maxArchitectureEvidenceJSONBytes {
			return errors.New("architecture evidence JSON exceeds 64 KiB")
		}
	}
	now := time.Now().UTC()
	if value.ID == "" {
		value.ID = uuid.NewString()
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = now
	}
	value.UpdatedAt = now
	attemptDelta := 0
	if reason == "baseline_started" {
		attemptDelta = 1
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	query := r.db.Rebind(`
		INSERT INTO task_architecture_evidence (
			id, task_id, repository_id, architecture_path, policy_version, required, relevance_rule,
			override_actor_id, override_reason, base_sha, head_sha, runtime_contract, state,
			validation_status, architecture_changed, semantic_change, affected_diagrams_json,
			diff_summary_json, diagnostics_json, baseline_receipt_json, baseline_hash,
			after_receipt_json, after_hash, delta_receipt_json, delta_hash, cache_key,
			source_workflow_step_id, attempt, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(task_id, repository_id, architecture_path) DO UPDATE SET
			policy_version=excluded.policy_version, required=excluded.required, relevance_rule=excluded.relevance_rule,
			override_actor_id=excluded.override_actor_id, override_reason=excluded.override_reason,
			base_sha=excluded.base_sha, head_sha=excluded.head_sha, runtime_contract=excluded.runtime_contract,
			state=excluded.state, validation_status=excluded.validation_status,
			architecture_changed=excluded.architecture_changed, semantic_change=excluded.semantic_change,
			affected_diagrams_json=excluded.affected_diagrams_json, diff_summary_json=excluded.diff_summary_json,
			diagnostics_json=excluded.diagnostics_json, baseline_receipt_json=excluded.baseline_receipt_json,
			baseline_hash=excluded.baseline_hash, after_receipt_json=excluded.after_receipt_json,
			after_hash=excluded.after_hash, delta_receipt_json=excluded.delta_receipt_json,
			delta_hash=excluded.delta_hash, cache_key=excluded.cache_key,
			source_workflow_step_id=excluded.source_workflow_step_id, attempt=task_architecture_evidence.attempt+?,
			updated_at=excluded.updated_at, completed_at=excluded.completed_at
	`)
	_, err = tx.ExecContext(ctx, query,
		value.ID, value.TaskID, value.RepositoryID, value.ArchitecturePath, value.PolicyVersion, dialect.BoolToInt(value.Required), value.RelevanceRule,
		value.OverrideActorID, value.OverrideReason, value.BaseSHA, value.HeadSHA, value.RuntimeContract, value.State,
		value.ValidationStatus, dialect.BoolToInt(value.ArchitectureChanged), dialect.BoolToInt(value.SemanticChange), jsonDefault(value.AffectedDiagramsJSON, "[]"),
		jsonDefault(value.DiffSummaryJSON, "{}"), jsonDefault(value.DiagnosticsJSON, "[]"), jsonDefault(value.BaselineReceiptJSON, "{}"), value.BaselineHash,
		jsonDefault(value.AfterReceiptJSON, "{}"), value.AfterHash, jsonDefault(value.DeltaReceiptJSON, "{}"), value.DeltaHash, value.CacheKey,
		value.SourceWorkflowStepID, value.CreatedAt, value.UpdatedAt, value.CompletedAt, attemptDelta)
	if err != nil {
		return fmt.Errorf("upsert architecture evidence: %w", err)
	}
	if err := tx.GetContext(ctx, value, r.db.Rebind(`SELECT * FROM task_architecture_evidence WHERE task_id=? AND repository_id=? AND architecture_path=?`), value.TaskID, value.RepositoryID, value.ArchitecturePath); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, r.db.Rebind(`INSERT INTO task_architecture_evidence_events (id,evidence_id,task_id,state,reason_code,base_sha,head_sha,actor_id,created_at) VALUES (?,?,?,?,?,?,?,?,?)`), uuid.NewString(), value.ID, value.TaskID, value.State, reason, value.BaseSHA, value.HeadSHA, actorID, now)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func jsonDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (r *Repository) GetArchitectureEvidence(ctx context.Context, taskID string) (*models.ArchitectureEvidence, error) {
	var value models.ArchitectureEvidence
	err := r.ro.GetContext(ctx, &value, r.ro.Rebind(`SELECT * FROM task_architecture_evidence WHERE task_id=? ORDER BY updated_at DESC LIMIT 1`), taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &value, err
}
