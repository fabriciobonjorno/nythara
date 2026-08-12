import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "../api";
import { useSessionStore } from "../store";
import { Landing } from "./Landing";

vi.mock("../api", () => ({ api: vi.fn() }));
vi.mock("../queries", () => ({ useCards: () => ({ data: { cards: [] } }) }));

afterEach(() => {
  vi.mocked(api).mockReset();
  useSessionStore.setState({ user: null, principal: null, tokens: null });
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
