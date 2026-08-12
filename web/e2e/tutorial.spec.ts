import { expect, test } from "@playwright/test";

test("o tutorial conduz pelas telas e registra cada etapa concluída", async ({ page }) => {
  await page.goto("/e2e/tutorial.html");

  await expect(page.getByRole("heading", { name: "Você sempre saberá o próximo passo." })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Etapas do guia" }).getByRole("button")).toHaveCount(6);

  await page.getByRole("button", { name: "Entendi o objetivo" }).click();
  await expect(page.getByRole("button", { name: /Etapa 1:.*concluída/ })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Conheça os dez Avatares" })).toBeVisible();

  await page.getByRole("button", { name: "Abrir Avatares" }).click();
  await expect(page.getByRole("heading", { name: "Destino Avatares" })).toBeVisible();
  await expect(page.getByTestId("completed")).toHaveText("goal,avatars");
});
