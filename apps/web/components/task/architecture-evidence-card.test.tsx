import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ArchitectureEvidenceCard } from "./architecture-evidence-card";

const refresh = vi.fn();
vi.mock("@/hooks/domains/task/use-architecture-evidence", () => ({
  useArchitectureEvidence: () => ({
    evidence: {
      state: "READY_FOR_REVIEW",
      required: true,
      validation_status: "PASS",
      architecture_changed: true,
      semantic_change: true,
      base_sha: "123456789",
      head_sha: "abcdef012",
      affected_diagrams_json: '["SYSTEM_OVERVIEW"]',
      diff_summary_json: '{"affected_diagrams":["SYSTEM_OVERVIEW"]}',
      diagnostics_json: '[{"code":"CHECK","message":"Review evidence"}]',
    },
    error: false,
    loading: false,
    refresh,
  }),
}));

afterEach(() => {
  cleanup();
  refresh.mockClear();
});

describe("ArchitectureEvidenceCard", () => {
  it("shows SHA-bound review evidence and touch-sized mobile actions", () => {
    render(<ArchitectureEvidenceCard taskId="task-1" mobile />);
    expect(screen.getByText(/12345678/)).toBeTruthy();
    expect(screen.getByText(/abcdef01/)).toBeTruthy();
    expect(screen.getAllByText(/SYSTEM_OVERVIEW/)).toHaveLength(2);
    expect(screen.getByText(/affected_diagrams/)).toBeTruthy();
    expect(screen.getByText(/CHECK: Review evidence/)).toBeTruthy();
    expect(
      screen.getByRole("link", { name: /architecture delta/i }).getAttribute("href"),
    ).toContain("task-1");
    const button = screen.getByRole("button");
    expect(button.className).toContain("min-h-11");
    fireEvent.click(button);
    expect(refresh).toHaveBeenCalledOnce();
  });
});
