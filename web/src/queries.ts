import { useQuery } from "@tanstack/react-query";
import { api } from "./api";
import { CURRENT_RULESET } from "./rules";
import type { CardDefinition, Champion, Collection, Deck, LeaderboardEntry, MatchReplayData, MatchSummary, Principal, ProgressSummary, RankedStanding, Season } from "./types";

export const useCards = () => useQuery({
  queryKey: ["catalog", "cards"],
  queryFn: () => api<{ ruleset_version: string; cards: CardDefinition[] }>("/v1/catalog/cards"),
  staleTime: 60 * 60 * 1000,
});

export const useChampions = () => useQuery({
  queryKey: ["catalog", "champions"],
  queryFn: () => api<{ ruleset_version: string; champions: Champion[] }>("/v1/catalog/champions"),
  staleTime: 60 * 60 * 1000,
});

export const useRuleset = () => useQuery({
  queryKey: ["ruleset", "current"],
  queryFn: () => api<{
    version: string;
    active: boolean;
    starting_vitality: number;
    guard_leak_cap: number;
    pressure_start_turn: number;
    pressure_base_loss: number;
    deck_size: number;
    min_assaults: number;
    min_guards: number;
    min_rites: number;
  }>("/v1/rulesets/current"),
  staleTime: 60 * 60 * 1000,
});

/** Versão publicada pelo servidor; a constante local é só rede de segurança. */
export const useActiveRulesetVersion = () => useRuleset().data?.version ?? CURRENT_RULESET;

export const useCollection = () => useQuery({ queryKey: ["collection"], queryFn: () => api<Collection>("/v1/collection") });
export const useDecks = () => useQuery({ queryKey: ["decks"], queryFn: () => api<{ decks: Deck[] }>("/v1/decks") });
export const useMe = () => useQuery({ queryKey: ["me"], queryFn: () => api<Principal>("/v1/me") });
export const useSeason = () => useQuery({ queryKey: ["season"], queryFn: () => api<Season>("/v1/seasons/current"), staleTime: 10 * 60 * 1000 });
export const useProgress = () => useQuery({ queryKey: ["progress"], queryFn: () => api<ProgressSummary>("/v1/progress"), staleTime: 60 * 1000 });
export const useLeaderboard = () => useQuery({
  queryKey: ["leaderboard"],
  queryFn: () => api<{ entries: LeaderboardEntry[]; me: RankedStanding | null }>("/v1/ranked/leaderboard?limit=20"),
  staleTime: 60 * 1000,
});
export const useMatchHistory = () => useQuery({
  queryKey: ["matches", "history"],
  queryFn: () => api<{ matches: MatchSummary[] | null }>("/v1/matches?limit=20"),
  staleTime: 30 * 1000,
});
export const useMatchReplay = (matchId: string) => useQuery({
  queryKey: ["matches", "replay", matchId],
  queryFn: () => api<MatchReplayData>(`/v1/matches/${matchId}/replay`),
  staleTime: Infinity,
  enabled: Boolean(matchId),
});
