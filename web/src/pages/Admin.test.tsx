import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { api } from "../api";
import { AlphaNotesPanel } from "./Admin";

vi.mock("../api", () => ({ api: vi.fn() }));

const suggestion = {
  id: "feedback-1",
  user_id: "player-1",
  match_id: "match-1",
  ruleset_version: "alpha-0.13.0",
  message: "A carta derrotada desaparece rápido demais.",
  created_at: "2026-08-12T20:00:00Z",
};

describe("caixa de sugestões do Salão", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(cleanup);

  it("exibe sugestões e agenda atualização automática", async () => {
    const interval = vi.spyOn(window, "setInterval");
    vi.mocked(api).mockResolvedValue({ feedback: [suggestion] });

    render(<AlphaNotesPanel />);

    expect(await screen.findByText(suggestion.message)).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(interval).toHaveBeenCalledWith(expect.any(Function), 30_000);
    interval.mockRestore();
  });

  it("permite recarregar e não transforma falha em caixa vazia", async () => {
    vi.mocked(api)
      .mockRejectedValueOnce(new Error("serviço indisponível"))
      .mockResolvedValueOnce({ feedback: [suggestion] });

    render(<AlphaNotesPanel />);

    expect(await screen.findByRole("alert")).toHaveTextContent("serviço indisponível");
    fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
    expect(await screen.findByText(suggestion.message)).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
  });
});
