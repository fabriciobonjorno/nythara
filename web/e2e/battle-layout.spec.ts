import { expect, test } from "@playwright/test";

for (const viewport of [
  { name: "telefone mínimo", width: 182, height: 696 },
  { name: "telefone estreito", width: 240, height: 696 },
  { name: "telefone", width: 390, height: 844 },
  { name: "desktop largo", width: 1895, height: 858 },
]) {
  test(`mesa ocupa a tela sem corte em ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.goto("/e2e/battle-layout.html");

    const overflow = await page.evaluate(() => ({
      body: document.body.scrollWidth - window.innerWidth,
      room: document.querySelector<HTMLElement>(".duel-room")!.scrollWidth
        - document.querySelector<HTMLElement>(".duel-room")!.clientWidth,
      duelists: [...document.querySelectorAll<HTMLElement>(".duelist")]
        .map((element) => element.scrollWidth - element.clientWidth),
    }));
    expect(overflow.body).toBeLessThanOrEqual(0);
    expect(overflow.room).toBeLessThanOrEqual(0);
    expect(overflow.duelists.every((value) => value <= 0)).toBe(true);
  });
}

test("cartas usam melhor o campo em desktop largo", async ({ page }) => {
  await page.setViewportSize({ width: 1895, height: 858 });
  await page.goto("/e2e/battle-layout.html");

  const tableCard = page.locator(".arena-slot__card .duel-card");
  const handCard = page.locator(".size-hand").first();
  await expect(tableCard).toBeVisible();
  expect((await tableCard.boundingBox())!.width).toBeGreaterThanOrEqual(175);
  expect((await handCard.boundingBox())!.width).toBeGreaterThanOrEqual(160);
});
