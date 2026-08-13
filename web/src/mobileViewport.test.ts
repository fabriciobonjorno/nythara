import { afterEach, describe, expect, it, vi } from "vitest";
import { hasFinePointer, releaseTextEntryFocus, settleRouteViewport } from "./mobileViewport";

afterEach(() => {
  document.body.innerHTML = "";
  vi.restoreAllMocks();
});

describe("viewport móvel", () => {
  it("libera somente campos de texto ativos", () => {
    document.body.innerHTML = '<input aria-label="E-mail"><button>Entrar</button>';
    const input = document.querySelector("input")!;
    const button = document.querySelector("button")!;
    input.focus();
    expect(releaseTextEntryFocus()).toBe(true);
    expect(document.activeElement).not.toBe(input);
    button.focus();
    expect(releaseTextEntryFocus()).toBe(false);
    expect(document.activeElement).toBe(button);
  });

  it("recompõe o topo e move o foco acessível para o título da nova rota", () => {
    vi.useFakeTimers();
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => window.setTimeout(callback, 0));
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation((timer) => window.clearTimeout(timer));
    document.body.innerHTML = '<main id="main-content"><h1>Início</h1></main>';

    const cleanup = settleRouteViewport();
    vi.runOnlyPendingTimers();

    expect(scrollTo).toHaveBeenCalledWith({ top: 0, left: 0, behavior: "auto" });
    expect(document.activeElement).toBe(document.querySelector("h1"));
    cleanup();
    vi.useRealTimers();
  });

  it("só permite autofoco editorial em dispositivos de ponteiro fino", () => {
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: true }));
    expect(hasFinePointer()).toBe(true);
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: false }));
    expect(hasFinePointer()).toBe(false);
    vi.unstubAllGlobals();
  });
});
