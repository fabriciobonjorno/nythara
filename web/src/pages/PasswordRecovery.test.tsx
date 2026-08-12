import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { usePreferencesStore } from "../store";
import { ForgotPasswordPage, ResetPasswordPage } from "./PasswordRecovery";

afterEach(() => {
  vi.unstubAllGlobals();
  usePreferencesStore.setState({ locale: "pt-BR" });
});

describe("recuperação de senha", () => {
  it("envia o locale e mostra sempre a resposta não enumerável", async () => {
    usePreferencesStore.setState({ locale: "es" });
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    render(<MemoryRouter><ForgotPasswordPage /></MemoryRouter>);

    fireEvent.change(screen.getByRole("textbox"), { target: { value: "player@example.test" } });
    fireEvent.click(screen.getByRole("button", { name: "Enviar enlace de recuperación" }));

    await screen.findByRole("status");
    expect(fetchMock).toHaveBeenCalledOnce();
    const [path, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(path).toBe("/v1/auth/forgot-password");
    expect(JSON.parse(String(init.body))).toEqual({ email: "player@example.test", locale: "es" });
  });

  it("bloqueia senhas diferentes sem enviar o token", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    render(<MemoryRouter initialEntries={["/reset-password?token=token-opaco-valido-com-mais-de-32-chars"]}>
      <ResetPasswordPage />
    </MemoryRouter>);

    const passwordFields = Array.from(document.querySelectorAll<HTMLInputElement>('input[type="password"]'));
    expect(passwordFields).toHaveLength(2);
    fireEvent.change(passwordFields[0], { target: { value: "senha-segura-2026" } });
    fireEvent.change(passwordFields[1], { target: { value: "senha-diferente-2026" } });
    fireEvent.click(screen.getByRole("button", { name: "Redefinir senha" }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("As senhas não coincidem."));
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
