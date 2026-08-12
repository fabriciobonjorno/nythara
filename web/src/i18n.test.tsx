import { afterEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { detectLocale, normalizeLocale } from "./locales";
import { formatNumber, translateError, translateText } from "./i18n";
import { localizeDocument } from "./documentLocalization";
import { LanguageSelector } from "./components/LanguageSelector";
import { usePreferencesStore } from "./store";

afterEach(() => {
  cleanup();
  usePreferencesStore.setState({ locale: "pt-BR" });
  document.body.innerHTML = "";
  document.documentElement.lang = "pt-BR";
});

describe("locales suportados", () => {
  it("normaliza variantes e respeita a primeira preferência suportada", () => {
    expect(normalizeLocale("pt-PT")).toBe("pt-BR");
    expect(normalizeLocale("es-MX")).toBe("es");
    expect(normalizeLocale("en-US")).toBe("en");
    expect(detectLocale(["fr-FR", "es-AR", "en"])).toBe("es");
    expect(detectLocale(["fr-FR"])).toBe("pt-BR");
  });
});

describe("catálogo de apresentação", () => {
  it("traduz interface, texto dinâmico e mecânica sem tocar em nomes próprios", () => {
    expect(translateText("Coleção", "es")).toBe("Colección");
    expect(translateText("Carregar mais cartas (14 restantes)", "en")).toBe("Load more cards (14 remaining)");
    expect(translateText("Com 24 ou menos de Vitalidade, seus Assaltos custam 1 a menos.", "en"))
      .toBe("At 24 or less Vitality, your Assaults cost 1 less.");
    expect(translateText("Declare este Assalto com Poder 9. O rival pode responder com uma Guarda.", "es"))
      .toContain("Declara este Asalto con Poder 9");
    expect(translateText("Corte Rubro", "en")).toBe("Corte Rubro");
  });

  it("traduz todas as descrições de acessibilidade sem misturar idiomas", () => {
    const copies = [
      ["Paisagem sonora discreta que reage ao turno e ao perigo; começa após sua primeira ação.", "Paisaje sonoro discreto que reacciona al turno y al peligro; comienza después de tu primera acción.", "A subtle soundscape that reacts to the turn and danger; starts after your first action."],
      ["Vibrações curtas em confrontos e golpes, quando o dispositivo permitir.", "Vibraciones cortas en confrontaciones y golpes, cuando el dispositivo lo permita.", "Short vibrations during confrontations and impacts, when supported by the device."],
      ["Mostra custo, Vitalidade restante e orientação da fase atual.", "Muestra el coste, la Vitalidad restante y la orientación de la fase actual.", "Shows cost, remaining Vitality, and guidance for the current phase."],
      ["Desativa rotações, pulsos e transições decorativas.", "Desactiva rotaciones, pulsos y transiciones decorativas.", "Disables rotations, pulses, and decorative transitions."],
      ["Reforça bordas e contraste dos controles.", "Refuerza los bordes y el contraste de los controles.", "Strengthens borders and control contrast."],
      ["Aumenta o tamanho-base da interface.", "Aumenta el tamaño base de la interfaz.", "Increases the interface's base size."],
    ] as const;

    for (const [pt, es, en] of copies) {
      expect(translateText(pt, "es")).toBe(es);
      expect(translateText(pt, "en")).toBe(en);
    }
  });

  it("traduz os estados da caixa administrativa de sugestões", () => {
    expect(translateText("As sugestões enviadas pelos jogadores aparecem aqui automaticamente.", "en"))
      .toBe("Suggestions sent by players appear here automatically.");
    expect(translateText("Não foi possível carregar as sugestões:", "es"))
      .toBe("No se pudieron cargar las sugerencias:");
  });

  it("localiza erros conhecidos por código e preserva diagnóstico desconhecido", () => {
    expect(translateError("subscriber_too_slow", "texto do servidor", "en"))
      .toBe("The connection fell behind the match. Reconnecting…");
    expect(translateError("custom_code", "diagnóstico 42", "es")).toBe("diagnóstico 42");
  });

  it("formata números no locale ativo", () => {
    expect(formatNumber(12.5, "pt-BR", { minimumFractionDigits: 1 })).toBe("12,5");
    expect(formatNumber(12.5, "en", { minimumFractionDigits: 1 })).toBe("12.5");
  });
});

describe("troca de idioma", () => {
  it("restaura o texto-fonte ao alternar sem recarregar", () => {
    document.body.innerHTML = '<button aria-label="Buscar rival">Buscar rival</button>';
    localizeDocument("en");
    expect(document.querySelector("button")).toHaveTextContent("Find rival");
    expect(document.querySelector("button")).toHaveAttribute("aria-label", "Find rival");
    localizeDocument("es");
    expect(document.querySelector("button")).toHaveTextContent("Buscar rival");
    localizeDocument("pt-BR");
    expect(document.querySelector("button")).toHaveTextContent("Buscar rival");
  });

  it("oferece os três idiomas e persiste a escolha", () => {
    usePreferencesStore.setState({ locale: "pt-BR" });
    render(<LanguageSelector />);
    const select = screen.getByRole("combobox");
    expect(select).toHaveDisplayValue("Português (Brasil)");
    expect(screen.getAllByRole("option")).toHaveLength(3);
    fireEvent.change(select, { target: { value: "en" } });
    expect(usePreferencesStore.getState().locale).toBe("en");
    expect(document.documentElement.lang).toBe("en");
  });
});
