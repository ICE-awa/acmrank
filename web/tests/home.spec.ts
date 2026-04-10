import { expect, test } from "@playwright/test";

test("shows the style preview lab and allows switching themes", async ({
  page,
}) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "ACMRank 风格预览" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: /门户布局：总览/ }),
  ).toBeVisible();

  await page.getByRole("button", { name: /个人布局/ }).click();
  await expect(
    page.getByRole("heading", {
      name: /个人布局：信息卡 \+ 曲线 \+ 过题/,
    }),
  ).toBeVisible();

  await page.getByRole("button", { name: /榜单布局/ }).click();
  await expect(
    page.getByRole("heading", {
      name: /榜单布局：排行榜优先/,
    }),
  ).toBeVisible();
});
