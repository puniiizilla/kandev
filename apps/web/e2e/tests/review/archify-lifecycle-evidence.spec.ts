import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { expect } from "@playwright/test";
import { test } from "../../fixtures/test-base";
import {
  cleanupArchitectureFixture,
  prepareArchitectureFixture,
} from "../../helpers/architecture-fixture";
import { makeGitEnv } from "../../helpers/git-helper";
import { waitForLatestSessionDone } from "../../helpers/session";
import { SessionPage } from "../../pages/session-page";

test("captures SHA-pinned evidence and presents it beside changes", async ({
  apiClient,
  backend,
  seedData,
  testPage,
}, testInfo) => {
  await prepareArchitectureFixture(apiClient, seedData, backend);
  try {
    const task = await apiClient.createTaskWithAgent(
      seedData.workspaceId,
      "Archify lifecycle E2E",
      seedData.agentProfileId,
      {
        description: "Confirm the architecture evidence fixture.",
        workflow_id: seedData.workflowId,
        workflow_step_id: seedData.startStepId,
        executor_profile_id: seedData.worktreeExecutorProfileId,
        repository_ids: [seedData.repositoryId],
      },
    );
    await waitForLatestSessionDone(
      apiClient,
      task.id,
      1,
      "architecture fixture session should finish",
    );
    const sessions = await apiClient.listTaskSessions(task.id);
    const session =
      sessions.sessions.find((candidate) => candidate.is_primary) ?? sessions.sessions[0];
    const worktree =
      session.worktrees?.find((candidate) => candidate.repository_id === seedData.repositoryId)
        ?.worktree_path ?? session.worktree_path;
    if (!worktree) throw new Error("prepared task has no repository worktree");
    const sourcePath = path.join(worktree, "docs/archify/ist/SYSTEM_OVERVIEW.architecture.json");
    const source = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
    source.meta.title = "Lifecycle after";
    fs.writeFileSync(sourcePath, `${JSON.stringify(source, null, 2)}\n`);
    const git = (...args: string[]) =>
      execFileSync("git", ["-C", worktree, ...args], {
        encoding: "utf8",
        env: makeGitEnv(backend.tmpDir),
      }).trim();
    git("add", "docs/archify/ist");
    git("commit", "-m", "test: architecture after");
    const response = await apiClient.rawRequest("PATCH", `/api/v1/tasks/${task.id}`, {
      labels: ["architecture"],
    });
    expect(response.ok).toBeTruthy();
    const evidence = await apiClient.refreshArchitectureEvidence(task.id);
    expect(evidence.state).toBe("READY_FOR_REVIEW");
    expect(evidence.base_sha).not.toBe(evidence.head_sha);
    await testPage.goto(`/t/${task.id}`);
    const page = new SessionPage(testPage);
    await page.waitForLoad(45_000);
    await page.clickTab("Changes");
    await expect(testPage.getByTestId("architecture-evidence-card")).toContainText(
      "SYSTEM_OVERVIEW",
    );
    await testPage.screenshot({
      path: testInfo.outputPath("archify-lifecycle-evidence-desktop.png"),
      fullPage: true,
    });

    fs.appendFileSync(sourcePath, "\n");
    const dirtyRefresh = await apiClient.rawRequest(
      "POST",
      `/api/v1/tasks/${task.id}/architecture-evidence/refresh`,
    );
    expect(dirtyRefresh.status).toBe(409);
    const dirtyEvidence = (await dirtyRefresh.json()) as { state?: string };
    expect(dirtyEvidence.state).toBe("ARCHIFY_AFTER_FAILED");
  } finally {
    await cleanupArchitectureFixture(apiClient, seedData, backend);
  }
});
