import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ArchitecturePageClient } from "./architecture-page-client";

const api = vi.hoisted(() => ({
  inventory: vi.fn(),
  refresh: vi.fn(),
  source: vi.fn(),
  render: vi.fn(),
}));
let mobile = false;

vi.mock("@/lib/api/domains/architecture-api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api/domains/architecture-api")>(
    "@/lib/api/domains/architecture-api",
  );
  return {
    ...actual,
    getArchitectureInventory: api.inventory,
    refreshArchitecture: api.refresh,
    getArchitectureSource: api.source,
    getArchitectureRender: api.render,
  };
});
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: mobile, usesDesktopWorkbench: !mobile }),
}));
vi.mock("@/components/page-shell", () => ({
  PageShell: ({ children, actions }: { children: React.ReactNode; actions: React.ReactNode }) => (
    <div>
      {actions}
      {children}
    </div>
  ),
}));
vi.mock("@/components/task/mobile/mobile-picker-sheet", () => ({
  MobilePickerSheet: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div data-testid="diagram-picker">{children}</div> : null,
}));

const inventory = {
  repository_id: "repo",
  repository: "TheBrain",
  git_ref: "main",
  resolved_sha: "abcdef123456",
  source_path: "docs/archify/ist",
  source_dirty: false,
  diagrams: [
    {
      id: "SYSTEM_OVERVIEW",
      title: "System Overview",
      type: "architecture",
      source_path: "docs/archify/ist/SYSTEM_OVERVIEW.architecture.json",
      validation: { status: "valid", diagnostics: [{ code: "OK", message: "Validated" }] },
    },
    {
      id: "DATA_FLOW",
      title: "Data Flow",
      type: "architecture",
      source_path: "docs/archify/ist/DATA_FLOW.architecture.json",
      validation: { status: "valid" },
    },
  ],
};

beforeEach(() => {
  mobile = false;
  api.inventory.mockReset().mockResolvedValue(inventory);
  api.refresh.mockReset().mockResolvedValue(inventory);
  api.source.mockReset().mockResolvedValue('{"diagram":"source"}');
  api.render.mockReset().mockResolvedValue("<html>render</html>");
});
afterEach(cleanup);

describe("ArchitecturePageClient", () => {
  it("navigates diagrams and exposes render, source, validation, and Git metadata on desktop", async () => {
    render(<ArchitecturePageClient workspaceId="workspace" />);
    expect(await screen.findByText("System Overview")).toBeTruthy();
    fireEvent.click(screen.getByText("Data Flow"));
    await waitFor(() =>
      expect(api.render).toHaveBeenCalledWith("workspace", "DATA_FLOW", "abcdef123456"),
    );
    expect(screen.getByTestId("architecture-render").getAttribute("srcdoc")).toContain("render");
    fireEvent.click(screen.getByRole("tab", { name: "Source" }));
    expect(await screen.findByText('{"diagram":"source"}')).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: "Validation" }));
    expect(await screen.findByText("valid")).toBeTruthy();
    expect(screen.getByText(/abcdef1/)).toBeTruthy();
  });

  it("uses a focused mobile mode and refreshes without mixing revisions", async () => {
    mobile = true;
    render(<ArchitecturePageClient workspaceId="workspace" />);
    fireEvent.click(await screen.findByTestId("architecture-diagram-trigger"));
    expect(screen.getByTestId("diagram-picker")).toBeTruthy();
    fireEvent.click(
      within(screen.getByTestId("diagram-picker")).getByRole("button", { name: "Data Flow" }),
    );
    fireEvent.click(screen.getByRole("tab", { name: "Source" }));
    expect(await screen.findByText('{"diagram":"source"}')).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    await waitFor(() => expect(api.refresh).toHaveBeenCalledWith("workspace"));
  });

  it("shows a fail-closed error and retry action", async () => {
    api.inventory.mockRejectedValue({
      body: {
        error: {
          code: "ARCHITECTURE_BINDING_MISSING",
          message: "Archify binding is not configured",
        },
      },
    });
    render(<ArchitecturePageClient workspaceId="workspace" />);
    expect(await screen.findByText("Archify binding is not configured")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(api.inventory).toHaveBeenCalledTimes(2);
  });
});
