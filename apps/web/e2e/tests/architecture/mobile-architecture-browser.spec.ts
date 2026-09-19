import { expect } from "@playwright/test";
import {
  ARCHITECTURE_DIAGRAMS,
  cleanupArchitectureFixture,
  prepareArchitectureFixture,
  repositoryEvidence,
} from "../../helpers/architecture-fixture";
import { test } from "../../fixtures/test-base";
import { ArchitecturePage } from "../../pages/architecture-page";

test("mobile architecture browsing preserves focused views and containment", async ({
  apiClient,
  backend,
  seedData,
  testPage,
}, testInfo) => {
  const fixture = await prepareArchitectureFixture(apiClient, seedData, backend);
  try {
    const architecture = new ArchitecturePage(testPage);
    await architecture.goto();
    await expect(testPage.getByTestId("architecture-mobile")).toBeVisible();

    for (const id of ARCHITECTURE_DIAGRAMS) {
      await architecture.selectMobileDiagram(id.replaceAll("_", " "));
    }
    await architecture.show("Source");
    await expect(testPage.getByTestId("architecture-source")).toContainText("END_TO_END_FLOWS");
    await architecture.show("Validation");
    await expect(testPage.getByTestId("architecture-validation")).toContainText(fixture.sha);
    await architecture.show("Render");
    await expect(
      testPage
        .getByTestId("architecture-render")
        .contentFrame()
        .getByText("Deterministic Archify render"),
    ).toBeVisible();

    const controls = testPage.getByRole("tab");
    for (let index = 0; index < (await controls.count()); index += 1) {
      expect((await controls.nth(index).boundingBox())?.height).toBeGreaterThanOrEqual(44);
    }
    const noHorizontalOverflow = await testPage.evaluate(
      () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
    );
    expect(noHorizontalOverflow).toBe(true);
    await testPage.getByRole("button", { name: "Refresh" }).click();
    expect(repositoryEvidence(seedData, backend)).toEqual({
      status: fixture.statusBefore,
      tree: fixture.treeBefore,
    });
    await testPage.screenshot({
      path: testInfo.outputPath("architecture-mobile.png"),
      fullPage: true,
    });
  } finally {
    await cleanupArchitectureFixture(apiClient, seedData, backend);
  }
});
