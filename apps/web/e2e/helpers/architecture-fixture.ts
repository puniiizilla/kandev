import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import type { BackendContext } from "../fixtures/backend";
import type { SeedData } from "../fixtures/test-base";
import type { ApiClient } from "./api-client";
import { makeGitEnv } from "./git-helper";

export const ARCHITECTURE_DIAGRAMS = [
  "SYSTEM_OVERVIEW",
  "MODULE_DEPENDENCIES",
  "DATA_FLOW",
  "READER_WRITER_AUTHORITY",
  "RUNTIME_TOPOLOGY",
  "DATABASE_PERSISTENCE",
  "END_TO_END_FLOWS",
] as const;

const SOURCE_ROOT = "docs/archify/ist";

export type ArchitectureFixture = {
  runtimePath: string;
  sha: string;
  statusBefore: string;
  treeBefore: string;
};

function git(seed: SeedData, backend: BackendContext, ...args: string[]): string {
  return execFileSync("git", ["-C", seed.repositoryPath, ...args], {
    env: makeGitEnv(backend.tmpDir),
    encoding: "utf8",
  }).trim();
}

function source(diagram: string, invalid = false): string {
  return `${JSON.stringify(
    {
      meta: { title: diagram.replaceAll("_", " ") },
      diagram_type: "architecture",
      invalid,
      nodes: [{ id: "node", label: diagram }],
      connections: [],
    },
    null,
    2,
  )}\n`;
}

function writeRuntime(runtimePath: string) {
  fs.writeFileSync(
    runtimePath,
    `#!/usr/bin/env node
const fs = require("node:fs");
const [, , command, kind, sourcePath, outputPath] = process.argv;
const document = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
if (command === "validate") {
  if (document.invalid) {
    process.stdout.write(JSON.stringify({ errors: [{ element: "document", code: "fixture_invalid", message: "Fixture validation failed", correction: "Set invalid to false" }] }));
    process.exit(1);
  }
  process.stdout.write(JSON.stringify({ checks: [], valid: true }));
  process.exit(0);
}
if (command === "render" && kind === "architecture") {
  if (document.invalid) {
    process.stderr.write("render blocked by validation");
    process.exit(1);
  }
  fs.writeFileSync(outputPath, "<!doctype html><html><body><h1>" + document.meta.title + "</h1><p>Deterministic Archify render</p></body></html>");
  process.exit(0);
}
if (command === "compare" && kind === "architecture") {
  const headPath = outputPath;
  const deltaPath = process.argv[6];
  const receiptPath = process.argv[process.argv.indexOf("--receipt") + 1];
  const head = JSON.parse(fs.readFileSync(headPath, "utf8"));
  const changed = JSON.stringify(document) !== JSON.stringify(head);
  fs.writeFileSync(deltaPath, "<!doctype html><html><body><h1>Architecture delta</h1></body></html>");
  fs.writeFileSync(receiptPath, JSON.stringify({ semanticChange: changed, validation: { status: "valid" }, changes: changed ? [{ kind: "component.changed" }] : [] }));
  process.stdout.write("{}");
  process.exit(0);
}
process.stderr.write("unsupported fixture command");
process.exit(2);
`,
    { mode: 0o755 },
  );
}

export async function prepareArchitectureFixture(
  apiClient: ApiClient,
  seed: SeedData,
  backend: BackendContext,
  invalidDiagram?: string,
): Promise<ArchitectureFixture> {
  const sourceRoot = path.join(seed.repositoryPath, SOURCE_ROOT);
  fs.mkdirSync(sourceRoot, { recursive: true });
  for (const diagram of ARCHITECTURE_DIAGRAMS) {
    fs.writeFileSync(
      path.join(sourceRoot, `${diagram}.architecture.json`),
      source(diagram, diagram === invalidDiagram),
    );
  }
  const runtimePath = path.join(backend.tmpDir, "archify-e2e-runtime.cjs");
  writeRuntime(runtimePath);
  git(seed, backend, "add", SOURCE_ROOT);
  git(seed, backend, "commit", "-m", "test: add architecture fixture");
  const sha = git(seed, backend, "rev-parse", "HEAD");
  await apiClient.updateRepository(seed.repositoryId, {
    architecture_git_ref: "HEAD",
    architecture_path: SOURCE_ROOT,
    archify_runtime: runtimePath,
  });
  return {
    runtimePath,
    sha,
    statusBefore: git(seed, backend, "status", "--porcelain=v1"),
    treeBefore: git(seed, backend, "write-tree"),
  };
}

export function correctArchitectureFixture(
  seed: SeedData,
  backend: BackendContext,
  diagram: string,
): string {
  fs.writeFileSync(
    path.join(seed.repositoryPath, SOURCE_ROOT, `${diagram}.architecture.json`),
    source(diagram),
  );
  git(seed, backend, "add", SOURCE_ROOT);
  git(seed, backend, "commit", "-m", "test: correct architecture fixture");
  return git(seed, backend, "rev-parse", "HEAD");
}

export async function cleanupArchitectureFixture(
  apiClient: ApiClient,
  seed: SeedData,
  backend: BackendContext,
) {
  await apiClient.updateRepository(seed.repositoryId, {
    architecture_git_ref: "",
    architecture_path: "",
    archify_runtime: "",
  });
  git(seed, backend, "reset", "--hard", seed.repositoryBaselineOID);
  git(seed, backend, "clean", "-fd", "--", SOURCE_ROOT);
}

export function repositoryEvidence(seed: SeedData, backend: BackendContext) {
  return {
    status: git(seed, backend, "status", "--porcelain=v1"),
    tree: git(seed, backend, "write-tree"),
  };
}
