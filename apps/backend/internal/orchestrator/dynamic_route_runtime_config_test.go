package orchestrator

import (
	"context"
	"testing"

	agentruntime "github.com/kandev/kandev/internal/agent/runtime"
	agentsettingsmodels "github.com/kandev/kandev/internal/agent/settings/models"
	"github.com/kandev/kandev/internal/task/models"
	"github.com/stretchr/testify/require"
)

func TestApplyResolvedExecutionClearsPredecessorRuntimeConfiguration(t *testing.T) {
	session := &models.TaskSession{
		ExecutionProfileID: "free-profile",
		RouteState:         "starting",
		Metadata: map[string]interface{}{
			models.SessionMetaKeySessionMode:            "default",
			models.SessionMetaKeyRuntimeConfig:          models.SessionRuntimeConfig{Model: "local-model"},
			models.SessionMetaKeyRuntimeConfigOverrides: models.SessionRuntimeConfig{Model: "local-model"},
			models.SessionMetaKeyACPConfigBaseline:      map[string]string{"model": "local-model"},
			models.SessionMetaKeyACPModelState:          map[string]string{"current_value": "local-model"},
			models.SessionMetaKeyContextWindow:          map[string]int{"used": 1},
			"unrelated":                                 "preserved",
		},
	}

	applyResolvedExecution(session, agentruntime.ProfileExecution{
		ExecutionProfileID: "free-profile",
		Profile: &agentsettingsmodels.AgentProfile{
			ID:    "free-profile",
			Model: "free-model",
		},
	})

	for _, key := range []string{
		models.SessionMetaKeySessionMode,
		models.SessionMetaKeyRuntimeConfig,
		models.SessionMetaKeyRuntimeConfigOverrides,
		models.SessionMetaKeyACPConfigBaseline,
		models.SessionMetaKeyACPModelState,
		models.SessionMetaKeyContextWindow,
	} {
		require.NotContains(t, session.Metadata, key)
	}
	require.Equal(t, "preserved", session.Metadata["unrelated"])
	effective, ok := models.LoadEffectiveSessionRuntimeConfig(session)
	require.True(t, ok)
	require.Equal(t, "free-model", effective.Model)
}

func TestClearDynamicRelaunchRuntimeConfigurationPersistsAfterPredecessorStops(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task-runtime-clear", "session-runtime-clear", models.TaskSessionStateWaitingForInput)
	require.NoError(t, repo.SetSessionMetadataKey(ctx, "session-runtime-clear", models.SessionMetaKeyRuntimeConfig,
		models.SessionRuntimeConfig{Model: "local-model"}))
	require.NoError(t, repo.SetSessionMetadataKey(ctx, "session-runtime-clear", models.SessionMetaKeyRuntimeConfigOverrides,
		models.SessionRuntimeConfig{Model: "local-model"}))

	svc := &Service{repo: repo}
	require.True(t, svc.clearDynamicRelaunchRuntimeConfiguration(ctx, "session-runtime-clear"))

	updated, err := repo.GetTaskSession(ctx, "session-runtime-clear")
	require.NoError(t, err)
	_, hasRuntime := models.LoadSessionRuntimeConfig(updated.Metadata)
	_, hasOverrides := models.LoadSessionRuntimeConfigOverrides(updated.Metadata)
	require.False(t, hasRuntime)
	require.False(t, hasOverrides)
}
