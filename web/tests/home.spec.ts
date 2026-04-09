import { expect, test } from "@playwright/test";

test("shows the ACMRank bootstrap page", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "ACMRank" })).toBeVisible();
  await expect(page.getByText("SCNU Rating")).toBeVisible();
});
