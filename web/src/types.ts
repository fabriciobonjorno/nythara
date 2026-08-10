export type CardType = "Assalto" | "Guarda" | "Rito" | "Relíquia" | "Manifestação";

export interface CardDefinition {
  id: string;
  name: string;
  faction: string;
  type: CardType;
  rarity: "Comum" | "Incomum" | "Rara" | "Épica" | "Lendária";
  cost: number;
  eclipse_shift: number;
  sigil: string;
  rules_text: string;
  flavor: string;
  design_role: string;
}

export interface Champion {
  id: string;
  name: string;
  faction: string;
  vitality: number;
  passive: string;
  ultimate: string;
  eclipse_form: string;
}

export interface Principal {
  user_id: string;
  role: "player" | "admin";
  display_name: string;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  role: "player" | "admin";
  created_at: string;
}

export interface SessionTokens {
  access_token: string;
  refresh_token: string;
  access_expires_at: string;
  refresh_expires_at: string;
}

export interface AuthEnvelope {
  user: User;
  tokens: SessionTokens;
}

export interface Collection {
  ruleset_version: string;
  cards: Array<{ card_id: string; quantity: number }>;
  champions: string[];
}

export interface DeckCard {
  card_id: string;
  quantity: number;
}

export interface Deck {
  id: string;
  user_id: string;
  name: string;
  champion_id: string;
  ruleset_version: string;
  cards: DeckCard[];
  version: number;
  created_at: string;
  updated_at: string;
}

export interface Season {
  id: string;
  name: string;
  ruleset_version: string;
  starts_at: string;
  ends_at?: string;
}

export interface QueueResult {
  status: "idle" | "queued" | "matched";
  match_id?: string;
  slot?: number;
}

export interface BattleTicket {
  ticket: string;
  match_id: string;
  mode: "player" | "spectator";
  slot: number;
  expires_at: string;
}

export interface BattleEvent {
  seq: number;
  round: number;
  kind: string;
  p: number;
  card?: string;
  def?: string;
  n: number;
  from: number;
  to: number;
  s?: string;
}

export interface PlayerView {
  champion: string;
  vitality: number;
  max_vitality: number;
  essence: number;
  essence_cap: number;
  temp_essence: number;
  deck_count: number;
  hand_count: number;
  hand?: string[];
  discard: string[];
  exile: string[];
  relics: string[];
  manifs: string[];
  stance?: string;
  stance_committed: boolean;
  mulligan_done: boolean;
  fatigue: number;
  trail: string[];
  ward: number;
  exposto: boolean;
  ultimate_used: boolean;
}

export interface BattleState {
  ruleset_version: string;
  round: number;
  phase: "mulligan" | "postura" | "rito" | "confronto" | "crepusculo" | "fim";
  initiative: number;
  active: number;
  eclipse: number;
  eclipse_state?: string;
  players: [PlayerView, PlayerView];
  cards: Record<string, { id: string; def: string; owner: number; zone: string }>;
  pending?: { id: number; player: number; kind: string; options?: string[]; n: number; min_n?: number; card?: string; source?: string };
  guard?: { attacker: number; defender: number; assault_inst: string; assault_def: string; base_damage: number; instances: number };
  rite_react?: { caster: number; inst: string; def: string; cost: number };
  extra?: { player: number; max_cost?: number; source: string };
  over: boolean;
  winner: number;
  end_reason?: string;
}

export interface BattleServerMessage {
  type: "sync" | "ready" | "events" | "match_cancelled" | "error";
  match_id?: string;
  status?: "waiting_ready" | "active" | "finished" | "cancelled";
  slot?: number;
  ready?: [boolean, boolean];
  state?: BattleState;
  events?: BattleEvent[];
  client_sequence?: number;
  duplicate?: boolean;
  deadline?: string;
  error?: { code: string; message: string };
}

export interface BattleIntent {
  kind: "mulligan" | "stance" | "play" | "choose" | "pass" | "concede" | "activate" | "ultimate";
  card?: string;
  cards?: string[];
  stance?: "Predação" | "Vigília" | "Arcano";
  decision_id?: number;
}

export interface LastBattle {
  matchId: string;
  slot: number;
  state: BattleState;
  events: BattleEvent[];
  finishedAt: string;
}

export interface RitualState {
  ritual_id: string;
  title: string;
  description: string;
  progress: number;
  target: number;
  reward: number;
  completed_at?: string;
}

export interface ChampionMastery {
  champion_id: string;
  xp: number;
  level: number;
  games: number;
  wins: number;
}

export interface RankedStanding {
  season_id: string;
  rating: number;
  games: number;
  wins: number;
  position?: number;
}

export interface ProgressSummary {
  day: string;
  rituals: RitualState[];
  fragments: number;
  mastery: ChampionMastery[];
  ranked?: RankedStanding;
}
