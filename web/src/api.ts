import { usePreferencesStore, useSessionStore } from "./store";
import type { SessionTokens } from "./types";
import { translateError } from "./i18n";

export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}

let refreshInFlight: Promise<SessionTokens> | null = null;

async function parseResponse<T>(response: Response): Promise<T> {
  if (response.status === 204) return undefined as T;
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    const code = body?.error?.code ?? "request_failed";
    const message = body?.error?.message ?? "Não foi possível concluir a solicitação.";
    throw new ApiError(response.status, code, translateError(code, message, usePreferencesStore.getState().locale));
  }
  return body as T;
}

async function refreshSession(): Promise<SessionTokens> {
  const refreshToken = useSessionStore.getState().tokens?.refresh_token;
  if (!refreshToken) throw new ApiError(401, "unauthorized", translateError("unauthorized", "Sua sessão terminou.", usePreferencesStore.getState().locale));
  if (!refreshInFlight) {
    refreshInFlight = fetch("/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
      .then(parseResponse<SessionTokens>)
      .then((tokens) => {
        useSessionStore.getState().setTokens(tokens);
        return tokens;
      })
      .finally(() => { refreshInFlight = null; });
  }
  return refreshInFlight;
}

export async function api<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const token = useSessionStore.getState().tokens?.access_token;
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(path, { ...init, headers });
  if (response.status === 401 && token && retry) {
    try {
      await refreshSession();
      return api<T>(path, init, false);
    } catch (error) {
      useSessionStore.getState().clear();
      throw error;
    }
  }
  return parseResponse<T>(response);
}

export function mutationHeaders(): HeadersInit {
  return { "Idempotency-Key": crypto.randomUUID() };
}

export function websocketURL(path: string): string {
  const url = new URL(path, window.location.href);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}
