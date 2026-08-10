import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { LastBattle, Principal, SessionTokens, User } from "./types";

interface SessionState {
  user: User | null;
  principal: Principal | null;
  tokens: SessionTokens | null;
  activeMatchId: string | null;
  lastBattle: LastBattle | null;
  setAuth: (user: User, tokens: SessionTokens) => void;
  setPrincipal: (principal: Principal) => void;
  setTokens: (tokens: SessionTokens) => void;
  setActiveMatch: (matchId: string | null) => void;
  setLastBattle: (battle: LastBattle) => void;
  clear: () => void;
}

export const useSessionStore = create<SessionState>()(
  persist(
    (set) => ({
      user: null,
      principal: null,
      tokens: null,
      activeMatchId: null,
      lastBattle: null,
      setAuth: (user, tokens) => set({ user, tokens, principal: { user_id: user.id, display_name: user.display_name, role: user.role } }),
      setPrincipal: (principal) => set({ principal }),
      setTokens: (tokens) => set({ tokens }),
      setActiveMatch: (activeMatchId) => set({ activeMatchId }),
      setLastBattle: (lastBattle) => set({ lastBattle, activeMatchId: null }),
      clear: () => set({ user: null, principal: null, tokens: null, activeMatchId: null, lastBattle: null }),
    }),
    { name: "veu-rubro-session", storage: createJSONStorage(() => sessionStorage) },
  ),
);

interface PreferencesState {
  reducedMotion: boolean;
  highContrast: boolean;
  sound: boolean;
  largeText: boolean;
  set: (key: "reducedMotion" | "highContrast" | "sound" | "largeText", value: boolean) => void;
}

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set) => ({
      reducedMotion: false,
      highContrast: false,
      sound: true,
      largeText: false,
      set: (key, value) => set({ [key]: value }),
    }),
    { name: "veu-rubro-preferences", storage: createJSONStorage(() => localStorage) },
  ),
);
