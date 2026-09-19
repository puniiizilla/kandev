import { expect } from "@playwright/test";
import { test } from "../../fixtures/test-base";
import { waitForLatestSessionDone } from "../../helpers/session";

test("mobile architecture evidence actions remain touch sized without overflow", async ({
  apiClient,
  seedData,
  testPage,
}) => {
  const task = await apiClient.createTaskWithAgent(
    seedData.workspaceId,
    "Mobile architecture evidence",
    seedData.agentProfileId,
    {
      description: "Confirm mobile evidence.",
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      repository_ids: [seedData.repositoryId],
    },
  );
  await waitForLatestSessionDone(
    apiClient,
    task.id,
    1,
    "mobile evidence fixture session should finish",
  );
  await testPage.route("**/api/v1/tasks/*/architecture-evidence", async (route) =>
    route.fulfill({
      json: {
        state: "READY_FOR_REVIEW",
        required: true,
        validation_status: "PASS",
        architecture_changed: true,
        semantic_change: true,
        base_sha: "123456789",
        head_sha: "abcdef012",
        affected_diagrams_json: '["SYSTEM_OVERVIEW"]',
      },
    }),
  );
  await testPage.goto(`/t/${task.id}`);
  await testPage.getByRole("button", { name: "Changes" }).tap();
  const card = testPage.getByTestId("architecture-evidence-card");
  await expect(card).toContainText("SYSTEM_OVERVIEW");
  const box = await card.getByRole("button").boundingBox();
  expect(box?.height).toBeGreaterThanOrEqual(44);
  expect(
    await testPage.evaluate(
      () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
    ),
  ).toBeTruthy();
});
