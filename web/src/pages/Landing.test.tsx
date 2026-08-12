import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../api";
import { usePreferencesStore, useSessionStore } from "../store";
import { Landing } from "./Landing";

vi.mock("../api", () => ({ api: vi.fn() }));
vi.mock("../queries", () => ({ useCards: () => ({ data: { cards: [] } }) }));

beforeEach(() => {
  useSessionStore.setState({ user: null, principal: null, tokens: null, activeMatchId: null, guidedMatchId: null, lastBattle: null });
  usePreferencesStore.setState({ locale: "pt-BR", onboardingUserId: null, tutorialByUser: {} });
});

afterEach(() => {
  vi.mocked(api).mockReset();
  useSessionStore.setState({ user: null, principal: null, tokens: null });
  usePreferencesStore.setState({ onboardingUserId: null, tutorialByUser: {} });
});

describe("primeiro acesso", () => {
  it("leva uma conta recém-criada diretamente ao tutorial", async () => {
    vi.mocked(api).mockImplementation(async (path) => {
      if (path === "/v1/auth/providers") return { google: false } as never;
      return {
        user: { id: "new-user", email: "new@example.test", display_name: "new_player", role: "player", password_set: true, created_at: "2026-08-12T00:00:00Z" },
        tokens: { access_token: "access", refresh_token: "refresh", access_expires_at: "2026-08-12T01:00:00Z", refresh_expires_at: "2026-08-13T00:00:00Z" },
      } as never;
    });

    render(<MemoryRouter><Routes><Route path="/" element={<Landing />} /><Route path="/tutorial" element={<h1>Destino tutorial</h1>} /></Routes></MemoryRouter>);
    fireEvent.click(screen.getByRole("tab", { name: /Criar conta|Crear cuenta|Create account/ }));
    fireEvent.change(screen.getByLabelText(/^(Nome de usuário|Nombre de usuario|Username)/), { target: { value: "new_player" } });
    fireEvent.change(screen.getByLabelText(/^(E-mail|Correo electrónico|Email)/), { target: { value: "new@example.test" } });
    fireEvent.change(screen.getByLabelText(/^(Senha|Contraseña|Password)/), { target: { value: "uma-senha-segura" } });
    fireEvent.click(screen.getByRole("button", { name: /Criar conta gratuita|Crear cuenta gratis|Create free account/ }));

    expect(await screen.findByRole("heading", { name: "Destino tutorial" })).toBeInTheDocument();
    expect(usePreferencesStore.getState().tutorialByUser["new-user"]?.started).toBe(true);
  });

  it("consome convite administrativo no cadastro e abre o salão", async () => {
    vi.mocked(api).mockImplementation(async (path) => {
      if (path === "/v1/auth/providers") return { google: true } as never;
      return {
        user: { id: "admin-user", email: "admin@example.test", display_name: "guardiao", role: "admin", password_set: true, created_at: "2026-08-12T00:00:00Z" },
        tokens: { access_token: "access", refresh_token: "refresh", access_expires_at: "2026-08-12T01:00:00Z", refresh_expires_at: "2026-08-13T00:00:00Z" },
      } as never;
    });

    render(<MemoryRouter initialEntries={["/?admin_invite=convite-opaco"]}><Routes><Route path="/" element={<Landing />} /><Route path="/salao" element={<h1>Destino salão</h1>} /></Routes></MemoryRouter>);
    expect(screen.getByText("Convite administrativo")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Continuar com Google" })).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(/^(Nome de usuário|Nombre de usuario|Username)/), { target: { value: "guardiao" } });
    fireEvent.change(screen.getByLabelText(/^(E-mail|Correo electrónico|Email)/), { target: { value: "admin@example.test" } });
    fireEvent.change(screen.getByLabelText(/^(Senha|Contraseña|Password)/), { target: { value: "uma-senha-segura" } });
    fireEvent.click(screen.getByRole("button", { name: /Criar conta gratuita|Crear cuenta gratis|Create free account/ }));

    expect(await screen.findByRole("heading", { name: "Destino salão" })).toBeInTheDocument();
    const registerCall = vi.mocked(api).mock.calls.find(([path]) => path === "/v1/auth/register");
    expect(registerCall?.[1]?.body).toContain('"admin_invite":"convite-opaco"');
  });
});

describe("entrada com provedores externos", () => {
  it("exibe o Google dentro do formulário quando habilitado pelo servidor", async () => {
    vi.mocked(api).mockResolvedValueOnce({ google: true });

    render(<MemoryRouter><Landing /></MemoryRouter>);

    const link = await screen.findByRole("link", { name: "Continuar com Google" });
    expect(link).toHaveAttribute("href", "/v1/auth/google/start");
    expect(link.closest("form")).not.toBeNull();
  });
});

describe("composição da home", () => {
  it("limita a arte de fundo ao primeiro bloco da página", () => {
    vi.mocked(api).mockResolvedValueOnce({ google: false });

    const { container } = render(<MemoryRouter><Landing /></MemoryRouter>);
    const hero = container.querySelector<HTMLElement>(".landing-hero");
    const art = container.querySelector<HTMLElement>(".landing-art");
    const showcase = container.querySelector<HTMLElement>(".landing-showcase");

    expect(hero).toContainElement(art);
    expect(hero).not.toContainElement(showcase);
  });
});
