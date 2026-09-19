package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kandev/kandev/internal/architecture"
	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/events/bus"
	"github.com/kandev/kandev/internal/task/models"
	v1 "github.com/kandev/kandev/pkg/api/v1"
)

const ArchifyRelevancePolicyVersion = "archify_relevance_v1"

var ErrArchitectureEvidenceNotReady = errors.New("architecture evidence is not ready for review")

type architectureEvidenceStore interface {
	GetArchitectureEvidence(context.Context, string) (*models.ArchitectureEvidence, error)
	UpsertArchitectureEvidence(context.Context, *models.ArchitectureEvidence, string, string) error
}

type architectureEvidenceFlight struct {
	done  chan struct{}
	value *models.ArchitectureEvidence
	err   error
}

type architectureEvidenceRun struct {
	task         *models.Task
	session      *models.TaskSession
	repository   *models.Repository
	store        architectureEvidenceStore
	value        *models.ArchitectureEvidence
	worktreePath string
}

func EvaluateArchifyRelevance(labels, changedPaths []string, architecturePath string, override *bool) (bool, string) {
	root := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(architecturePath)), "/") + "/"
	for _, path := range changedPaths {
		if strings.HasPrefix(filepath.ToSlash(path), root) {
			return true, "architecture_source_changed"
		}
	}
	set := map[string]bool{}
	for _, label := range labels {
		set[strings.ToLower(strings.TrimSpace(label))] = true
	}
	if set["archify:required"] {
		return true, "required_label"
	}
	if override != nil {
		return *override, "human_override"
	}
	for _, label := range []string{"architecture", "runtime", "dataflow", "persistence", "authority"} {
		if set[label] {
			return true, "impact_label"
		}
	}
	return false, "no_match"
}

func (s *Service) SetArchitectureService(value *architecture.Service) { s.architecture = value }

func (s *Service) GetArchitectureEvidence(ctx context.Context, taskID string) (*models.ArchitectureEvidence, error) {
	if err := s.authorizeTaskID(ctx, taskID); err != nil {
		return nil, err
	}
	store, ok := s.tasks.(architectureEvidenceStore)
	if !ok {
		return nil, errors.New("architecture evidence store unavailable")
	}
	return store.GetArchitectureEvidence(ctx, taskID)
}

func (s *Service) RefreshArchitectureEvidence(ctx context.Context, taskID, actorID string) (*models.ArchitectureEvidence, error) {
	s.architectureFlightsMu.Lock()
	if flight := s.architectureEvidenceFlights[taskID]; flight != nil {
		s.architectureFlightsMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-flight.done:
			return flight.value, flight.err
		}
	}
	if s.architectureEvidenceFlights == nil {
		s.architectureEvidenceFlights = make(map[string]*architectureEvidenceFlight)
	}
	flight := &architectureEvidenceFlight{done: make(chan struct{})}
	s.architectureEvidenceFlights[taskID] = flight
	s.architectureFlightsMu.Unlock()

	flight.value, flight.err = s.refreshArchitectureEvidence(ctx, taskID, actorID)
	s.architectureFlightsMu.Lock()
	delete(s.architectureEvidenceFlights, taskID)
	close(flight.done)
	s.architectureFlightsMu.Unlock()
	return flight.value, flight.err
}

func (s *Service) refreshArchitectureEvidence(ctx context.Context, taskID, actorID string) (*models.ArchitectureEvidence, error) {
	run, err := s.prepareArchitectureEvidenceRun(ctx, taskID)
	if err != nil {
		return nil, err
	}
	value, store := run.value, run.store
	if !value.Required {
		value.State = models.ArchitectureEvidenceNotRequired
		value.ValidationStatus = "SKIPPED"
		return value, s.persistArchitectureEvidence(ctx, store, value, "policy_not_required", actorID)
	}
	if strings.TrimSpace(run.session.BaseCommitSHA) == "" {
		value.State = models.ArchitectureEvidenceBaselineFailed
		_ = s.persistArchitectureEvidence(ctx, store, value, "base_sha_missing", actorID)
		return value, ErrArchitectureEvidenceNotReady
	}
	if dirty, dirtyErr := gitLines(ctx, run.worktreePath, "status", "--porcelain"); dirtyErr != nil || len(dirty) > 0 {
		value.State = models.ArchitectureEvidenceAfterFailed
		value.DiagnosticsJSON = `[{"code":"DIRTY_WORKTREE","message":"Commit task changes before architecture capture."}]`
		_ = s.persistArchitectureEvidence(ctx, store, value, "dirty_worktree", actorID)
		return value, ErrArchitectureEvidenceNotReady
	}
	headLines, err := gitLines(ctx, run.worktreePath, "rev-parse", "HEAD")
	if err != nil || len(headLines) != 1 {
		value.State = models.ArchitectureEvidenceAfterFailed
		_ = s.persistArchitectureEvidence(ctx, store, value, "head_sha_missing", actorID)
		return value, ErrArchitectureEvidenceNotReady
	}
	value.HeadSHA = headLines[0]
	return s.captureArchitectureEvidence(ctx, run, actorID)
}

func (s *Service) prepareArchitectureEvidenceRun(ctx context.Context, taskID string) (*architectureEvidenceRun, error) {
	if s.architecture == nil {
		return nil, errors.New("architecture service unavailable")
	}
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	session, err := s.GetPrimarySession(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("primary session required: %w", err)
	}
	repository, err := s.architectureRepository(ctx, task)
	if err != nil {
		return nil, err
	}
	store, ok := s.tasks.(architectureEvidenceStore)
	if !ok {
		return nil, errors.New("architecture evidence store unavailable")
	}
	current, err := store.GetArchitectureEvidence(ctx, taskID)
	if err != nil {
		return nil, err
	}
	labels := parseArchitectureLabels(task.Labels)
	worktreePath := architectureWorktreePath(session, repository.ID)
	changed, err := gitLines(ctx, worktreePath, "diff", "--name-only", session.BaseCommitSHA+"...HEAD")
	if err != nil {
		return nil, err
	}
	var override *bool
	if current != nil && current.OverrideReason != "" {
		value := current.Required
		override = &value
	}
	required, rule := EvaluateArchifyRelevance(labels, changed, repository.ArchitecturePath, override)
	value := current
	if value == nil {
		value = &models.ArchitectureEvidence{ID: uuid.NewString(), TaskID: taskID, RepositoryID: repository.ID, ArchitecturePath: repository.ArchitecturePath, CreatedAt: time.Now().UTC()}
	}
	value.PolicyVersion = ArchifyRelevancePolicyVersion
	value.Required = required
	value.RelevanceRule = rule
	value.BaseSHA = session.BaseCommitSHA
	value.SourceWorkflowStepID = task.WorkflowStepID
	return &architectureEvidenceRun{task: task, session: session, repository: repository, store: store, value: value, worktreePath: worktreePath}, nil
}

func (s *Service) captureArchitectureEvidence(ctx context.Context, run *architectureEvidenceRun, actorID string) (*models.ArchitectureEvidence, error) {
	value, store := run.value, run.store
	value.State = models.ArchitectureEvidenceBaselinePending
	_ = s.persistArchitectureEvidence(ctx, store, value, "baseline_started", actorID)
	base, err := s.architecture.CaptureRevision(ctx, run.task.WorkspaceID, value.BaseSHA)
	if err != nil {
		value.State = models.ArchitectureEvidenceBaselineFailed
		value.DiagnosticsJSON = architectureDiagnostic("BASELINE_CAPTURE_FAILED", err)
		_ = s.persistArchitectureEvidence(ctx, store, value, "baseline_failed", actorID)
		return value, ErrArchitectureEvidenceNotReady
	}
	value.RuntimeContract = base.RuntimeContract
	value.BaselineReceiptJSON = string(base.Receipt)
	value.BaselineHash = base.Hash
	value.State = models.ArchitectureEvidenceAfterPending
	_ = s.persistArchitectureEvidence(ctx, store, value, "baseline_ready", actorID)
	after, err := s.architecture.CaptureRevision(ctx, run.task.WorkspaceID, value.HeadSHA)
	if err != nil {
		value.State = models.ArchitectureEvidenceValidationFailed
		value.ValidationStatus = "FAIL"
		value.DiagnosticsJSON = architectureDiagnostic("AFTER_CAPTURE_FAILED", err)
		_ = s.persistArchitectureEvidence(ctx, store, value, "after_failed", actorID)
		return value, ErrArchitectureEvidenceNotReady
	}
	value.AfterReceiptJSON = string(after.Receipt)
	value.AfterHash = after.Hash
	value.ValidationStatus = "PASS"
	value.State = models.ArchitectureEvidenceDiffPending
	_ = s.persistArchitectureEvidence(ctx, store, value, "after_ready", actorID)
	delta, err := s.architecture.CompareRevisions(ctx, run.task.WorkspaceID, value.BaseSHA, value.HeadSHA)
	if err != nil {
		value.State = models.ArchitectureEvidenceDiffFailed
		value.DiagnosticsJSON = architectureDiagnostic("ARCHITECTURE_DIFF_FAILED", err)
		_ = s.persistArchitectureEvidence(ctx, store, value, "diff_failed", actorID)
		return value, ErrArchitectureEvidenceNotReady
	}
	value.ArchitectureChanged = delta.ArchitectureChanged
	value.SemanticChange = delta.SemanticChange
	diagrams, _ := json.Marshal(delta.AffectedDiagrams)
	value.AffectedDiagramsJSON = string(diagrams)
	value.DiffSummaryJSON = string(delta.Summary)
	value.DeltaReceiptJSON = string(delta.Receipt)
	value.DeltaHash = delta.Hash
	value.CacheKey = delta.CacheKey
	if !s.architectureEvidenceInputsCurrent(ctx, run) {
		value.State = models.ArchitectureEvidenceStale
		_ = s.persistArchitectureEvidence(ctx, store, value, "evidence_inputs_changed", actorID)
		return value, ErrArchitectureEvidenceNotReady
	}
	value.State = models.ArchitectureEvidenceReadyForReview
	now := time.Now().UTC()
	value.CompletedAt = &now
	return value, s.persistArchitectureEvidence(ctx, store, value, "ready_for_review", actorID)
}

func (s *Service) architectureEvidenceInputsCurrent(ctx context.Context, run *architectureEvidenceRun) bool {
	value := run.value
	currentHead, headErr := gitLines(ctx, run.worktreePath, "rev-parse", "HEAD")
	latestTask, taskErr := s.GetTask(ctx, value.TaskID)
	latestSession, sessionErr := s.GetPrimarySession(ctx, value.TaskID)
	return headErr == nil && len(currentHead) == 1 && currentHead[0] == value.HeadSHA &&
		taskErr == nil && latestTask != nil && latestTask.WorkflowStepID == value.SourceWorkflowStepID &&
		sessionErr == nil && latestSession != nil && latestSession.BaseCommitSHA == value.BaseSHA
}

func (s *Service) persistArchitectureEvidence(ctx context.Context, store architectureEvidenceStore, value *models.ArchitectureEvidence, reason, actorID string) error {
	if err := store.UpsertArchitectureEvidence(ctx, value, reason, actorID); err != nil {
		return err
	}
	if s.eventBus != nil {
		snapshot := *value
		payload := map[string]any{"task_id": value.TaskID, "evidence": &snapshot}
		if err := s.eventBus.Publish(ctx, events.TaskArchitectureEvidenceUpdated, bus.NewEvent(events.TaskArchitectureEvidenceUpdated, "task-service", payload)); err != nil && s.logger != nil {
			s.logger.Error("failed to publish architecture evidence update")
		}
	}
	return nil
}

func (s *Service) architectureRepository(ctx context.Context, task *models.Task) (*models.Repository, error) {
	repositories, err := s.ListRepositories(ctx, task.WorkspaceID)
	if err != nil {
		return nil, err
	}
	attached := map[string]bool{}
	for _, link := range task.Repositories {
		attached[link.RepositoryID] = true
	}
	var match *models.Repository
	for _, repository := range repositories {
		if attached[repository.ID] && repository.ArchitecturePath != "" && repository.ArchifyRuntime != "" && repository.ArchitectureGitRef != "" {
			if match != nil {
				return nil, errors.New("multiple attached Archify repositories")
			}
			match = repository
		}
	}
	if match == nil {
		return nil, errors.New("attached Archify repository is required")
	}
	return match, nil
}

func parseArchitectureLabels(raw string) []string {
	var labels []string
	_ = json.Unmarshal([]byte(raw), &labels)
	return labels
}

func architectureWorktreePath(session *models.TaskSession, repositoryID string) string {
	for _, worktree := range session.Worktrees {
		if worktree != nil && worktree.RepositoryID == repositoryID && strings.TrimSpace(worktree.WorktreePath) != "" {
			return worktree.WorktreePath
		}
	}
	return session.WorkspacePath
}
func gitLines(ctx context.Context, directory string, args ...string) ([]string, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("git worktree path is required")
	}
	output, err := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, args...)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git command failed: %w", err)
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func architectureDiagnostic(code string, err error) string {
	body, _ := json.Marshal([]map[string]string{{"code": code, "message": err.Error()}})
	return string(body)
}

func (s *Service) OverrideArchitectureEvidence(ctx context.Context, taskID string, required bool, reason, actorID string) (*models.ArchitectureEvidence, error) {
	if strings.TrimSpace(reason) == "" || strings.TrimSpace(actorID) == "" {
		return nil, errors.New("override reason and actor are required")
	}
	current, err := s.GetArchitectureEvidence(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, errors.New("architecture evidence must be initialized")
	}
	current.Required = required
	current.OverrideReason = reason
	current.OverrideActorID = actorID
	current.RelevanceRule = "human_override"
	store := s.tasks.(architectureEvidenceStore)
	return current, s.persistArchitectureEvidence(ctx, store, current, "override", actorID)
}

func (s *Service) RequireArchitectureEvidence(ctx context.Context, taskID string) error {
	value, err := s.RefreshArchitectureEvidence(ctx, taskID, "")
	if err != nil {
		return err
	}
	if value.Required && value.State != models.ArchitectureEvidenceReadyForReview {
		return ErrArchitectureEvidenceNotReady
	}
	return nil
}

func (s *Service) PrepareArchitectureBaseline(ctx context.Context, taskID string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	current, err := s.GetArchitectureEvidence(ctx, taskID)
	if err != nil {
		return err
	}
	var override *bool
	if current != nil && current.OverrideReason != "" {
		value := current.Required
		override = &value
	}
	required, _ := EvaluateArchifyRelevance(parseArchitectureLabels(task.Labels), nil, "", override)
	if !required {
		return nil
	}
	repository, err := s.architectureRepository(ctx, task)
	if err != nil {
		return err
	}
	for {
		session, sessionErr := s.GetPrimarySession(ctx, taskID)
		if sessionErr != nil {
			return sessionErr
		}
		if architectureWorkspaceReady(session, repository.ID) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	value, err := s.RefreshArchitectureEvidence(ctx, taskID, "")
	if err != nil {
		return err
	}
	if value.State != models.ArchitectureEvidenceReadyForReview {
		return ErrArchitectureEvidenceNotReady
	}
	return nil
}

func architectureWorkspaceReady(session *models.TaskSession, repositoryID string) bool {
	return session != nil && strings.TrimSpace(session.BaseCommitSHA) != "" &&
		strings.TrimSpace(architectureWorktreePath(session, repositoryID)) != ""
}

func (s *Service) ArchitectureEvidenceDelta(ctx context.Context, taskID, diagramID string) ([]byte, error) {
	value, err := s.GetArchitectureEvidence(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if value == nil || value.State != models.ArchitectureEvidenceReadyForReview {
		return nil, ErrArchitectureEvidenceNotReady
	}
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	comparison, err := s.architecture.CompareRevisions(ctx, task.WorkspaceID, value.BaseSHA, value.HeadSHA)
	if err != nil {
		return nil, err
	}
	if comparison.Hash != value.DeltaHash {
		return nil, errors.New("architecture delta receipt does not match pinned evidence")
	}
	return s.architecture.Delta(ctx, task.WorkspaceID, value.BaseSHA, value.HeadSHA, diagramID)
}

func (s *Service) preflightArchitectureReviewState(ctx context.Context, taskID string, state v1.TaskState) error {
	if state != v1.TaskStateReview || s.architecture == nil {
		return nil
	}
	return s.RequireArchitectureEvidence(ctx, taskID)
}
