import { expect, test } from "@playwright/test";

test("shows the style preview lab and allows switching themes", async ({
  page,
}) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "ACMRank Frontend Preview Lab" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: /黑白灰的训练档案页/ }),
  ).toBeVisible();

  await page.getByRole("button", { name: /Paper Column/i }).click();
  await expect(
    page.getByRole("heading", { name: /白底黑字，像公开档案册/ }),
  ).toBeVisible();

  await page.getByRole("button", { name: /Steel Frame/i }).click();
  await expect(
    page.getByRole("heading", { name: /中性、规整、克制/ }),
  ).toBeVisible();
});
