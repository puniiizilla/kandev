import type { ArchitectureEvidence } from "@/lib/api/domains/architecture-evidence-api";

type Listener = (evidence: ArchitectureEvidence) => void;
const listeners = new Set<Listener>();

export function publishArchitectureEvidence(evidence: ArchitectureEvidence): void {
  listeners.forEach((listener) => listener(evidence));
}

export function subscribeArchitectureEvidence(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
