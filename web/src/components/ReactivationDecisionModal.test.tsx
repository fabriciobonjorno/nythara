import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useSessionStore } from "../store";
import { ReactivationDecisionModal } from "./ReactivationDecisionModal";

afterEach(() => {
  vi.unstubAllGlobals();
  useSessionStore.getState().clear();
});

describe("decisão após reativação", () => {
  it("não pode ser dispensada e preserva os dados somente após confirmação do servidor", async () => {
    useSessionStore.setState({
      user: { id: "player", email: "player@example.test", display_name: "Duelista", role: "player",
        password_set: true, reactivation_reset_pending: true, created_at: "2026-08-12T00:00:00Z" },
      principal: { user_id: "player", display_name: "Duelista", role: "player",
        password_set: true, reactivation_reset_pending: true },
      tokens: { access_token: "access", refresh_token: "refresh",
        access_expires_at: "2026-08-12T01:00:00Z", refresh_expires_at: "2026-09-12T01:00:00Z" },
    });
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      user_id: "player", display_name: "Duelista", role: "player", password_set: true,
      reactivation_reset_pending: false,
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new QueryClient();
    render(<QueryClientProvider client={client}><MemoryRouter><ReactivationDecisionModal /></MemoryRouter></QueryClientProvider>);

    expect(screen.getByRole("dialog", { name: "Como você quer retornar?" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /fechar/i })).not.toBeInTheDocument();
	const keepData = screen.getByRole("button", { name: /Manter meus dados/i });
	expect(keepData).toHaveFocus();
	fireEvent.click(keepData);

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(fetchMock).toHaveBeenCalledOnce();
    const [path, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(path).toBe("/v1/me/reactivation");
    expect(JSON.parse(String(init.body))).toEqual({ reset_data: false });
    expect(useSessionStore.getState().principal?.reactivation_reset_pending).toBe(false);
  });
});
