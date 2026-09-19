package models

import "time"

type ArchitectureEvidenceState string

const (
	ArchitectureEvidenceNotRequired      ArchitectureEvidenceState = "NOT_REQUIRED"
	ArchitectureEvidenceBaselinePending  ArchitectureEvidenceState = "BASELINE_PENDING"
	ArchitectureEvidenceBaselineReady    ArchitectureEvidenceState = "BASELINE_READY"
	ArchitectureEvidenceAfterPending     ArchitectureEvidenceState = "AFTER_PENDING"
	ArchitectureEvidenceDiffPending      ArchitectureEvidenceState = "DIFF_PENDING"
	ArchitectureEvidenceReadyForReview   ArchitectureEvidenceState = "READY_FOR_REVIEW"
	ArchitectureEvidenceBaselineFailed   ArchitectureEvidenceState = "ARCHIFY_BASELINE_FAILED"
	ArchitectureEvidenceAfterFailed      ArchitectureEvidenceState = "ARCHIFY_AFTER_FAILED"
	ArchitectureEvidenceDiffFailed       ArchitectureEvidenceState = "ARCHIFY_DIFF_FAILED"
	ArchitectureEvidenceValidationFailed ArchitectureEvidenceState = "ARCHIFY_VALIDATION_FAILED"
	ArchitectureEvidenceStale            ArchitectureEvidenceState = "STALE"
)

// ArchitectureEvidence is a bounded, SHA-pinned projection of Archify output.
// It intentionally contains no architecture source bytes.
type ArchitectureEvidence struct {
	ID                   string                    `db:"id" json:"id"`
	TaskID               string                    `db:"task_id" json:"task_id"`
	RepositoryID         string                    `db:"repository_id" json:"repository_id"`
	ArchitecturePath     string                    `db:"architecture_path" json:"architecture_path"`
	PolicyVersion        string                    `db:"policy_version" json:"policy_version"`
	Required             bool                      `db:"required" json:"required"`
	RelevanceRule        string                    `db:"relevance_rule" json:"relevance_rule"`
	OverrideActorID      string                    `db:"override_actor_id" json:"override_actor_id,omitempty"`
	OverrideReason       string                    `db:"override_reason" json:"override_reason,omitempty"`
	BaseSHA              string                    `db:"base_sha" json:"base_sha,omitempty"`
	HeadSHA              string                    `db:"head_sha" json:"head_sha,omitempty"`
	RuntimeContract      string                    `db:"runtime_contract" json:"runtime_contract,omitempty"`
	State                ArchitectureEvidenceState `db:"state" json:"state"`
	ValidationStatus     string                    `db:"validation_status" json:"validation_status,omitempty"`
	ArchitectureChanged  bool                      `db:"architecture_changed" json:"architecture_changed"`
	SemanticChange       bool                      `db:"semantic_change" json:"semantic_change"`
	AffectedDiagramsJSON string                    `db:"affected_diagrams_json" json:"affected_diagrams_json,omitempty"`
	DiffSummaryJSON      string                    `db:"diff_summary_json" json:"diff_summary_json,omitempty"`
	DiagnosticsJSON      string                    `db:"diagnostics_json" json:"diagnostics_json,omitempty"`
	BaselineReceiptJSON  string                    `db:"baseline_receipt_json" json:"baseline_receipt_json,omitempty"`
	BaselineHash         string                    `db:"baseline_hash" json:"baseline_hash,omitempty"`
	AfterReceiptJSON     string                    `db:"after_receipt_json" json:"after_receipt_json,omitempty"`
	AfterHash            string                    `db:"after_hash" json:"after_hash,omitempty"`
	DeltaReceiptJSON     string                    `db:"delta_receipt_json" json:"delta_receipt_json,omitempty"`
	DeltaHash            string                    `db:"delta_hash" json:"delta_hash,omitempty"`
	CacheKey             string                    `db:"cache_key" json:"cache_key,omitempty"`
	SourceWorkflowStepID string                    `db:"source_workflow_step_id" json:"source_workflow_step_id,omitempty"`
	Attempt              int                       `db:"attempt" json:"attempt"`
	CreatedAt            time.Time                 `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time                 `db:"updated_at" json:"updated_at"`
	CompletedAt          *time.Time                `db:"completed_at" json:"completed_at,omitempty"`
}

type ArchitectureEvidenceEvent struct {
	ID         string                    `db:"id" json:"id"`
	EvidenceID string                    `db:"evidence_id" json:"evidence_id"`
	TaskID     string                    `db:"task_id" json:"task_id"`
	State      ArchitectureEvidenceState `db:"state" json:"state"`
	ReasonCode string                    `db:"reason_code" json:"reason_code,omitempty"`
	BaseSHA    string                    `db:"base_sha" json:"base_sha,omitempty"`
	HeadSHA    string                    `db:"head_sha" json:"head_sha,omitempty"`
	ActorID    string                    `db:"actor_id" json:"actor_id,omitempty"`
	CreatedAt  time.Time                 `db:"created_at" json:"created_at"`
}
