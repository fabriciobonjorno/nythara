import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    if (!window.localStorage.getItem("nythara-preferences")) {
      window.localStorage.setItem("nythara-preferences", JSON.stringify({ state: { locale: "pt-BR" }, version: 0 }));
    }
  });
});

test("sheet confirma exatamente N na ordem visual do toque", async ({ page }) => {
  await page.goto("/e2e/decision-sheet.html");

  const confirm = page.getByRole("button", { name: "Confirmar escolha" });
  await expect(page.getByRole("dialog", { name: "Descarte 2 cartas" })).toBeVisible();
  await expect(confirm).toBeDisabled();

  await page.getByRole("button", { name: "Cinza" }).click();
  await page.getByRole("button", { name: "Vidro" }).click();
  await expect(page.getByRole("button", { name: "Cinza, marcada" }).locator(".decision-option__badge")).toHaveText("1");
  await expect(page.getByRole("button", { name: "Vidro, marcada" }).locator(".decision-option__badge")).toHaveText("2");
  await expect(confirm).toBeEnabled();

  // A terceira opção não ultrapassa o N exigido.
  await page.getByRole("button", { name: "Rubro" }).click();
  await expect(page.getByRole("button", { name: "Rubro" })).toHaveAttribute("aria-pressed", "false");

  // Ao retirar a primeira, a segunda vira 1 e a próxima escolha vira 2.
  await page.getByRole("button", { name: "Cinza, marcada" }).click();
  await expect(page.getByRole("button", { name: "Vidro, marcada" }).locator(".decision-option__badge")).toHaveText("1");
  await page.getByRole("button", { name: "Rubro" }).click();
  await expect(page.getByRole("button", { name: "Rubro, marcada" }).locator(".decision-option__badge")).toHaveText("2");

  await confirm.click();
  await expect(page.getByTestId("confirmed")).toHaveText("b,c");
});

test("troca entre português, espanhol e inglês sem recarregar", async ({ page }) => {
  await page.goto("/e2e/decision-sheet.html");
  const language = page.getByRole("combobox", { name: "Escolher idioma" });
  await expect(language.locator("option")).toHaveCount(3);

  await language.selectOption("es");
  await expect(page.locator("html")).toHaveAttribute("lang", "es");
  await expect(page.getByRole("dialog", { name: "Descarta 2 cartas" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Confirmar elección" })).toBeVisible();

  await page.getByRole("combobox", { name: "Elegir idioma" }).selectOption("en");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(page.getByRole("dialog", { name: "Discard 2 cards" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Confirm choice" })).toBeVisible();

  await page.getByRole("combobox", { name: "Choose language" }).selectOption("pt-BR");
  await expect(page.locator("html")).toHaveAttribute("lang", "pt-BR");
  await expect(page.getByRole("dialog", { name: "Descarte 2 cartas" })).toBeVisible();
});

test("a entrada troca de idioma e conserva a preferência", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Jogue sua carta. Veja o duelo acontecer." })).toBeVisible();

  await page.getByRole("combobox", { name: "Escolher idioma" }).selectOption("es");
  await expect(page.getByRole("heading", { name: "Juega tu carta. Mira cómo sucede el duelo." })).toBeVisible();

  await page.getByRole("combobox", { name: "Elegir idioma" }).selectOption("en");
  await expect(page.getByRole("heading", { name: "Play your card. Watch the duel unfold." })).toBeVisible();
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(page.getByRole("combobox", { name: "Choose language" })).toHaveValue("en");
});
