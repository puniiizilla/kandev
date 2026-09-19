import { beforeEach, describe, expect, it, vi } from "vitest";
import { getArchitectureEvidence, refreshArchitectureEvidence } from "./architecture-evidence-api";

describe("architecture evidence API", () => {
  beforeEach(() =>
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockImplementation(
          async () => new Response(JSON.stringify({ state: "READY_FOR_REVIEW" }), { status: 200 }),
        ),
    ),
  );
  it("uses task-scoped stable endpoints", async () => {
    await getArchitectureEvidence("task/one");
    await refreshArchitectureEvidence("task/one");
    expect(vi.mocked(fetch).mock.calls.map(([url, init]) => [String(url), init?.method])).toEqual([
      [expect.stringContaining("/api/v1/tasks/task%2Fone/architecture-evidence"), "GET"],
      [expect.stringContaining("/api/v1/tasks/task%2Fone/architecture-evidence/refresh"), "POST"],
    ]);
  });
});
