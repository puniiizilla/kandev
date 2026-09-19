import { expect, type Page } from "@playwright/test";

export class ArchitecturePage {
  constructor(private readonly page: Page) {}

  async goto() {
    await this.page.goto("/architecture");
    await expect(this.page.getByTestId("architecture-page").last()).toBeVisible();
  }

  async selectDesktopDiagram(title: string) {
    await this.page
      .getByRole("navigation", { name: "Diagrams" })
      .getByRole("button", { name: title })
      .click();
  }

  async selectMobileDiagram(title: string) {
    await this.page.getByTestId("architecture-diagram-trigger").click();
    await this.page
      .getByTestId("architecture-diagram-list")
      .getByRole("button", { name: title })
      .click();
    await expect(this.page.getByTestId("architecture-diagram-trigger")).toContainText(title);
  }

  async show(view: "Render" | "Source" | "Validation") {
    await this.page.getByRole("tab", { name: view, exact: true }).click();
  }
}
