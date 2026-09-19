import { useCallback, useEffect, useState } from "react";
import {
  getArchitectureInventory,
  getArchitectureRender,
  getArchitectureSource,
  refreshArchitecture,
  type ArchitectureInventory,
} from "@/lib/api/domains/architecture-api";

export type ArchitectureView = "render" | "source" | "validation";

function errorMessage(error: unknown): string {
  if (!error || typeof error !== "object" || !("body" in error)) return "";
  const body = error.body;
  if (!body || typeof body !== "object" || !("error" in body)) return "";
  const detail = body.error;
  return detail &&
    typeof detail === "object" &&
    "message" in detail &&
    typeof detail.message === "string"
    ? detail.message
    : "";
}

export function useArchitectureBrowser(workspaceId?: string) {
  const [inventory, setInventory] = useState<ArchitectureInventory | null>(null);
  const [selectedId, setSelectedId] = useState("");
  const [view, setView] = useState<ArchitectureView>("render");
  const [source, setSource] = useState("");
  const [render, setRender] = useState("");
  const [renderError, setRenderError] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");

  const acceptInventory = useCallback((next: ArchitectureInventory) => {
    setInventory(next);
    setSelectedId((current) =>
      next.diagrams.some((diagram) => diagram.id === current)
        ? current
        : (next.diagrams[0]?.id ?? ""),
    );
    setError("");
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    if (!workspaceId) {
      setError("workspace_required");
      setLoading(false);
      return;
    }
    try {
      acceptInventory(await getArchitectureInventory(workspaceId));
    } catch (caught) {
      setError(errorMessage(caught) || "request_failed");
    } finally {
      setLoading(false);
    }
  }, [acceptInventory, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    setSource("");
    setRender("");
    setRenderError(false);
    if (!workspaceId || !inventory || !selectedId) return;
    let active = true;
    getArchitectureSource(workspaceId, selectedId, inventory.resolved_sha)
      .then((value) => {
        if (active) setSource(value);
      })
      .catch((caught) => {
        if (active) setError(errorMessage(caught) || "source_failed");
      });
    getArchitectureRender(workspaceId, selectedId, inventory.resolved_sha)
      .then((value) => {
        if (active) setRender(value);
      })
      .catch(() => {
        if (active) setRenderError(true);
      });
    return () => {
      active = false;
    };
  }, [inventory, selectedId, workspaceId]);

  const refresh = useCallback(async () => {
    if (!workspaceId) return;
    setRefreshing(true);
    try {
      acceptInventory(await refreshArchitecture(workspaceId));
    } catch (caught) {
      setError(errorMessage(caught) || "request_failed");
    } finally {
      setRefreshing(false);
    }
  }, [acceptInventory, workspaceId]);

  return {
    inventory,
    selectedId,
    setSelectedId,
    view,
    setView,
    source,
    render,
    renderError,
    loading,
    refreshing,
    error,
    retry: load,
    refresh,
  };
}
