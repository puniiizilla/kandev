import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getArchitectureInventory,
  getArchitectureRenderUrl,
  getArchitectureSource,
  refreshArchitecture,
} from "./architecture-api";

const inventory = {
  repository_id: "repo",
  repository: "TheBrain",
  git_ref: "main",
  resolved_sha: "abc",
  source_path: "docs/archify/ist",
  source_dirty: false,
  diagrams: [],
};

afterEach(() => vi.unstubAllGlobals());

describe("architecture API", () => {
  it("scopes inventory and refresh to the explicit workspace", async () => {
    const fetchMock = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(new Response(JSON.stringify(inventory), { status: 200 })),
      );
    vi.stubGlobal("fetch", fetchMock);
    await getArchitectureInventory("workspace one");
    await refreshArchitecture("workspace one");
    expect(fetchMock.mock.calls.map(([url, init]) => [String(url), init.method ?? "GET"])).toEqual([
      [expect.stringContaining("/api/v1/architecture?workspace_id=workspace+one"), "GET"],
      [expect.stringContaining("/api/v1/architecture/refresh?workspace_id=workspace+one"), "POST"],
    ]);
  });

  it("pins source and render requests to the inventory revision", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response('{"ok":true}', { status: 200 })));
    await expect(getArchitectureSource("workspace", "SYSTEM_OVERVIEW", "sha-1")).resolves.toBe(
      '{"ok":true}',
    );
    expect(getArchitectureRenderUrl("workspace", "SYSTEM_OVERVIEW", "sha-1")).toContain(
      "revision=sha-1",
    );
  });
});
