import { useQuery } from "@tanstack/react-query";
import { api } from "./api";
import type { CardDefinition, Champion, Collection, Deck, Principal, Season } from "./types";

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

export const useCollection = () => useQuery({ queryKey: ["collection"], queryFn: () => api<Collection>("/v1/collection") });
export const useDecks = () => useQuery({ queryKey: ["decks"], queryFn: () => api<{ decks: Deck[] }>("/v1/decks") });
export const useMe = () => useQuery({ queryKey: ["me"], queryFn: () => api<Principal>("/v1/me") });
export const useSeason = () => useQuery({ queryKey: ["season"], queryFn: () => api<Season>("/v1/seasons/current"), staleTime: 10 * 60 * 1000 });
