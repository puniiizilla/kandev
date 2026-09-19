import { fetchJson } from "../client";

export type ArchitectureEvidence = {
  task_id?: string;
  state: string;
  required: boolean;
  validation_status?: string;
  architecture_changed: boolean;
  semantic_change: boolean;
  base_sha?: string;
  head_sha?: string;
  affected_diagrams_json?: string;
  diff_summary_json?: string;
  diagnostics_json?: string;
};

const endpoint = (taskId: string) =>
  `/api/v1/tasks/${encodeURIComponent(taskId)}/architecture-evidence`;

export function getArchitectureEvidence(taskId: string): Promise<ArchitectureEvidence> {
  return fetchJson(endpoint(taskId), { cache: "no-store", init: { method: "GET" } });
}

export function refreshArchitectureEvidence(taskId: string): Promise<ArchitectureEvidence> {
  return fetchJson(`${endpoint(taskId)}/refresh`, { cache: "no-store", init: { method: "POST" } });
}

export function architectureEvidenceDeltaUrl(taskId: string, diagramId: string): string {
  return `${endpoint(taskId)}/delta/${encodeURIComponent(diagramId)}`;
}
