package sqlite

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

func TestRepositoryArchitectureBindingRoundTrip(t *testing.T) {
	repositoryStore := newRepoForEntityTests(t)
	seedWorkspace(t, repositoryStore, "workspace")
	want := &models.Repository{ID: "repository", WorkspaceID: "workspace", Name: "TheBrain", ArchitectureGitRef: "refs/heads/main", ArchitecturePath: "docs/archify/ist", ArchifyRuntime: "/opt/archify.mjs"}
	if err := repositoryStore.CreateRepository(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := repositoryStore.GetRepository(context.Background(), want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ArchitectureGitRef != want.ArchitectureGitRef || got.ArchitecturePath != want.ArchitecturePath || got.ArchifyRuntime != want.ArchifyRuntime {
		t.Fatalf("binding did not round-trip: %#v", got)
	}
}
