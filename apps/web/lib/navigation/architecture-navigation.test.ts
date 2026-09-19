import { describe, expect, it } from "vitest";
import { APP_DESTINATIONS } from "./core-destinations";

describe("architecture navigation", () => {
  it("publishes architecture to desktop, mobile, and command navigation", () => {
    const architecture = APP_DESTINATIONS.find((destination) => destination.id === "architecture");
    expect(architecture?.href).toBe("/architecture");
    expect(architecture?.surfaces).toEqual(
      expect.arrayContaining(["sidebar", "mobileMenu", "palette"]),
    );
  });
});
