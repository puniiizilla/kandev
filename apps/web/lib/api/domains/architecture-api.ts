import { fetchBlob, fetchJson } from "../client";

export type ArchitectureDiagnostic = {
  subject?: string;
  code?: string;
  message?: string;
  supportedFixes?: string[];
};

export type ArchitectureDiagram = {
  id: string;
  title: string;
  type: string;
  source_path: string;
  validation: {
    status: string;
    diagnostics?: ArchitectureDiagnostic[] | Record<string, unknown>;
  };
};

export type ArchitectureInventory = {
  repository_id: string;
  repository: string;
  git_ref: string;
  resolved_sha: string;
  branch?: string;
  source_path: string;
  source_dirty: boolean;
  diagrams: ArchitectureDiagram[];
};

function query(workspaceId: string, revision?: string): string {
  const values = new URLSearchParams({ workspace_id: workspaceId });
  if (revision) values.set("revision", revision);
  return values.toString();
}

export function getArchitectureInventory(workspaceId: string): Promise<ArchitectureInventory> {
  return fetchJson(`/api/v1/architecture?${query(workspaceId)}`, { cache: "no-store" });
}

export function refreshArchitecture(workspaceId: string): Promise<ArchitectureInventory> {
  return fetchJson(`/api/v1/architecture/refresh?${query(workspaceId)}`, {
    cache: "no-store",
    init: { method: "POST" },
  });
}

export async function getArchitectureSource(
  workspaceId: string,
  diagramId: string,
  revision: string,
): Promise<string> {
  const body = await fetchBlob(
    `/api/v1/architecture/diagrams/${encodeURIComponent(diagramId)}/source?${query(workspaceId, revision)}`,
    { cache: "no-store" },
  );
  return body.text();
}

export function getArchitectureRenderUrl(
  workspaceId: string,
  diagramId: string,
  revision: string,
): string {
  return `/api/v1/architecture/diagrams/${encodeURIComponent(diagramId)}/render?${query(workspaceId, revision)}`;
}

export async function getArchitectureRender(
  workspaceId: string,
  diagramId: string,
  revision: string,
): Promise<string> {
  const body = await fetchBlob(getArchitectureRenderUrl(workspaceId, diagramId, revision), {
    cache: "no-store",
  });
  return body.text();
}
