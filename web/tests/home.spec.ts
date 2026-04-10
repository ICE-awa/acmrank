import { expect, test } from "@playwright/test";

test("shows the style preview lab and allows switching themes", async ({
  page,
}) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "ACMRank Frontend Preview Lab" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Contest Signals, Clean Facts" }),
  ).toBeVisible();

  await page.getByRole("button", { name: /Archive Ledger/i }).click();
  await expect(
    page.getByRole("heading", { name: /训练档案像校史馆一样可靠/ }),
  ).toBeVisible();

  await page.getByRole("button", { name: /Trackside Pulse/i }).click();
  await expect(
    page.getByRole("heading", {
      name: "把训练曲线做成一张有速度感的竞赛战报",
    }),
  ).toBeVisible();
});
