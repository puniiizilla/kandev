"use client";

import { useTranslation } from "react-i18next";
import { IconRefresh } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { PageShell } from "@/components/page-shell";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import type { ArchitectureDiagram } from "@/lib/api/domains/architecture-api";
import { DesktopArchitecture, MobileArchitecture } from "./architecture-surfaces";
import { useArchitectureBrowser } from "./use-architecture-browser";

export function ArchitecturePageClient({ workspaceId }: { workspaceId?: string }) {
  const { t } = useTranslation("architecture");
  const responsive = useResponsiveBreakpoint();
  const state = useArchitectureBrowser(workspaceId);
  const selected = state.inventory?.diagrams.find((diagram) => diagram.id === state.selectedId);
  const action = (
    <Button
      type="button"
      variant="outline"
      className="cursor-pointer max-md:min-h-11 max-md:min-w-11"
      disabled={state.refreshing || !workspaceId}
      onClick={() => void state.refresh()}
      aria-label={t("refresh")}
    >
      <IconRefresh className={state.refreshing ? "animate-spin" : ""} />
      <span className="max-sm:hidden">{t("refresh")}</span>
    </Button>
  );

  const subtitle = state.inventory
    ? t("metadata", {
        repository: state.inventory.repository,
        sha: state.inventory.resolved_sha.slice(0, 8),
        state: state.inventory.source_dirty ? t("dirty") : t("clean"),
      })
    : undefined;
  return (
    <PageShell
      title={t("title")}
      subtitle={subtitle}
      actions={action}
      scroll="none"
      contentTestId="architecture-page"
    >
      <ArchitectureContent
        state={state}
        selected={selected}
        workspaceId={workspaceId}
        desktop={responsive.usesDesktopWorkbench}
      />
    </PageShell>
  );
}

function ArchitectureContent({
  state,
  selected,
  workspaceId,
  desktop,
}: {
  state: ReturnType<typeof useArchitectureBrowser>;
  selected?: ArchitectureDiagram;
  workspaceId?: string;
  desktop: boolean;
}) {
  const { t } = useTranslation("architecture");
  if (state.loading) return <StateMessage text={t("loading")} />;
  if (state.error) {
    const knownError =
      state.error === "workspace_required" ? t("workspaceRequired") : t("unavailable");
    const text =
      state.error === "workspace_required" || state.error === "request_failed"
        ? knownError
        : state.error;
    return (
      <StateMessage
        text={text}
        action={
          <Button
            type="button"
            variant="outline"
            className="cursor-pointer max-md:min-h-11"
            onClick={() => void state.retry()}
          >
            {t("retry")}
          </Button>
        }
      />
    );
  }
  if (!state.inventory?.diagrams.length || !selected) return <StateMessage text={t("empty")} />;
  const props = {
    inventory: state.inventory,
    selected,
    selectedId: state.selectedId,
    onSelect: state.setSelectedId,
    view: state.view,
    onView: state.setView,
    source: state.source,
    render: state.render,
    renderError: state.renderError,
    workspaceId: workspaceId ?? "",
  };
  return (
    <div
      className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      data-testid="architecture-page"
    >
      {desktop ? <DesktopArchitecture {...props} /> : <MobileArchitecture {...props} />}
    </div>
  );
}

function StateMessage({ text, action }: { text: string; action?: React.ReactNode }) {
  return (
    <div
      className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 p-6 text-center"
      role="status"
    >
      <p className="text-sm text-muted-foreground">{text}</p>
      {action}
    </div>
  );
}
