import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@kandev/ui/badge";
import { Button } from "@kandev/ui/button";
import { MobilePickerSheet } from "@/components/task/mobile/mobile-picker-sheet";
import { MobilePillButton } from "@/components/task/mobile/mobile-pill-button";
import {
  type ArchitectureDiagram,
  type ArchitectureInventory,
} from "@/lib/api/domains/architecture-api";
import type { ArchitectureView } from "./use-architecture-browser";

type SurfaceProps = {
  inventory: ArchitectureInventory;
  selected: ArchitectureDiagram;
  selectedId: string;
  onSelect: (id: string) => void;
  view: ArchitectureView;
  onView: (view: ArchitectureView) => void;
  source: string;
  render: string;
  renderError: boolean;
  workspaceId: string;
};

export function DesktopArchitecture({
  inventory,
  selected,
  selectedId,
  onSelect,
  view,
  onView,
  source,
  render,
  renderError,
}: SurfaceProps) {
  const { t } = useTranslation("architecture");
  return (
    <div
      className="grid min-h-0 flex-1 grid-cols-[15rem_minmax(0,1fr)_22rem] overflow-hidden"
      data-testid="architecture-desktop"
    >
      <nav className="min-h-0 overflow-y-auto border-r p-2" aria-label={t("diagrams")}>
        {inventory.diagrams.map((diagram) => (
          <DiagramButton
            key={diagram.id}
            diagram={diagram}
            active={diagram.id === selectedId}
            onClick={() => onSelect(diagram.id)}
          />
        ))}
      </nav>
      <RenderFrame selected={selected} html={render} error={renderError} />
      <aside className="flex min-h-0 flex-col border-l">
        <ViewTabs
          value={view === "render" ? "source" : view}
          onValueChange={onView}
          includeRender={false}
        />
        <DetailBody
          view={view === "render" ? "source" : view}
          selected={selected}
          source={source}
          inventory={inventory}
        />
      </aside>
    </div>
  );
}

export function MobileArchitecture(props: SurfaceProps) {
  const { t } = useTranslation("architecture");
  const [pickerOpen, setPickerOpen] = useState(false);
  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-hidden pb-[env(safe-area-inset-bottom)]"
      data-testid="architecture-mobile"
    >
      <div className="shrink-0 border-b p-2">
        <MobilePillButton
          label={props.selected.title}
          fullWidth
          isOpen={pickerOpen}
          onClick={() => setPickerOpen(true)}
          data-testid="architecture-diagram-trigger"
        />
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        {props.view === "render" ? (
          <RenderFrame selected={props.selected} html={props.render} error={props.renderError} />
        ) : (
          <DetailBody
            view={props.view}
            selected={props.selected}
            source={props.source}
            inventory={props.inventory}
          />
        )}
      </div>
      <ViewTabs value={props.view} onValueChange={props.onView} includeRender />
      <MobilePickerSheet
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        title={t("chooseDiagram")}
        contentTestId="architecture-diagram-list"
      >
        <div className="space-y-1">
          {props.inventory.diagrams.map((diagram) => (
            <DiagramButton
              key={diagram.id}
              diagram={diagram}
              active={diagram.id === props.selectedId}
              onClick={() => {
                props.onSelect(diagram.id);
                setPickerOpen(false);
              }}
              touch
            />
          ))}
        </div>
      </MobilePickerSheet>
    </div>
  );
}

function DiagramButton({
  diagram,
  active,
  onClick,
  touch = false,
}: {
  diagram: ArchitectureDiagram;
  active: boolean;
  onClick: () => void;
  touch?: boolean;
}) {
  return (
    <Button
      type="button"
      variant={active ? "secondary" : "ghost"}
      className={`w-full cursor-pointer justify-start ${touch ? "min-h-11" : ""}`}
      onClick={onClick}
    >
      {diagram.title}
    </Button>
  );
}

function RenderFrame({
  selected,
  html,
  error,
}: {
  selected: ArchitectureDiagram;
  html: string;
  error: boolean;
}) {
  const { t } = useTranslation("architecture");
  if (error)
    return (
      <div
        className="flex h-full items-center justify-center p-6 text-sm text-muted-foreground"
        role="alert"
      >
        {t("renderUnavailable")}
      </div>
    );
  if (!html)
    return (
      <div
        className="flex h-full items-center justify-center p-6 text-sm text-muted-foreground"
        role="status"
      >
        {t("loadingRender")}
      </div>
    );
  return (
    <iframe
      data-testid="architecture-render"
      title={t("renderTitle", { title: selected.title })}
      srcDoc={html}
      className="h-full min-h-0 w-full border-0"
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-modals"
      referrerPolicy="no-referrer"
    />
  );
}

function ViewTabs({
  value,
  onValueChange,
  includeRender,
}: {
  value: ArchitectureView;
  onValueChange: (value: ArchitectureView) => void;
  includeRender: boolean;
}) {
  const { t } = useTranslation("architecture");
  const views: ArchitectureView[] = includeRender
    ? ["render", "source", "validation"]
    : ["source", "validation"];
  return (
    <div
      role="tablist"
      className={`grid shrink-0 border-b bg-muted p-1 ${includeRender ? "grid-cols-3" : "grid-cols-2"}`}
    >
      {views.map((next) => (
        <Button
          key={next}
          type="button"
          role="tab"
          aria-selected={value === next}
          variant={value === next ? "secondary" : "ghost"}
          className={`cursor-pointer ${includeRender ? "min-h-11" : ""}`}
          onClick={() => onValueChange(next)}
        >
          {t(next)}
        </Button>
      ))}
    </div>
  );
}

function DetailBody({
  view,
  selected,
  source,
  inventory,
}: {
  view: ArchitectureView;
  selected: ArchitectureDiagram;
  source: string;
  inventory: ArchitectureInventory;
}) {
  const { t } = useTranslation("architecture");
  if (view === "source")
    return (
      <pre
        className="h-full overflow-auto whitespace-pre-wrap break-words p-4 text-xs"
        data-testid="architecture-source"
      >
        {source || t("loadingSource")}
      </pre>
    );
  const diagnostics = selected.validation.diagnostics;
  return (
    <div className="h-full overflow-y-auto p-4 text-sm" data-testid="architecture-validation">
      <div className="mb-3 flex items-center gap-2">
        <Badge variant="outline">{selected.validation.status}</Badge>
        <span>{selected.type}</span>
      </div>
      <pre className="whitespace-pre-wrap break-words text-xs">
        {diagnostics ? JSON.stringify(diagnostics, null, 2) : t("noDiagnostics")}
      </pre>
      <dl className="mt-4 space-y-2 text-xs text-muted-foreground">
        <div>
          <dt>{t("sourcePath")}</dt>
          <dd>{selected.source_path}</dd>
        </div>
        <div>
          <dt>{t("gitRef")}</dt>
          <dd>{inventory.git_ref}</dd>
        </div>
        <div>
          <dt>{t("resolvedSha")}</dt>
          <dd>{inventory.resolved_sha}</dd>
        </div>
      </dl>
    </div>
  );
}
