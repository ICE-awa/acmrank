import { expect, test } from "@playwright/test";

test("shows the ACMRank bootstrap page", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "ACMRank" })).toBeVisible();
  await expect(
    page.getByText("个人页保留 SCNU Rating 折线图与每日新 AC 热力图入口"),
  ).toBeVisible();
});
