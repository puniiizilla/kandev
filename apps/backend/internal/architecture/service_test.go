package architecture

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

type fakeRepositories struct{ repositories []*models.Repository }

func (f fakeRepositories) ListRepositories(context.Context, string) ([]*models.Repository, error) {
	return f.repositories, nil
}

func TestInventoryAndRenderUseCommittedSourcesAndTemporaryCache(t *testing.T) {
	repoDir := t.TempDir()
	root := filepath.Join(repoDir, "docs", "archify", "ist")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 7; index++ {
		path := filepath.Join(root, fmt.Sprintf("DIAGRAM_%d.architecture.json", index))
		if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"diagram_type":"architecture","meta":{"title":"Diagram %d"}}`, index)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "sources")
	sha := runGit(t, repoDir, "rev-parse", "HEAD")
	runtimePath := filepath.Join(t.TempDir(), "archify.mjs")
	runtime := `import fs from "node:fs"; const args=process.argv.slice(2); if(args[0]==="validate"){console.log("{}");process.exit(0)} const out=args[3]; fs.writeFileSync(out,"<html>render</html>");`
	if err := os.WriteFile(runtimePath, []byte(runtime), 0o600); err != nil {
		t.Fatal(err)
	}
	repository := &models.Repository{ID: "repo", WorkspaceID: "workspace", Name: "TheBrain", LocalPath: repoDir, ArchitectureGitRef: "HEAD", ArchitecturePath: "docs/archify/ist", ArchifyRuntime: runtimePath}
	service := NewService(fakeRepositories{[]*models.Repository{repository}}, t.TempDir())
	inventory, err := service.Inventory(context.Background(), "workspace", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Diagrams) != 7 {
		t.Fatalf("diagrams = %d, want 7", len(inventory.Diagrams))
	}
	if inventory.ResolvedSHA != sha {
		t.Fatalf("sha = %q, want %q", inventory.ResolvedSHA, sha)
	}
	checkout := filepath.Join(service.cacheRoot, repository.ID, sha, "checkout")
	if checkoutSHA := runGit(t, checkout, "rev-parse", "HEAD"); checkoutSHA != sha {
		t.Fatalf("checkout sha = %q, want %q", checkoutSHA, sha)
	}
	render, err := service.Render(context.Background(), "workspace", "DIAGRAM_0", sha, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(render) != "<html>render</html>" {
		t.Fatalf("render = %q", render)
	}
	if status := runGit(t, repoDir, "status", "--porcelain"); status != "" {
		t.Fatalf("source checkout changed: %s", status)
	}
}

func runGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func TestResolveBindingFailsClosed(t *testing.T) {
	tests := []struct {
		name  string
		repos []*models.Repository
		code  string
	}{
		{name: "missing", code: CodeBindingMissing},
		{name: "partial", repos: []*models.Repository{{ID: "one", ArchitectureGitRef: "main"}}, code: CodeBindingInvalid},
		{name: "ambiguous", repos: []*models.Repository{
			{ID: "one", LocalPath: "/one", ArchitectureGitRef: "main", ArchitecturePath: "docs/archify/ist", ArchifyRuntime: "/runtime"},
			{ID: "two", LocalPath: "/two", ArchitectureGitRef: "main", ArchitecturePath: "docs/archify/ist", ArchifyRuntime: "/runtime"},
		}, code: CodeBindingAmbiguous},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(fakeRepositories{tt.repos}, t.TempDir()).resolveBinding(context.Background(), "workspace")
			if ErrorCode(err) != tt.code {
				t.Fatalf("code = %q, want %q (%v)", ErrorCode(err), tt.code, err)
			}
		})
	}
}

func TestResolveBindingAcceptsOnlyExplicitSafeConfiguration(t *testing.T) {
	repoDir := t.TempDir()
	runtimePath := filepath.Join(t.TempDir(), "archify.mjs")
	if err := os.WriteFile(runtimePath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := &models.Repository{ID: "repo", WorkspaceID: "workspace", LocalPath: repoDir,
		ArchitectureGitRef: "refs/heads/main", ArchitecturePath: "docs/archify/ist", ArchifyRuntime: runtimePath}
	binding, err := NewService(fakeRepositories{[]*models.Repository{repo}}, t.TempDir()).resolveBinding(context.Background(), "workspace")
	if err != nil {
		t.Fatal(err)
	}
	if binding.Repository.ID != "repo" {
		t.Fatalf("repository = %q", binding.Repository.ID)
	}

	repo.ArchitecturePath = "../escape"
	_, err = NewService(fakeRepositories{[]*models.Repository{repo}}, t.TempDir()).resolveBinding(context.Background(), "workspace")
	if ErrorCode(err) != CodePathInvalid {
		t.Fatalf("code = %q, want %q", ErrorCode(err), CodePathInvalid)
	}
}

func TestCaptureAndCompareAreRevisionExplicit(t *testing.T) {
	repoDir := t.TempDir()
	root := filepath.Join(repoDir, "docs", "archify", "ist")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "SYSTEM_OVERVIEW.architecture.json")
	if err := os.WriteFile(sourcePath, []byte(`{"diagram_type":"architecture","meta":{"title":"Before"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "before")
	base := runGit(t, repoDir, "rev-parse", "HEAD")
	if err := os.WriteFile(sourcePath, []byte(`{"diagram_type":"architecture","meta":{"title":"After"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "after")
	head := runGit(t, repoDir, "rev-parse", "HEAD")
	runtimePath := filepath.Join(t.TempDir(), "archify.mjs")
	runtime := `import fs from "node:fs"; const a=process.argv.slice(2); if(a[0]==="validate"){console.log("{}");process.exit(0)} if(a[0]==="render"){fs.writeFileSync(a[3],"<html/>");process.exit(0)} if(a[0]==="compare"){const out=a[4], receipt=a[a.indexOf("--receipt")+1];fs.writeFileSync(out,"<html>delta</html>");fs.writeFileSync(receipt,JSON.stringify({semanticChange:true,validation:{status:"valid"},changes:[{kind:"component.changed"}]}));console.log("{}");}`
	if err := os.WriteFile(runtimePath, []byte(runtime), 0o600); err != nil {
		t.Fatal(err)
	}
	repository := &models.Repository{ID: "repo", WorkspaceID: "workspace", Name: "repo", LocalPath: repoDir, ArchitectureGitRef: "HEAD", ArchitecturePath: "docs/archify/ist", ArchifyRuntime: runtimePath}
	service := NewService(fakeRepositories{[]*models.Repository{repository}}, t.TempDir())
	baseCapture, err := service.CaptureRevision(context.Background(), "workspace", base)
	if err != nil {
		t.Fatal(err)
	}
	afterCapture, err := service.CaptureRevision(context.Background(), "workspace", head)
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := service.CompareRevisions(context.Background(), "workspace", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if baseCapture.SHA != base || afterCapture.SHA != head || !comparison.ArchitectureChanged || !comparison.SemanticChange {
		t.Fatalf("unexpected captures: %#v %#v %#v", baseCapture, afterCapture, comparison)
	}
	var receipt struct {
		SourceHashes map[string]string `json:"source_hashes"`
	}
	if err := json.Unmarshal(baseCapture.Receipt, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.SourceHashes["SYSTEM_OVERVIEW"] == "" {
		t.Fatalf("receipt does not pin source hashes: %s", baseCapture.Receipt)
	}
	if status := runGit(t, repoDir, "status", "--porcelain"); status != "" {
		t.Fatalf("repository mutated: %s", status)
	}
}
