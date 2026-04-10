import { expect, test } from "@playwright/test";

test("shows the style preview lab and allows switching themes", async ({
  page,
}) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "ACMRank Frontend Preview Lab" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: /保留洛谷这套蓝灰白颜色/ }),
  ).toBeVisible();

  await page.getByRole("button", { name: /Luogu Profile/i }).click();
  await expect(
    page.getByRole("heading", {
      name: /同一套洛谷配色，换成更正常的“个人页优先”布局/,
    }),
  ).toBeVisible();

  await page.getByRole("button", { name: /Luogu Rankings/i }).click();
  await expect(
    page.getByRole("heading", {
      name: /用洛谷这套颜色，直接把榜单和趋势做成站点主视觉/,
    }),
  ).toBeVisible();
});
