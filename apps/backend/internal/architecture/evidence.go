package architecture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RevisionCapture struct {
	SHA             string            `json:"sha"`
	RuntimeContract string            `json:"runtime_contract"`
	SourceHashes    map[string]string `json:"source_hashes"`
	Receipt         json.RawMessage   `json:"receipt"`
	Hash            string            `json:"hash"`
	Validation      string            `json:"validation"`
}

type RevisionComparison struct {
	ArchitectureChanged bool            `json:"architecture_changed"`
	SemanticChange      bool            `json:"semantic_change"`
	AffectedDiagrams    []string        `json:"affected_diagrams"`
	Summary             json.RawMessage `json:"summary"`
	Receipt             json.RawMessage `json:"receipt"`
	Hash                string          `json:"hash"`
	CacheKey            string          `json:"cache_key"`
}

func (s *Service) CaptureRevision(ctx context.Context, workspaceID, revision string) (*RevisionCapture, error) {
	binding, sha, err := s.bindingAt(ctx, workspaceID, revision)
	if err != nil {
		return nil, err
	}
	paths, err := s.sourcePathsAt(ctx, binding, sha)
	if err != nil {
		return nil, err
	}
	result := &RevisionCapture{SHA: sha, SourceHashes: map[string]string{}, Validation: "PASS"}
	runtimeBytes, err := os.ReadFile(binding.Repository.ArchifyRuntime)
	if err != nil {
		return nil, fail(CodeRuntimeInvalid, "cannot read Archify runtime")
	}
	runtimeHash := sha256.Sum256(runtimeBytes)
	result.RuntimeContract = hex.EncodeToString(runtimeHash[:])
	receipts := make(map[string]json.RawMessage, len(paths))
	for _, path := range paths {
		source, sourceErr := s.sourceAt(ctx, binding, sha, path)
		if sourceErr != nil {
			return nil, sourceErr
		}
		hash := sha256.Sum256(source)
		result.SourceHashes[diagramID(path)] = hex.EncodeToString(hash[:])
		validation := s.validate(ctx, binding, sha, diagramID(path), source)
		if validation.Status != "valid" {
			return nil, fail(CodeOperationFailed, "Archify validation failed")
		}
		receipts[diagramID(path)] = validation.Diagnostics
		if _, renderErr := s.renderAt(ctx, binding, diagramID(path), sha, source, true); renderErr != nil {
			return nil, renderErr
		}
	}
	result.Receipt, _ = json.Marshal(map[string]any{
		"source_hashes": result.SourceHashes,
		"validation":    receipts,
	})
	h := sha256.Sum256(result.Receipt)
	result.Hash = hex.EncodeToString(h[:])
	return result, nil
}

func (s *Service) CompareRevisions(ctx context.Context, workspaceID, baseRevision, headRevision string) (*RevisionComparison, error) {
	binding, baseSHA, err := s.bindingAt(ctx, workspaceID, baseRevision)
	if err != nil {
		return nil, err
	}
	_, headSHA, err := s.bindingAt(ctx, workspaceID, headRevision)
	if err != nil {
		return nil, err
	}
	basePaths, err := s.sourcePathsAt(ctx, binding, baseSHA)
	if err != nil {
		return nil, err
	}
	headPaths, err := s.sourcePathsAt(ctx, binding, headSHA)
	if err != nil {
		return nil, err
	}
	baseSet, headSet := pathMap(basePaths), pathMap(headPaths)
	result := &RevisionComparison{}
	receipts := map[string]json.RawMessage{}
	for id := range unionIDs(baseSet, headSet) {
		basePath, inBase := baseSet[id]
		headPath, inHead := headSet[id]
		if !inBase || !inHead {
			result.ArchitectureChanged = true
			result.SemanticChange = true
			result.AffectedDiagrams = append(result.AffectedDiagrams, id)
			continue
		}
		baseSource, _ := s.sourceAt(ctx, binding, baseSHA, basePath)
		headSource, _ := s.sourceAt(ctx, binding, headSHA, headPath)
		if string(baseSource) == string(headSource) {
			continue
		}
		receipt, compareErr := s.compareDiagram(ctx, binding, baseSHA, headSHA, id, baseSource, headSource)
		if compareErr != nil {
			return nil, compareErr
		}
		result.ArchitectureChanged = true
		result.AffectedDiagrams = append(result.AffectedDiagrams, id)
		receipts[id] = receipt
		var projected struct {
			SemanticChange bool `json:"semanticChange"`
		}
		if json.Unmarshal(receipt, &projected) != nil {
			return nil, fail(CodeOperationFailed, "Archify compare receipt is invalid")
		}
		result.SemanticChange = result.SemanticChange || projected.SemanticChange
	}
	sort.Strings(result.AffectedDiagrams)
	result.Receipt, _ = json.Marshal(receipts)
	result.Summary, _ = json.Marshal(map[string]any{"affected_diagrams": result.AffectedDiagrams})
	h := sha256.Sum256(result.Receipt)
	result.Hash = hex.EncodeToString(h[:])
	result.CacheKey = filepath.Join(binding.Repository.ID, baseSHA+"-"+headSHA)
	return result, nil
}

func (s *Service) bindingAt(ctx context.Context, workspaceID, revision string) (*Binding, string, error) {
	binding, err := s.resolveBinding(ctx, workspaceID)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(revision) == "" {
		return nil, "", fail(CodeRefInvalid, "revision is required")
	}
	resolved, err := gitOutput(ctx, binding.Repository.LocalPath, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return nil, "", fail(CodeRefInvalid, "revision does not resolve to a commit")
	}
	return binding, strings.TrimSpace(string(resolved)), nil
}

func (s *Service) sourcePathsAt(ctx context.Context, binding *Binding, sha string) ([]string, error) {
	output, err := gitOutput(ctx, binding.Repository.LocalPath, "ls-tree", "-r", "--name-only", sha, "--", binding.SourceRoot)
	if err != nil {
		return nil, fail(CodeOperationFailed, "cannot enumerate architecture sources")
	}
	var paths []string
	for _, path := range strings.Fields(string(output)) {
		if strings.HasSuffix(path, ".architecture.json") {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func diagramID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".architecture.json")
}
func pathMap(paths []string) map[string]string {
	out := map[string]string{}
	for _, path := range paths {
		out[diagramID(path)] = path
	}
	return out
}
func unionIDs(a, b map[string]string) map[string]struct{} {
	out := map[string]struct{}{}
	for id := range a {
		out[id] = struct{}{}
	}
	for id := range b {
		out[id] = struct{}{}
	}
	return out
}

func (s *Service) renderAt(ctx context.Context, binding *Binding, id, sha string, source []byte, refresh bool) ([]byte, error) {
	checkout, err := s.immutableCheckout(ctx, binding, sha)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(s.cacheRoot, binding.Repository.ID, sha)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	input, output := filepath.Join(dir, id+".architecture.json"), filepath.Join(dir, id+".html")
	if !refresh {
		if cached, readErr := os.ReadFile(output); readErr == nil {
			return cached, nil
		}
	}
	if err := os.WriteFile(input, source, 0o600); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	bytes, err := exec.CommandContext(runCtx, "node", binding.Repository.ArchifyRuntime, "render", "architecture", input, output, "--quality", "showcase", "--repo-root", checkout).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("render failed: %s", strings.TrimSpace(string(bytes)))
	}
	return os.ReadFile(output)
}

func (s *Service) compareDiagram(ctx context.Context, binding *Binding, baseSHA, headSHA, id string, base, head []byte) (json.RawMessage, error) {
	dir := filepath.Join(s.cacheRoot, binding.Repository.ID, baseSHA+"-"+headSHA, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	baseFile, headFile, output, receipt := filepath.Join(dir, "base.json"), filepath.Join(dir, "head.json"), filepath.Join(dir, "delta.html"), filepath.Join(dir, "receipt.json")
	if err := os.WriteFile(baseFile, base, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(headFile, head, 0o600); err != nil {
		return nil, err
	}
	checkout, err := s.immutableCheckout(ctx, binding, headSHA)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	bytes, err := exec.CommandContext(runCtx, "node", binding.Repository.ArchifyRuntime, "compare", "architecture", baseFile, headFile, output, "--receipt", receipt, "--json", "--quality", "showcase", "--repo-root", checkout).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compare failed: %s", strings.TrimSpace(string(bytes)))
	}
	body, err := os.ReadFile(receipt)
	if err != nil || !json.Valid(body) {
		return nil, fail(CodeOperationFailed, "Archify compare receipt is unavailable")
	}
	return body, nil
}

func (s *Service) Delta(ctx context.Context, workspaceID, baseSHA, headSHA, id string) ([]byte, error) {
	if !diagramIDPattern.MatchString(id) {
		return nil, fail(CodeDiagramNotFound, "invalid diagram ID")
	}
	if _, err := s.CompareRevisions(ctx, workspaceID, baseSHA, headSHA); err != nil {
		return nil, err
	}
	binding, _, err := s.bindingAt(ctx, workspaceID, headSHA)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(s.cacheRoot, binding.Repository.ID, baseSHA+"-"+headSHA, id, "delta.html"))
	if err != nil {
		return nil, fail(CodeDiagramNotFound, "delta is unavailable")
	}
	return body, nil
}
