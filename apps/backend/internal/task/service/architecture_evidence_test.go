package service

import (
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

func TestEvaluateArchifyRelevancePrecedence(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		paths    []string
		override *bool
		want     bool
		rule     string
	}{
		{name: "source change defeats no override", paths: []string{"docs/archify/ist/A.architecture.json"}, override: boolPtr(false), want: true, rule: "architecture_source_changed"},
		{name: "required label", labels: []string{"archify:required"}, want: true, rule: "required_label"},
		{name: "human override", override: boolPtr(true), want: true, rule: "human_override"},
		{name: "domain label", labels: []string{"runtime"}, want: true, rule: "impact_label"},
		{name: "default", want: false, rule: "no_match"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rule := EvaluateArchifyRelevance(tt.labels, tt.paths, "docs/archify/ist", tt.override)
			if got != tt.want || rule != tt.rule {
				t.Fatalf("got %v %q", got, rule)
			}
		})
	}
}

func boolPtr(value bool) *bool { return &value }

func TestArchitectureWorkspaceReadyRequiresPinnedMaterializedSession(t *testing.T) {
	session := &models.TaskSession{BaseCommitSHA: "base"}
	if architectureWorkspaceReady(session, "repo") {
		t.Fatal("session without a worktree must not admit baseline capture")
	}
	session.Worktrees = []*models.TaskEnvironmentRepo{{RepositoryID: "repo", WorktreePath: "/worktree"}}
	if !architectureWorkspaceReady(session, "repo") {
		t.Fatal("pinned materialized session should admit baseline capture")
	}
}
