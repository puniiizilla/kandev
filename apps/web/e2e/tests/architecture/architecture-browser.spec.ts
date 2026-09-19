import { expect } from "@playwright/test";
import {
  ARCHITECTURE_DIAGRAMS,
  cleanupArchitectureFixture,
  correctArchitectureFixture,
  prepareArchitectureFixture,
  repositoryEvidence,
} from "../../helpers/architecture-fixture";
import { test } from "../../fixtures/test-base";
import { ArchitecturePage } from "../../pages/architecture-page";

test.describe("repository-bound architecture browser", () => {
  test("browses seven diagrams, refreshes derived output, and leaves Git unchanged", async ({
    apiClient,
    backend,
    seedData,
    testPage,
  }, testInfo) => {
    const fixture = await prepareArchitectureFixture(apiClient, seedData, backend);
    try {
      const architecture = new ArchitecturePage(testPage);
      await architecture.goto();
      await expect(testPage.getByTestId("architecture-desktop")).toBeVisible();
      await expect(testPage.getByText(fixture.sha.slice(0, 8), { exact: false })).toBeVisible();

      for (const id of ARCHITECTURE_DIAGRAMS) {
        const title = id.replaceAll("_", " ");
        await architecture.selectDesktopDiagram(title);
        await expect(
          testPage
            .getByTestId("architecture-render")
            .contentFrame()
            .getByRole("heading", { name: title }),
        ).toBeVisible();
      }

      await expect(testPage.getByTestId("architecture-source")).toContainText("END_TO_END_FLOWS");
      await architecture.show("Validation");
      await expect(testPage.getByTestId("architecture-validation")).toContainText("valid");
      await expect(testPage.getByTestId("architecture-validation")).toContainText(
        "docs/archify/ist",
      );
      await expect(testPage.getByTestId("architecture-validation")).toContainText(fixture.sha);

      await testPage.getByRole("button", { name: "Refresh" }).click();
      await expect(
        testPage
          .getByTestId("architecture-render")
          .contentFrame()
          .getByText("Deterministic Archify render"),
      ).toBeVisible();
      expect(repositoryEvidence(seedData, backend)).toEqual({
        status: fixture.statusBefore,
        tree: fixture.treeBefore,
      });

      await testPage.screenshot({
        path: testInfo.outputPath("architecture-desktop.png"),
        fullPage: true,
      });
    } finally {
      await cleanupArchitectureFixture(apiClient, seedData, backend);
    }
  });

  test("fails closed, exposes invalid source diagnostics, and recovers after correction", async ({
    apiClient,
    backend,
    seedData,
    testPage,
  }) => {
    const invalid = "DATA_FLOW";
    const fixture = await prepareArchitectureFixture(apiClient, seedData, backend, invalid);
    try {
      const architecture = new ArchitecturePage(testPage);
      await architecture.goto();
      await architecture.selectDesktopDiagram("DATA FLOW");
      await expect(testPage.getByRole("alert")).toContainText("rendered diagram is unavailable");
      await expect(testPage.getByTestId("architecture-source")).toContainText('"invalid": true');
      await architecture.show("Validation");
      await expect(testPage.getByTestId("architecture-validation")).toContainText(
        "fixture_invalid",
      );
      await expect(testPage.getByTestId("architecture-validation")).toContainText(
        "Set invalid to false",
      );

      correctArchitectureFixture(seedData, backend, invalid);
      await testPage.getByRole("button", { name: "Refresh" }).click();
      await architecture.selectDesktopDiagram("DATA FLOW");
      await expect(testPage.getByTestId("architecture-render")).toBeVisible();

      await apiClient.updateRepository(seedData.repositoryId, {
        architecture_git_ref: "",
        architecture_path: "",
        archify_runtime: "",
      });
      await testPage.reload();
      await expect(testPage.getByRole("status")).toContainText(
        "ARCHITECTURE_BINDING_INVALID: Archify binding is incomplete",
      );
      await expect(testPage.getByRole("button", { name: "Retry" })).toBeVisible();
      expect(repositoryEvidence(seedData, backend).status).toBe(fixture.statusBefore);
    } finally {
      await cleanupArchitectureFixture(apiClient, seedData, backend);
    }
  });
});
