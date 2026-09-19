"use client";

import { Button } from "@kandev/ui/button";
import { useTranslation } from "react-i18next";
import { useArchitectureEvidence } from "@/hooks/domains/task/use-architecture-evidence";
import { architectureEvidenceDeltaUrl } from "@/lib/api/domains/architecture-evidence-api";
import type { ArchitectureEvidence } from "@/lib/api/domains/architecture-evidence-api";

function shortSha(value?: string) {
  return value?.slice(0, 8) || "-";
}
function list(value?: string): string[] {
  try {
    const parsed = JSON.parse(value || "[]");
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function summary(value?: string): string {
  try {
    const parsed = JSON.parse(value || "{}");
    return Object.entries(parsed)
      .map(([key, entry]) => `${key}: ${Array.isArray(entry) ? entry.join(", ") : String(entry)}`)
      .join("; ");
  } catch {
    return "";
  }
}

function diagnostics(value?: string): string[] {
  try {
    const parsed = JSON.parse(value || "[]");
    if (!Array.isArray(parsed)) return [];
    return parsed.map((entry) =>
      [entry?.code, entry?.message, entry?.correction].filter(Boolean).join(": "),
    );
  } catch {
    return [];
  }
}

function EvidenceDetails({ evidence, taskId }: { evidence: ArchitectureEvidence; taskId: string }) {
  const { t } = useTranslation();
  const diagrams = list(evidence.affected_diagrams_json);
  const diffSummary = summary(evidence.diff_summary_json);
  const diagnosticLines = diagnostics(evidence.diagnostics_json);
  return (
    <div className="mt-2 grid min-w-0 grid-cols-2 gap-2 sm:grid-cols-4">
      <span>
        {t("task:architectureEvidenceRequired")}: {evidence.required ? t("task:yes") : t("task:no")}
      </span>
      <span>
        {t("task:validation")}: {evidence.validation_status || evidence.state}
      </span>
      <span>
        {t("task:architectureChanged")}:{" "}
        {evidence.architecture_changed ? t("task:yes") : t("task:no")}
      </span>
      <span>
        {t("task:semanticChange")}: {evidence.semantic_change ? t("task:yes") : t("task:no")}
      </span>
      <span className="break-all">
        {t("task:baseSha")}: {shortSha(evidence.base_sha)}
      </span>
      <span className="break-all">
        {t("task:headSha")}: {shortSha(evidence.head_sha)}
      </span>
      {diagrams.length > 0 && (
        <div className="col-span-full min-w-0">
          <span>{t("task:affectedDiagrams")}: </span>
          {diagrams.join(", ")}
        </div>
      )}
      {diffSummary && (
        <div className="col-span-full min-w-0 break-words">
          <span>{t("task:architectureDiffSummary")}: </span>
          {diffSummary}
        </div>
      )}
      {diagnosticLines.length > 0 && (
        <div className="col-span-full min-w-0 break-words text-destructive">
          <span>{t("task:architectureDiagnostics")}: </span>
          {diagnosticLines.join("; ")}
        </div>
      )}
      <div className="col-span-full flex flex-col gap-2 sm:flex-row">
        <a
          className="inline-flex min-h-11 items-center text-primary underline sm:min-h-7"
          href="/architecture"
        >
          {t("task:openArchitecture")}
        </a>
        {diagrams[0] && (
          <a
            className="inline-flex min-h-11 items-center text-primary underline sm:min-h-7"
            href={architectureEvidenceDeltaUrl(taskId, diagrams[0])}
          >
            {t("task:openArchitectureDelta")}
          </a>
        )}
      </div>
    </div>
  );
}

export function ArchitectureEvidenceCard({
  taskId,
  mobile = false,
}: {
  taskId?: string | null;
  mobile?: boolean;
}) {
  const { t } = useTranslation();
  const { evidence, error, loading, refresh } = useArchitectureEvidence(taskId);
  if (!taskId) return null;
  const failed = error || evidence?.state?.includes("FAILED") || evidence?.state === "STALE";
  return (
    <section
      className="mx-3 mt-2 overflow-hidden rounded-md border bg-card p-3 text-xs"
      data-testid="architecture-evidence-card"
      aria-labelledby="architecture-evidence-title"
    >
      <div className="flex items-center justify-between gap-2">
        <h3 id="architecture-evidence-title" className="font-medium">
          {t("task:architectureEvidence")}
        </h3>
        <Button
          variant="outline"
          size="sm"
          className={mobile ? "min-h-11 cursor-pointer" : "cursor-pointer"}
          disabled={loading}
          onClick={() => void refresh()}
        >
          {failed ? t("task:retry") : t("task:refresh")}
        </Button>
      </div>
      {loading && !evidence ? (
        <p role="status" className="mt-2 text-muted-foreground">
          {t("task:architectureEvidenceLoading")}
        </p>
      ) : null}
      {error ? (
        <p role="alert" className="mt-2 text-destructive">
          {t("task:architectureEvidenceUnavailable")}
        </p>
      ) : null}
      {evidence ? <EvidenceDetails evidence={evidence} taskId={taskId} /> : null}
    </section>
  );
}
