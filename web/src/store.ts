import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { detectLocale, type Locale } from "./locales";
import type { LastBattle, Principal, SessionTokens, User } from "./types";

function migrateLegacyStorage(storage: Storage, legacyKey: string, nextKey: string) {
  try {
    if (storage.getItem(nextKey)) return;
    const legacyValue = storage.getItem(legacyKey);
    if (!legacyValue) return;
    storage.setItem(nextKey, legacyValue);
    storage.removeItem(legacyKey);
  } catch {
    // Persistência pode estar bloqueada pelo navegador; a aplicação segue em memória.
  }
}

if (typeof window !== "undefined") {
  migrateLegacyStorage(sessionStorage, "veu-rubro-session", "nythara-session");
  migrateLegacyStorage(localStorage, "veu-rubro-preferences", "nythara-preferences");
}

interface SessionState {
  user: User | null;
  principal: Principal | null;
  tokens: SessionTokens | null;
  activeMatchId: string | null;
  guidedMatchId: string | null;
  lastBattle: LastBattle | null;
  setAuth: (user: User, tokens: SessionTokens) => void;
  setPrincipal: (principal: Principal) => void;
  setTokens: (tokens: SessionTokens) => void;
  setActiveMatch: (matchId: string | null) => void;
  setGuidedMatch: (matchId: string | null) => void;
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
      guidedMatchId: null,
      lastBattle: null,
	  setAuth: (user, tokens) => set({ user, tokens, principal: { user_id: user.id, display_name: user.display_name,
		role: user.role, avatar_id: user.avatar_id, password_set: user.password_set } }),
	  setPrincipal: (principal) => set((state) => ({ principal, user: state.user ? { ...state.user,
		display_name: principal.display_name, avatar_id: principal.avatar_id, password_set: principal.password_set } : null })),
      setTokens: (tokens) => set({ tokens }),
      setActiveMatch: (activeMatchId) => set({ activeMatchId }),
      setGuidedMatch: (guidedMatchId) => set({ guidedMatchId }),
      setLastBattle: (lastBattle) => set({ lastBattle, activeMatchId: null }),
      clear: () => set({ user: null, principal: null, tokens: null, activeMatchId: null, guidedMatchId: null, lastBattle: null }),
    }),
    { name: "nythara-session", storage: createJSONStorage(() => sessionStorage) },
  ),
);

interface PreferencesState {
  locale: Locale;
  animationPace: "cinematic" | "normal" | "quick";
  reducedMotion: boolean;
  highContrast: boolean;
  sound: boolean;
  ambience: boolean;
  haptics: boolean;
  combatHints: boolean;
  largeText: boolean;
  onboardingUserId: string | null;
  set: (key: "reducedMotion" | "highContrast" | "sound" | "ambience" | "haptics" | "combatHints" | "largeText", value: boolean) => void;
  setAnimationPace: (value: "cinematic" | "normal" | "quick") => void;
  completeOnboarding: (userId: string) => void;
  restartOnboarding: () => void;
  setLocale: (locale: Locale) => void;
}

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set) => ({
      locale: detectLocale(),
      animationPace: "cinematic",
      reducedMotion: false,
      highContrast: false,
      sound: true,
      ambience: true,
      haptics: true,
      combatHints: true,
      largeText: false,
      onboardingUserId: null,
      set: (key, value) => set({ [key]: value }),
      setAnimationPace: (animationPace) => set({ animationPace }),
      completeOnboarding: (onboardingUserId) => set({ onboardingUserId }),
      restartOnboarding: () => set({ onboardingUserId: null }),
      setLocale: (locale) => set({ locale }),
    }),
    { name: "nythara-preferences", storage: createJSONStorage(() => localStorage) },
  ),
);
