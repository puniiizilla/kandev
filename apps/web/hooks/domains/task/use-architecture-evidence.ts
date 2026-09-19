import { useCallback, useEffect, useState } from "react";
import {
  getArchitectureEvidence,
  refreshArchitectureEvidence,
  type ArchitectureEvidence,
} from "@/lib/api/domains/architecture-evidence-api";
import { subscribeArchitectureEvidence } from "@/lib/architecture-evidence-events";

export function useArchitectureEvidence(taskId: string | null | undefined) {
  const [evidence, setEvidence] = useState<ArchitectureEvidence | null>(null);
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(false);
  const load = useCallback(
    async (refresh = false) => {
      if (!taskId) return;
      setLoading(true);
      setError(false);
      try {
        setEvidence(
          await (refresh ? refreshArchitectureEvidence(taskId) : getArchitectureEvidence(taskId)),
        );
      } catch {
        setError(true);
      } finally {
        setLoading(false);
      }
    },
    [taskId],
  );
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(
    () =>
      subscribeArchitectureEvidence((updated) => {
        if (updated.task_id === taskId) setEvidence(updated);
      }),
    [taskId],
  );
  return { evidence, error, loading, refresh: () => load(true) };
}
