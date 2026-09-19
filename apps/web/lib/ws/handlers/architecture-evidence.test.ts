import { describe, expect, it, vi } from "vitest";
import { subscribeArchitectureEvidence } from "@/lib/architecture-evidence-events";
import type { ArchitectureEvidence } from "@/lib/api/domains/architecture-evidence-api";
import { registerArchitectureEvidenceHandlers } from "./architecture-evidence";

describe("architecture evidence websocket updates", () => {
  it("publishes a matching task-scoped projection", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeArchitectureEvidence(listener);
    const evidence: ArchitectureEvidence = {
      task_id: "task-1",
      state: "READY_FOR_REVIEW",
      required: true,
      architecture_changed: true,
      semantic_change: true,
    };
    registerArchitectureEvidenceHandlers()["task.architecture_evidence.updated"]!({
      type: "notification",
      action: "task.architecture_evidence.updated",
      payload: { task_id: "task-1", evidence },
    });
    expect(listener).toHaveBeenCalledWith(evidence);
    unsubscribe();
  });

  it("rejects mismatched task projections", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeArchitectureEvidence(listener);
    registerArchitectureEvidenceHandlers()["task.architecture_evidence.updated"]!({
      type: "notification",
      action: "task.architecture_evidence.updated",
      payload: {
        task_id: "task-1",
        evidence: {
          task_id: "task-2",
          state: "READY_FOR_REVIEW",
          required: true,
          architecture_changed: true,
          semantic_change: true,
        },
      },
    });
    expect(listener).not.toHaveBeenCalled();
    unsubscribe();
  });
});
