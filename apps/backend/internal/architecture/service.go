package architecture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kandev/kandev/internal/common/subproc"
	"github.com/kandev/kandev/internal/task/models"
)

const (
	CodeBindingMissing   = "ARCHITECTURE_BINDING_MISSING"
	CodeBindingInvalid   = "ARCHITECTURE_BINDING_INVALID"
	CodeBindingAmbiguous = "ARCHITECTURE_BINDING_AMBIGUOUS"
	CodePathInvalid      = "ARCHITECTURE_PATH_INVALID"
	CodeRefInvalid       = "ARCHITECTURE_REF_INVALID"
	CodeRuntimeInvalid   = "ARCHIFY_RUNTIME_INVALID"
	CodeDiagramNotFound  = "ARCHITECTURE_DIAGRAM_NOT_FOUND"
	CodeOperationFailed  = "ARCHIFY_OPERATION_FAILED"
)

var diagramIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type codedError struct {
	Code string
	Err  error
}

func (e *codedError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *codedError) Unwrap() error { return e.Err }
func ErrorCode(err error) string {
	var target *codedError
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}
func fail(code, message string) error { return &codedError{Code: code, Err: errors.New(message)} }

type RepositoryLister interface {
	ListRepositories(context.Context, string) ([]*models.Repository, error)
}

type Service struct {
	repositories RepositoryLister
	cacheRoot    string
	cacheMu      sync.Mutex
}
type Binding struct {
	Repository *models.Repository
	SourceRoot string
}

type Diagram struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Type       string     `json:"type"`
	SourcePath string     `json:"source_path"`
	Validation Validation `json:"validation"`
}
type Validation struct {
	Status      string          `json:"status"`
	Diagnostics json.RawMessage `json:"diagnostics,omitempty"`
}
type Inventory struct {
	RepositoryID string    `json:"repository_id"`
	Repository   string    `json:"repository"`
	GitRef       string    `json:"git_ref"`
	ResolvedSHA  string    `json:"resolved_sha"`
	Branch       string    `json:"branch,omitempty"`
	SourcePath   string    `json:"source_path"`
	SourceDirty  bool      `json:"source_dirty"`
	Diagrams     []Diagram `json:"diagrams"`
}

func NewService(repositories RepositoryLister, cacheRoot string) *Service {
	if cacheRoot == "" {
		cacheRoot = filepath.Join(os.TempDir(), "kandev-archify")
	}
	return &Service{repositories: repositories, cacheRoot: cacheRoot}
}

func (s *Service) resolveBinding(ctx context.Context, workspaceID string) (*Binding, error) {
	if strings.TrimSpace(workspaceID) == "" {
		return nil, fail(CodeBindingMissing, "workspace_id is required")
	}
	repositories, err := s.repositories.ListRepositories(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	var complete []*models.Repository
	partial := false
	for _, repository := range repositories {
		set := configuredBindingFields(repository)
		if set == bindingFieldCount {
			complete = append(complete, repository)
		} else if set != 0 {
			partial = true
		}
	}
	if len(complete) > 1 {
		return nil, fail(CodeBindingAmbiguous, "more than one repository has an Archify binding")
	}
	if len(complete) == 0 {
		if partial {
			return nil, fail(CodeBindingInvalid, "Archify binding is incomplete")
		}
		return nil, fail(CodeBindingMissing, "Archify binding is not configured")
	}
	return validateBinding(complete[0])
}

func configuredBindingFields(repository *models.Repository) int {
	fields := []string{repository.LocalPath, repository.ArchitectureGitRef, repository.ArchitecturePath, repository.ArchifyRuntime}
	set := 0
	for _, field := range fields {
		if strings.TrimSpace(field) != "" {
			set++
		}
	}
	return set
}

func validateBinding(repository *models.Repository) (*Binding, error) {
	if !filepath.IsAbs(repository.LocalPath) {
		return nil, fail(CodeBindingInvalid, "repository path must be absolute")
	}
	if strings.HasPrefix(repository.ArchitectureGitRef, "-") || strings.ContainsAny(repository.ArchitectureGitRef, "\x00\r\n") {
		return nil, fail(CodeRefInvalid, "configured Git ref is unsafe")
	}
	path := filepath.Clean(filepath.FromSlash(repository.ArchitecturePath))
	if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return nil, fail(CodePathInvalid, "architecture path must remain repository-relative")
	}
	runtimeInfo, err := os.Lstat(repository.ArchifyRuntime)
	if err != nil || !filepath.IsAbs(repository.ArchifyRuntime) || runtimeInfo.Mode()&os.ModeSymlink != 0 || !runtimeInfo.Mode().IsRegular() {
		return nil, fail(CodeRuntimeInvalid, "Archify runtime must be an existing absolute regular file")
	}
	resolvedRuntime, err := filepath.EvalSymlinks(repository.ArchifyRuntime)
	if err != nil || resolvedRuntime != filepath.Clean(repository.ArchifyRuntime) {
		return nil, fail(CodeRuntimeInvalid, "Archify runtime must not traverse symlinks")
	}
	return &Binding{Repository: repository, SourceRoot: filepath.ToSlash(path)}, nil
}

const bindingFieldCount = 4

func gitOutput(ctx context.Context, repositoryPath string, args ...string) ([]byte, error) {
	argv := append([]string{"-C", repositoryPath}, args...)
	return subproc.RunGitOutputClass(ctx, subproc.GitInteractive, subproc.NewGitCommand(ctx, argv...))
}

func runGitCommand(ctx context.Context, args ...string) error {
	return subproc.RunGitClass(ctx, subproc.GitInteractive, subproc.NewGitCommand(ctx, args...))
}

func (s *Service) Inventory(ctx context.Context, workspaceID string, refresh bool) (*Inventory, error) {
	binding, err := s.resolveBinding(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	shaBytes, err := gitOutput(ctx, binding.Repository.LocalPath, "rev-parse", "--verify", "--end-of-options", binding.Repository.ArchitectureGitRef+"^{commit}")
	if err != nil {
		return nil, fail(CodeRefInvalid, "configured Git ref does not resolve to a commit")
	}
	sha := strings.TrimSpace(string(shaBytes))
	if err := s.prepareCache(binding.Repository.ID, sha, refresh); err != nil {
		return nil, err
	}
	filesBytes, err := gitOutput(ctx, binding.Repository.LocalPath, "ls-tree", "-r", "--name-only", sha, "--", binding.SourceRoot)
	if err != nil {
		return nil, fail(CodeOperationFailed, "cannot enumerate architecture sources")
	}
	statusBytes, _ := gitOutput(ctx, binding.Repository.LocalPath, "status", "--porcelain", "--", binding.SourceRoot)
	result := &Inventory{RepositoryID: binding.Repository.ID, Repository: binding.Repository.Name, GitRef: binding.Repository.ArchitectureGitRef, ResolvedSHA: sha, SourcePath: binding.SourceRoot, SourceDirty: len(statusBytes) != 0}
	for _, path := range strings.Fields(string(filesBytes)) {
		if !strings.HasSuffix(path, ".architecture.json") {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(path), ".architecture.json")
		source, sourceErr := s.sourceAt(ctx, binding, sha, path)
		if sourceErr != nil {
			return nil, sourceErr
		}
		var metadata struct {
			DiagramType string `json:"diagram_type"`
			Meta        struct {
				Title string `json:"title"`
			} `json:"meta"`
		}
		validation := Validation{Status: "valid"}
		if json.Unmarshal(source, &metadata) != nil {
			validation.Status = "invalid_json"
		} else {
			validation = s.validate(ctx, binding, sha, id, source)
		}
		if metadata.Meta.Title == "" {
			metadata.Meta.Title = id
		}
		if metadata.DiagramType == "" {
			metadata.DiagramType = "architecture"
		}
		result.Diagrams = append(result.Diagrams, Diagram{ID: id, Title: metadata.Meta.Title, Type: metadata.DiagramType, SourcePath: path, Validation: validation})
	}
	sort.Slice(result.Diagrams, func(i, j int) bool { return result.Diagrams[i].ID < result.Diagrams[j].ID })
	return result, nil
}

func (s *Service) prepareCache(repositoryID, sha string, refresh bool) error {
	repositoryCache := filepath.Join(s.cacheRoot, repositoryID)
	if refresh {
		if err := os.RemoveAll(repositoryCache); err != nil {
			return fail(CodeOperationFailed, "cannot refresh render cache")
		}
		return nil
	}
	entries, err := os.ReadDir(repositoryCache)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != sha {
			_ = os.RemoveAll(filepath.Join(repositoryCache, entry.Name()))
		}
	}
	return nil
}

func (s *Service) validate(ctx context.Context, binding *Binding, sha, diagramID string, source []byte) Validation {
	checkout, err := s.immutableCheckout(ctx, binding, sha)
	if err != nil {
		return Validation{Status: "error"}
	}
	dir := filepath.Join(s.cacheRoot, binding.Repository.ID, sha)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Validation{Status: "error"}
	}
	sourceFile := filepath.Join(dir, diagramID+".architecture.json")
	if err := os.WriteFile(sourceFile, source, 0o600); err != nil {
		return Validation{Status: "error"}
	}
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "node", binding.Repository.ArchifyRuntime, "validate", "architecture", sourceFile, "--quality", "showcase", "--repo-root", checkout, "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if !json.Valid(output) {
			output, _ = json.Marshal(map[string]string{"message": strings.TrimSpace(string(output))})
		}
		return Validation{Status: "invalid", Diagnostics: output}
	}
	if json.Valid(output) {
		return Validation{Status: "valid", Diagnostics: output}
	}
	return Validation{Status: "valid"}
}

func (s *Service) immutableCheckout(ctx context.Context, binding *Binding, sha string) (string, error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	root := filepath.Join(s.cacheRoot, binding.Repository.ID, sha, "checkout")
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		return root, nil
	}
	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(parent, ".checkout-")
	if err != nil {
		return "", err
	}
	if err := os.Remove(staging); err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := runGitCommand(ctx, "clone", "--no-checkout", "--no-hardlinks", "--local", "--", binding.Repository.LocalPath, staging); err != nil {
		return "", err
	}
	if err := runGitCommand(ctx, "-C", staging, "checkout", "--detach", sha, "--"); err != nil {
		return "", err
	}
	if err := os.Rename(staging, root); err != nil {
		return "", err
	}
	return root, nil
}

func (s *Service) sourceAt(ctx context.Context, binding *Binding, sha, path string) ([]byte, error) {
	if !strings.HasPrefix(path, binding.SourceRoot+"/") {
		return nil, fail(CodePathInvalid, "source is outside configured architecture path")
	}
	bytes, err := gitOutput(ctx, binding.Repository.LocalPath, "show", sha+":"+path)
	if err != nil {
		return nil, fail(CodeDiagramNotFound, "diagram source is unavailable at resolved revision")
	}
	return bytes, nil
}

func (s *Service) Source(ctx context.Context, workspaceID, diagramID, revision string) ([]byte, error) {
	if !diagramIDPattern.MatchString(diagramID) || revision == "" {
		return nil, fail(CodeDiagramNotFound, "diagram and revision are required")
	}
	binding, err := s.resolveBinding(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	resolved, err := gitOutput(ctx, binding.Repository.LocalPath, "rev-parse", "--verify", "--end-of-options", binding.Repository.ArchitectureGitRef+"^{commit}")
	if err != nil || strings.TrimSpace(string(resolved)) != revision {
		return nil, fail(CodeRefInvalid, "revision does not match configured Git ref")
	}
	return s.sourceAt(ctx, binding, revision, binding.SourceRoot+"/"+diagramID+".architecture.json")
}

func (s *Service) Render(ctx context.Context, workspaceID, diagramID, revision string, refresh bool) ([]byte, error) {
	binding, err := s.resolveBinding(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	source, err := s.Source(ctx, workspaceID, diagramID, revision)
	if err != nil {
		return nil, err
	}
	checkout, err := s.immutableCheckout(ctx, binding, revision)
	if err != nil {
		return nil, &codedError{Code: CodeOperationFailed, Err: fmt.Errorf("cannot materialize immutable revision: %w", err)}
	}
	dir := filepath.Join(s.cacheRoot, binding.Repository.ID, revision)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fail(CodeOperationFailed, "cannot create render cache")
	}
	output := filepath.Join(dir, diagramID+".html")
	if !refresh {
		if cached, readErr := os.ReadFile(output); readErr == nil {
			return cached, nil
		}
	}
	sourceFile := filepath.Join(dir, diagramID+".architecture.json")
	if err := os.WriteFile(sourceFile, source, 0o600); err != nil {
		return nil, fail(CodeOperationFailed, "cannot stage source in render cache")
	}
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "node", binding.Repository.ArchifyRuntime, "render", "architecture", sourceFile, output, "--quality", "showcase", "--repo-root", checkout)
	if bytes, runErr := cmd.CombinedOutput(); runErr != nil {
		_ = os.Remove(output)
		return nil, &codedError{Code: CodeOperationFailed, Err: fmt.Errorf("render failed: %s", strings.TrimSpace(string(bytes)))}
	}
	return os.ReadFile(output)
}
