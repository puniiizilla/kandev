import { publishArchitectureEvidence } from "@/lib/architecture-evidence-events";
import type { WsHandlers } from "@/lib/ws/handlers/types";

export function registerArchitectureEvidenceHandlers(): WsHandlers {
  return {
    "task.architecture_evidence.updated": (message) => {
      const { task_id, evidence } = message.payload;
      if (task_id && evidence?.task_id === task_id) publishArchitectureEvidence(evidence);
    },
  };
}
