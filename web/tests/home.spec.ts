import { expect, test } from "@playwright/test";

test("shows the ACMRank dashboard preview", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "我的主页" })).toBeVisible();
  await expect(page.getByText("搜索用户、题号或比赛 ID")).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "SCNU Rating" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Daily New AC" }),
  ).toBeVisible();
  await expect(page.getByRole("heading", { name: "最新动态" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "排行榜" })).toBeVisible();
});
