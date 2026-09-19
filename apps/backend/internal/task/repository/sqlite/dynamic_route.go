package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
	"github.com/kandev/kandev/internal/agent/runtime/routingerr"
	"github.com/kandev/kandev/internal/task/models"
)

var _ dynamicruntime.Persistence = (*Repository)(nil)
var _ dynamicruntime.ContinuationPersistence = (*Repository)(nil)
var _ dynamicruntime.GenerationStatusClaimer = (*Repository)(nil)

func (r *Repository) SaveRouteState(ctx context.Context, state dynamicruntime.RouteState) error {
	if isTransientRouteSession(state.SessionID) {
		return nil
	}
	updatedAt := state.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO dynamic_route_states (
			session_id, logical_profile_id, execution_profile_id,
			route_generation, profile_version, state, continuation_json, policy_state_json, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			logical_profile_id = excluded.logical_profile_id,
			execution_profile_id = excluded.execution_profile_id,
			route_generation = excluded.route_generation,
			profile_version = excluded.profile_version,
			state = excluded.state,
			continuation_json = excluded.continuation_json,
			policy_state_json = excluded.policy_state_json,
			updated_at = excluded.updated_at
	`), state.SessionID, state.LogicalProfileID, state.ExecutionProfileID,
		state.Generation, state.ProfileVersion, state.Status, state.ContinuationJSON, state.PolicyStateJSON, updatedAt)
	return err
}

// ClaimRouteState advances a route generation only when the durable row still
// has the caller's expected generation. The insert path is reserved for the
// initial generation, so a restart cannot accidentally reset an existing
// session to generation one.
func (r *Repository) ClaimRouteState(ctx context.Context, expectedGeneration int64, state dynamicruntime.RouteState) (bool, error) {
	if isTransientRouteSession(state.SessionID) {
		return true, nil
	}
	var (
		result sql.Result
		err    error
	)
	if expectedGeneration == 0 {
		result, err = r.db.ExecContext(ctx, r.db.Rebind(`
			INSERT INTO dynamic_route_states (
				session_id, logical_profile_id, execution_profile_id,
				route_generation, profile_version, state, continuation_json, policy_state_json, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(session_id) DO NOTHING
		`), state.SessionID, state.LogicalProfileID, state.ExecutionProfileID,
			state.Generation, state.ProfileVersion, state.Status, state.ContinuationJSON, state.PolicyStateJSON, state.UpdatedAt)
	} else {
		result, err = r.db.ExecContext(ctx, r.db.Rebind(`
			UPDATE dynamic_route_states
			SET logical_profile_id = ?, execution_profile_id = ?,
				route_generation = ?, profile_version = ?, state = ?, continuation_json = ?, policy_state_json = ?, updated_at = ?
			WHERE session_id = ? AND route_generation = ?
		`), state.LogicalProfileID, state.ExecutionProfileID, state.Generation,
			state.ProfileVersion, state.Status, state.ContinuationJSON, state.PolicyStateJSON, state.UpdatedAt, state.SessionID,
			expectedGeneration)
	}
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

// ClaimRouteStateFrom updates a route generation only while both its
// generation and status still match the caller's observation. Same-generation
// transitions use this narrower fence so a late recovery callback cannot
// overwrite a route that became active (or vice versa).
func (r *Repository) ClaimRouteStateFrom(
	ctx context.Context,
	expectedGeneration int64,
	expectedStatus string,
	state dynamicruntime.RouteState,
) (bool, error) {
	if isTransientRouteSession(state.SessionID) {
		return true, nil
	}
	result, err := r.db.ExecContext(ctx, r.db.Rebind(`
		UPDATE dynamic_route_states
		SET logical_profile_id = ?, execution_profile_id = ?,
			route_generation = ?, profile_version = ?, state = ?, continuation_json = ?, policy_state_json = ?, updated_at = ?
		WHERE session_id = ? AND route_generation = ? AND state = ?
	`), state.LogicalProfileID, state.ExecutionProfileID,
		state.Generation, state.ProfileVersion, state.Status, state.ContinuationJSON, state.PolicyStateJSON, state.UpdatedAt,
		state.SessionID, expectedGeneration, expectedStatus)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

// RecordRouteDecision commits the generation claim and immutable attempt row
// together. A stale claim returns dynamicruntime.ErrStaleGeneration.
func (r *Repository) RecordRouteDecision(ctx context.Context, decision dynamicruntime.RouteDecision, state dynamicruntime.RouteState) error {
	if isTransientRouteSession(state.SessionID) {
		return nil
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	expectedGeneration := state.Generation - 1
	var result sql.Result
	if expectedGeneration == 0 {
		result, err = tx.ExecContext(ctx, r.db.Rebind(`
			INSERT INTO dynamic_route_states (
				session_id, logical_profile_id, execution_profile_id,
				route_generation, profile_version, state, continuation_json, policy_state_json, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(session_id) DO NOTHING
		`), state.SessionID, state.LogicalProfileID, state.ExecutionProfileID,
			state.Generation, state.ProfileVersion, state.Status, state.ContinuationJSON, state.PolicyStateJSON, state.UpdatedAt)
	} else {
		result, err = tx.ExecContext(ctx, r.db.Rebind(`
			UPDATE dynamic_route_states
			SET logical_profile_id = ?, execution_profile_id = ?,
				route_generation = ?, profile_version = ?, state = ?, continuation_json = ?, policy_state_json = ?, updated_at = ?
			WHERE session_id = ? AND route_generation = ?
		`), state.LogicalProfileID, state.ExecutionProfileID, state.Generation,
			state.ProfileVersion, state.Status, state.ContinuationJSON, state.PolicyStateJSON, state.UpdatedAt, state.SessionID,
			expectedGeneration)
	}
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return dynamicruntime.ErrStaleGeneration
	}
	createdAt := decisionReasonTime(decision, state)
	var taskID string
	if err := tx.QueryRowContext(ctx, r.db.Rebind(`SELECT task_id FROM task_sessions WHERE id = ?`), decision.SessionID).Scan(&taskID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO dynamic_route_attempts (
			id, session_id, task_id, logical_profile_id, execution_profile_id,
			route_generation, profile_version, route_class, task_class, task_class_source,
			attempt_ordinal, escalation_reason, failure_category, reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`), uuid.New().String(), decision.SessionID, taskID, decision.LogicalProfileID,
		decision.ExecutionProfileID, decision.Generation, decision.ProfileVersion,
		decision.RouteClass, decision.TaskClass, decision.TaskClassSource,
		decision.Generation, decision.EscalationReason, decision.FailureCategory,
		decision.Reason, createdAt); err != nil {
		return err
	}
	return tx.Commit()
}

func decisionReasonTime(_ dynamicruntime.RouteDecision, state dynamicruntime.RouteState) time.Time {
	if !state.UpdatedAt.IsZero() {
		return state.UpdatedAt
	}
	return time.Now().UTC()
}

func (r *Repository) AppendRouteAttempt(ctx context.Context, attempt dynamicruntime.RouteAttempt) error {
	if isTransientRouteSession(attempt.SessionID) {
		return nil
	}
	createdAt := attempt.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO dynamic_route_attempts (
			id, session_id, task_id, logical_profile_id, execution_profile_id,
			route_generation, profile_version, route_class, task_class, task_class_source,
			attempt_ordinal, escalation_reason, failure_category, reason, created_at
		) VALUES (?, ?, COALESCE(NULLIF(?, ''), (SELECT task_id FROM task_sessions WHERE id = ?)), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`), uuid.New().String(), attempt.SessionID, attempt.TaskID, attempt.SessionID, attempt.LogicalProfileID,
		attempt.ExecutionProfileID, attempt.Generation, attempt.ProfileVersion,
		attempt.RouteClass, attempt.TaskClass, attempt.TaskClassSource,
		attempt.AttemptOrdinal, attempt.EscalationReason, attempt.FailureCategory,
		attempt.Reason, createdAt)
	return err
}

func (r *Repository) LoadRouteState(ctx context.Context, sessionID string) (*dynamicruntime.RouteState, error) {
	if isTransientRouteSession(sessionID) {
		return nil, nil
	}
	state := &dynamicruntime.RouteState{}
	err := r.ro.QueryRowContext(ctx, r.ro.Rebind(`
		SELECT session_id, logical_profile_id, execution_profile_id,
		route_generation, profile_version, state, continuation_json, policy_state_json, updated_at
		FROM dynamic_route_states WHERE session_id = ?
	`), sessionID).Scan(
		&state.SessionID, &state.LogicalProfileID, &state.ExecutionProfileID,
		&state.Generation, &state.ProfileVersion, &state.Status, &state.ContinuationJSON, &state.PolicyStateJSON, &state.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

// ListPendingRouteStates returns only states whose durable policy deadline can
// be reconciled automatically. States marked retrying are intentionally not
// returned after restart because dispatch may already have crossed the
// process boundary and must remain a manual recovery decision.
func (r *Repository) ListPendingRouteStates(ctx context.Context) ([]dynamicruntime.RouteState, error) {
	rows, err := r.ro.QueryContext(ctx, r.ro.Rebind(`
		SELECT session_id, logical_profile_id, execution_profile_id,
			route_generation, profile_version, state, continuation_json, policy_state_json, updated_at
		FROM dynamic_route_states
		WHERE state IN (?, ?) ORDER BY updated_at ASC
	`), string("retry_wait"), string("waiting_for_reset"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	states := make([]dynamicruntime.RouteState, 0)
	for rows.Next() {
		var state dynamicruntime.RouteState
		if err := rows.Scan(
			&state.SessionID, &state.LogicalProfileID, &state.ExecutionProfileID,
			&state.Generation, &state.ProfileVersion, &state.Status,
			&state.ContinuationJSON, &state.PolicyStateJSON, &state.UpdatedAt,
		); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return states, nil
}

// ListStartingRouteStates returns every route whose durable status is still
// "starting". Startup reconciliation is the only caller: a healthy route also
// passes through "starting" en route to "active", so listing this status is
// only a safe orphan signal once the caller has confirmed there is no live
// session backing the row (see Service.reconcileOrphanedDynamicStartingRoutes).
func (r *Repository) ListStartingRouteStates(ctx context.Context) ([]dynamicruntime.RouteState, error) {
	rows, err := r.ro.QueryContext(ctx, r.ro.Rebind(`
		SELECT session_id, logical_profile_id, execution_profile_id,
			route_generation, profile_version, state, continuation_json, policy_state_json, updated_at
		FROM dynamic_route_states
		WHERE state = ? ORDER BY updated_at ASC
	`), "starting")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	states := make([]dynamicruntime.RouteState, 0)
	for rows.Next() {
		var state dynamicruntime.RouteState
		if err := rows.Scan(
			&state.SessionID, &state.LogicalProfileID, &state.ExecutionProfileID,
			&state.Generation, &state.ProfileVersion, &state.Status,
			&state.ContinuationJSON, &state.PolicyStateJSON, &state.UpdatedAt,
		); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return states, nil
}

func (r *Repository) SaveRouteContinuation(ctx context.Context, record dynamicruntime.ContinuationRecord) error {
	if isTransientRouteSession(record.SessionID) {
		return nil
	}
	payload, err := json.Marshal(record.Continuation)
	if err != nil {
		return err
	}
	updatedAt := record.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	result, err := r.db.ExecContext(ctx, r.db.Rebind(`
		UPDATE dynamic_route_states
		SET continuation_json = ?, updated_at = ?
		WHERE session_id = ? AND route_generation = ?
	`), string(payload), updatedAt, record.SessionID, record.Generation)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return dynamicruntime.ErrStaleGeneration
	}
	return nil
}

func (r *Repository) SaveCircuit(ctx context.Context, snapshot dynamicruntime.CircuitSnapshot) error {
	_, err := r.db.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO dynamic_resource_circuits
			(resource_key, state, until_at, code, probe_until, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(resource_key) DO UPDATE SET
			state = excluded.state,
			until_at = excluded.until_at,
			code = excluded.code,
			probe_until = excluded.probe_until,
			updated_at = excluded.updated_at
	`), snapshot.Key, snapshot.State, nullableTime(snapshot.Until), snapshot.Code,
		nullableTime(snapshot.ProbeUntil), time.Now().UTC())
	return err
}

func (r *Repository) LoadCircuits(ctx context.Context) ([]dynamicruntime.CircuitSnapshot, error) {
	rows, err := r.ro.QueryContext(ctx, r.ro.Rebind(`
		SELECT resource_key, state, until_at, code, probe_until
		FROM dynamic_resource_circuits
		WHERE state <> ? ORDER BY resource_key
	`), dynamicruntime.CircuitClosed)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var snapshots []dynamicruntime.CircuitSnapshot
	for rows.Next() {
		var snapshot dynamicruntime.CircuitSnapshot
		var until, probeUntil sql.NullTime
		var state string
		var code string
		if err := rows.Scan(&snapshot.Key, &state, &until, &code, &probeUntil); err != nil {
			return nil, err
		}
		snapshot.State = dynamicruntime.CircuitState(state)
		snapshot.Code = routingerr.Code(code)
		if until.Valid {
			snapshot.Until = until.Time
		}
		if probeUntil.Valid {
			snapshot.ProbeUntil = probeUntil.Time
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (r *Repository) LoadOrCreate(ctx context.Context) ([]byte, error) {
	var key []byte
	err := r.ro.QueryRowContext(ctx, `SELECT key_bytes FROM dynamic_installation_keys WHERE id = 1`).Scan(&key)
	if err == nil {
		return append([]byte(nil), key...), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO dynamic_installation_keys (id, key_bytes, created_at) VALUES (1, ?, ?)
	`), key, time.Now().UTC())
	if err != nil {
		// Another process may have won the first insert. Read its stable key.
		if readErr := r.ro.QueryRowContext(ctx, `SELECT key_bytes FROM dynamic_installation_keys WHERE id = 1`).Scan(&key); readErr == nil {
			return append([]byte(nil), key...), nil
		}
		return nil, err
	}
	return key, nil
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value
}

func (r *Repository) ListRouteAttempts(ctx context.Context, sessionID string) ([]dynamicruntime.RouteAttempt, error) {
	if isTransientRouteSession(sessionID) {
		return []dynamicruntime.RouteAttempt{}, nil
	}
	rows, err := r.ro.QueryContext(ctx, r.ro.Rebind(`
		SELECT id, session_id, logical_profile_id, execution_profile_id,
			route_generation, profile_version, reason, created_at
		FROM dynamic_route_attempts
		WHERE session_id = ? ORDER BY route_generation ASC
	`), sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	attempts := make([]dynamicruntime.RouteAttempt, 0)
	for rows.Next() {
		var attempt dynamicruntime.RouteAttempt
		if err := rows.Scan(&attempt.ID, &attempt.SessionID, &attempt.LogicalProfileID,
			&attempt.ExecutionProfileID, &attempt.Generation, &attempt.ProfileVersion,
			&attempt.Reason, &attempt.CreatedAt); err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attempts, nil
}

func (r *Repository) RecordRouteQualityResult(ctx context.Context, result dynamicruntime.RouteQualityResult) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var taskID string
	var attemptGeneration, currentGeneration int64
	err = tx.QueryRowContext(ctx, r.db.Rebind(`
		SELECT s.task_id, a.route_generation, rs.route_generation
		FROM dynamic_route_attempts a
		JOIN task_sessions s ON s.id = a.session_id
		JOIN dynamic_route_states rs ON rs.session_id = a.session_id
		WHERE a.id = ? AND a.session_id = ?
	`), result.RouteAttemptID, result.SessionID).Scan(&taskID, &attemptGeneration, &currentGeneration)
	if errors.Is(err, sql.ErrNoRows) {
		return dynamicruntime.ErrStaleQualityResult
	}
	if err != nil {
		return err
	}
	if taskID != result.TaskID {
		return dynamicruntime.ErrQualityResultOwnership
	}
	if attemptGeneration != currentGeneration {
		return dynamicruntime.ErrStaleQualityResult
	}
	var existing int
	err = tx.QueryRowContext(ctx, r.db.Rebind(`
		SELECT COUNT(*) FROM dynamic_route_quality_results
		WHERE route_attempt_id = ? AND source = ? AND source_reference = ?
	`), result.RouteAttemptID, result.Source, result.SourceReference).Scan(&existing)
	if err != nil {
		return err
	}
	if existing != 0 {
		return dynamicruntime.ErrDuplicateQualityResult
	}
	createdAt := result.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if _, err := tx.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO dynamic_route_quality_results
			(id, task_id, session_id, route_attempt_id, result, source, source_reference, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`), uuid.NewString(), result.TaskID, result.SessionID, result.RouteAttemptID,
		result.Result, result.Source, result.SourceReference, createdAt); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return dynamicruntime.ErrDuplicateQualityResult
		}
		return err
	}
	failureCategory := dynamicruntime.FailureCategory("")
	escalationReason := dynamicruntime.EscalationReason("")
	switch result.Result {
	case dynamicruntime.QualityFail:
		failureCategory = dynamicruntime.FailureCategoryQualityGate
		escalationReason = dynamicruntime.EscalationQualityFail
	case dynamicruntime.QualityEscalate:
		failureCategory = dynamicruntime.FailureCategoryQualityGate
		escalationReason = dynamicruntime.EscalationQualityEscalate
	}
	if _, err := tx.ExecContext(ctx, r.db.Rebind(`
		UPDATE dynamic_route_attempts
		SET quality_result = ?, quality_source = ?, failure_category = ?,
			escalation_reason = CASE WHEN ? <> '' THEN ? ELSE escalation_reason END
		WHERE id = ?
	`), result.Result, result.Source, failureCategory, escalationReason, escalationReason, result.RouteAttemptID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) RecordRouteAttemptFailure(ctx context.Context, sessionID string, generation int64, category dynamicruntime.FailureCategory) error {
	result, err := r.db.ExecContext(ctx, r.db.Rebind(`
		UPDATE dynamic_route_attempts SET failure_category = ?
		WHERE session_id = ? AND route_generation = ?
	`), category, sessionID, generation)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return dynamicruntime.ErrStaleGeneration
	}
	return nil
}

func (r *Repository) linkUsageEventToRouteAttemptTx(ctx context.Context, tx *sqlx.Tx, event *models.TaskUsageEvent) error {
	if event.SessionID == "" {
		return nil
	}
	var attemptID string
	var createdAt time.Time
	err := tx.QueryRowContext(ctx, r.db.Rebind(`
		SELECT a.id, a.created_at
		FROM dynamic_route_attempts a
		JOIN dynamic_route_states s ON s.session_id = a.session_id AND s.route_generation = a.route_generation
		WHERE a.session_id = ? AND a.task_id = ? AND a.usage_event_id IS NULL
	`), event.SessionID, event.TaskID).Scan(&attemptID, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	latency := event.OccurredAt.Sub(createdAt).Milliseconds()
	if latency < 0 {
		latency = 0
	}
	_, err = tx.ExecContext(ctx, r.db.Rebind(`
		UPDATE dynamic_route_attempts SET usage_event_id = ?, latency_ms = ?
		WHERE id = ? AND usage_event_id IS NULL
	`), event.UsageEventID, latency, attemptID)
	return err
}

func (r *Repository) ListTaskRoutingEvidence(ctx context.Context, taskID string) (dynamicruntime.RoutingEvidenceResult, error) {
	rows, err := r.ro.QueryxContext(ctx, r.ro.Rebind(`
		SELECT a.id, a.task_id, a.session_id, a.execution_profile_id,
			a.route_class, a.task_class, a.task_class_source, a.attempt_ordinal,
			a.escalation_reason, a.failure_category, a.quality_result, a.quality_source,
			NULLIF(u.provider, ''), NULLIF(u.model, ''),
			u.tokens_in, u.tokens_cached_read, u.tokens_cached_write, u.tokens_out,
			u.tokens_thought, u.tokens_total, u.cost_subcents, NULLIF(u.cost_source, ''), u.estimated,
			a.usage_event_id, a.latency_ms, a.created_at
		FROM dynamic_route_attempts a
		LEFT JOIN task_usage_events u ON u.usage_event_id = a.usage_event_id
		WHERE a.task_id = ?
		ORDER BY a.created_at ASC, a.id ASC
	`), taskID)
	if err != nil {
		return dynamicruntime.RoutingEvidenceResult{}, err
	}
	defer func() { _ = rows.Close() }()
	result := dynamicruntime.RoutingEvidenceResult{Attempts: make([]dynamicruntime.RoutingEvidence, 0)}
	for rows.Next() {
		attempt, scanErr := scanRoutingEvidence(rows)
		if scanErr != nil {
			return dynamicruntime.RoutingEvidenceResult{}, scanErr
		}
		result.Attempts = append(result.Attempts, attempt)
		if attempt.RouteClass == dynamicruntime.RouteClassStrong {
			result.PaidEscalationCount++
		}
	}
	return result, rows.Err()
}

func scanRoutingEvidence(rows *sqlx.Rows) (dynamicruntime.RoutingEvidence, error) {
	var evidence dynamicruntime.RoutingEvidence
	var provider, model, costSource, usageEventID sql.NullString
	var tokensIn, cachedRead, cachedWrite, tokensOut, tokensThought, tokensTotal sql.NullInt64
	var reportedCost, estimated, latency sql.NullInt64
	err := rows.Scan(
		&evidence.AttemptID, &evidence.TaskID, &evidence.SessionID, &evidence.SelectedRoute,
		&evidence.RouteClass, &evidence.TaskClass, &evidence.TaskClassSource, &evidence.AttemptOrdinal,
		&evidence.EscalationReason, &evidence.FailureCategory, &evidence.QualityResult, &evidence.QualitySource,
		&provider, &model, &tokensIn, &cachedRead, &cachedWrite, &tokensOut, &tokensThought,
		&tokensTotal, &reportedCost, &costSource, &estimated, &usageEventID, &latency, &evidence.Timestamp,
	)
	if err != nil {
		return evidence, err
	}
	evidence.Provider = nullStringPointer(provider)
	evidence.Model = nullStringPointer(model)
	evidence.TokensIn = nullInt64Pointer(tokensIn)
	evidence.TokensCachedRead = nullInt64Pointer(cachedRead)
	evidence.TokensCachedWrite = nullInt64Pointer(cachedWrite)
	evidence.TokensOut = nullInt64Pointer(tokensOut)
	evidence.TokensThought = nullInt64Pointer(tokensThought)
	evidence.TokensTotal = nullInt64Pointer(tokensTotal)
	evidence.ReportedCost = nullInt64Pointer(reportedCost)
	evidence.CostSource = nullStringPointer(costSource)
	if estimated.Valid {
		value := estimated.Int64 != 0
		evidence.Estimated = &value
	}
	evidence.UsageEventID = nullStringPointer(usageEventID)
	evidence.LatencyMS = nullInt64Pointer(latency)
	return evidence, nil
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func isTransientRouteSession(sessionID string) bool {
	return strings.HasPrefix(sessionID, "utility:")
}
