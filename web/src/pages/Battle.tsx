import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { CardTile, MiniBattleCard } from "../components/CardTile";
import { EclipseStage } from "../components/EclipseStage";
import { useCards, useChampions } from "../queries";
import { usePreferencesStore } from "../store";
import type { BattleEvent, BattleIntent, BattleState, CardDefinition } from "../types";
import { useBattleSocket } from "../useBattleSocket";

export function BattlePage() {
  const { matchId = "" } = useParams();
  const navigate = useNavigate();
  const { data: cardData } = useCards();
  const { data: championData } = useChampions();
  const reducedMotion = usePreferencesStore((store) => store.reducedMotion);
  const battle = useBattleSocket(matchId);
  const [mulligan, setMulligan] = useState<string[]>([]);
  const [choices, setChoices] = useState<string[]>([]);
  const [zoomed, setZoomed] = useState<CardDefinition | null>(null);
  const [logOpen, setLogOpen] = useState(false);
  const cardsById = useMemo(() => new Map(cardData?.cards.map((card) => [card.id, card]) ?? []), [cardData]);
  const champions = useMemo(() => new Map(championData?.champions.map((champion) => [champion.id, champion]) ?? []), [championData]);
  useEffect(() => { setChoices([]); }, [battle.state?.pending?.id]);

  if (!battle.state || battle.slot === null) {
    return <main className="battle-loading"><div className="queue-orbit is-searching" aria-hidden="true"><span className="orbit orbit-a" /><span className="orbit orbit-b" /><strong>◐</strong></div><p className="eyebrow">SALA AUTHORITATIVE</p><h1>{battle.status === "reconnecting" ? "Restaurando o círculo…" : "Preparando a mesa…"}</h1><p>{battle.error ?? "Sincronizando estado protegido e aguardando os dois jogadores."}</p><Link className="ghost-button" to="/queue">Voltar à fila</Link></main>;
  }

  const state = battle.state;
  const mySlot = battle.slot;
  const me = state.players[mySlot];
  const opponentSlot = 1 - mySlot;
  const opponent = state.players[opponentSlot];
  const actor = expectedActor(state);
  const myTurn = actor === mySlot && battle.status === "connected" && !battle.pending;
  const hand = (me.hand ?? []).map((instanceId) => ({ instanceId, card: cardsById.get(state.cards[instanceId]?.def) })).filter((item): item is { instanceId: string; card: CardDefinition } => Boolean(item.card));
  const playable = (card: CardDefinition) => {
    if (!myTurn) return false;
    if (state.phase === "mulligan") return true;
    if (state.guard || state.rite_react) return card.type === "Guarda";
    if (state.phase === "rito") return card.type === "Rito" || card.type === "Relíquia" || card.type === "Manifestação";
    if (state.phase === "confronto") return card.type === "Assalto";
    return false;
  };
  const play = (instanceId: string) => {
    if (state.phase === "mulligan") {
      setMulligan((current) => current.includes(instanceId) ? current.filter((id) => id !== instanceId) : [...current, instanceId]);
    } else battle.send({ kind: "play", card: instanceId });
  };

  return <main className="battle-page">
    <EclipseStage value={state.eclipse} reducedMotion={reducedMotion} />
    <header className="battle-topbar"><Link to="/app" aria-label="Sair para o início">◐</Link><div><span className={`connection-dot ${battle.status}`} />{battle.status === "connected" ? "Conectado" : "Reconectando"}</div><strong>RODADA {state.round} · {state.phase.toLocaleUpperCase("pt-BR")}</strong><Deadline value={battle.deadline} /><button type="button" onClick={() => setLogOpen((open) => !open)} aria-expanded={logOpen}>☷ <span>Histórico</span></button></header>

    <section className="player-rack opponent-rack" aria-label="Oponente"><ChampionPlate name={champions.get(opponent.champion)?.name ?? opponent.champion} player={opponent} active={actor === opponentSlot} /><Permanents player={opponent} state={state} cards={cardsById} /><div className="opponent-hand" aria-label={`${opponent.hand_count} cartas na mão`}>{Array.from({ length: Math.min(opponent.hand_count, 8) }, (_, index) => <span key={index} />)}<b>{opponent.hand_count}</b></div></section>

    <section className="battle-center">
      <div className="eclipse-meter" aria-label={`Eclipse em ${state.eclipse}; ${state.eclipse_state || "sem estado total"}`}><div className="eclipse-label"><span>AURORA</span><strong>{state.eclipse_state || "ECLIPSE"}</strong><span>NOITE</span></div><div className="eclipse-track">{[-3, -2, -1, 0, 1, 2, 3].map((value) => <span className={value === state.eclipse ? "is-current" : ""} key={value}><i />{value > 0 ? `+${value}` : value}</span>)}</div></div>
      {state.guard && <div className={`guard-banner ${state.guard.defender === mySlot ? "is-urgent" : ""}`} role="status"><span>⬡</span><div><strong>Janela de Guarda</strong><small>{state.guard.defender === mySlot ? `Reaja ao Assalto de ${state.guard.base_damage} de dano ou passe.` : "Aguardando a reação do defensor."}</small></div></div>}
      {state.rite_react && <div className="guard-banner is-urgent" role="status"><span>◈</span><div><strong>Janela de reação</strong><small>Um Rito direcionado pode ser anulado por uma Guarda compatível.</small></div></div>}
      {state.pending && <DecisionPanel state={state} mySlot={mySlot} choices={choices} setChoices={setChoices} send={(intent) => { if (battle.send(intent)) setChoices([]); }} cards={cardsById} />}
      {!state.pending && <ActionPanel state={state} mySlot={mySlot} myTurn={myTurn} mulligan={mulligan} send={battle.send} onMulligan={() => { if (battle.send({ kind: "mulligan", cards: mulligan })) setMulligan([]); }} />}
      {battle.error && <p className="battle-error" role="alert">{battle.error}</p>}
    </section>

    <section className="player-rack own-rack" aria-label="Seu campo"><div className="resonance" aria-label={`Sua Ressonância: ${me.trail.join(", ") || "vazia"}`}><span>RESSONÂNCIA</span><div>{Array.from({ length: Math.max(5, me.trail.length) }, (_, index) => <i className={me.trail[index] ? "filled" : ""} key={index}>{me.trail[index]?.slice(0, 1) ?? ""}</i>)}</div></div><Permanents player={me} state={state} cards={cardsById} /><ChampionPlate name={champions.get(me.champion)?.name ?? me.champion} player={me} active={myTurn} own /></section>

    <section className="battle-hand" aria-label="Sua mão">{hand.map(({ instanceId, card }) => <MiniBattleCard key={instanceId} card={card} instanceId={instanceId} selected={mulligan.includes(instanceId)} onPlay={playable(card) ? () => play(instanceId) : undefined} onZoom={setZoomed} />)}</section>

    <aside className={`battle-log ${logOpen ? "is-open" : ""}`} aria-label="Histórico de eventos"><header><div><p className="eyebrow">LINHA DO TEMPO</p><h2>Histórico</h2></div><button type="button" onClick={() => setLogOpen(false)} aria-label="Fechar histórico">×</button></header><div>{battle.events.slice().reverse().map((event) => <EventLine event={event} key={event.seq} cards={cardsById} ownSlot={mySlot} />)}</div></aside>

    {state.over && <div className="match-over-overlay" role="dialog" aria-modal="true"><p className="eyebrow">O VÉU SE FECHOU</p><h2>{state.winner === mySlot ? "Vitória" : "Derrota"}</h2><p>{endReason(state.end_reason)}</p><button className="primary-button" type="button" onClick={() => navigate("/result")}>Ver resultado</button></div>}
    {zoomed && <div className="modal-backdrop" onMouseDown={() => setZoomed(null)}><div className="battle-card-zoom" onMouseDown={(event) => event.stopPropagation()}><button className="modal-close" onClick={() => setZoomed(null)} aria-label="Fechar">×</button><CardTile card={zoomed} /></div></div>}
  </main>;
}

function ChampionPlate({ name, player, active, own = false }: { name: string; player: BattleState["players"][number]; active: boolean; own?: boolean }) {
  return <div className={`champion-plate ${active ? "is-active" : ""}`}><span className="champion-token" aria-hidden="true">{own ? "♙" : "♟"}</span><div><strong>{name}</strong><small>{active ? "JANELA ATIVA" : player.stance || "AGUARDANDO"}</small></div><span className="resource">◆ {player.essence}<small>ESSÊNCIA</small></span><span className="health">♥ {player.vitality}<small>VITALIDADE</small></span>{player.ward > 0 && <span className="ward">⬡ {player.ward}</span>}</div>;
}

function Permanents({ player, state, cards }: { player: BattleState["players"][number]; state: BattleState; cards: Map<string, CardDefinition> }) {
  const ids = [...player.relics, ...player.manifs];
  return <div className="permanent-row" aria-label="Relíquias e Manifestações">{ids.map((id) => { const def = cards.get(state.cards[id]?.def); return <span title={def?.rules_text} key={id}>{def?.name?.slice(0, 2).toLocaleUpperCase("pt-BR") ?? "?"}</span>; })}{Array.from({ length: Math.max(0, 4 - ids.length) }, (_, index) => <i key={index} />)}</div>;
}

function ActionPanel({ state, mySlot, myTurn, mulligan, send, onMulligan }: { state: BattleState; mySlot: number; myTurn: boolean; mulligan: string[]; send: (intent: BattleIntent) => boolean; onMulligan: () => void }) {
  if (!myTurn) return <div className="action-panel waiting"><span className="pulse-dot" /><div><strong>Aguardando o oponente</strong><small>Você pode explorar as cartas e o histórico enquanto espera.</small></div></div>;
  if (state.phase === "mulligan") return <div className="action-panel"><div><strong>Escolha seu mulligan</strong><small>Marque as cartas que deseja devolver. Você também pode manter todas.</small></div><button className="primary-button" onClick={onMulligan}>{mulligan.length ? `Trocar ${mulligan.length}` : "Manter a mão"}</button></div>;
  if (state.phase === "postura") return <div className="action-panel stance-actions"><div><strong>Escolha sua Postura</strong><small>As escolhas serão reveladas juntas.</small></div>{(["Predação", "Vigília", "Arcano"] as const).map((stance) => <button type="button" onClick={() => send({ kind: "stance", stance })} key={stance}>{stance}</button>)}</div>;
  const permanents = [...state.players[mySlot].relics, ...state.players[mySlot].manifs];
  return <div className="action-panel"><div><strong>{state.guard ? "Defenda-se agora" : state.rite_react ? "Reaja ao Rito" : "Sua janela está aberta"}</strong><small>{state.guard || state.rite_react ? "Jogue uma Guarda da mão ou passe." : "Jogue uma carta compatível, ative um efeito ou encerre sua janela."}</small></div><div className="action-buttons">{state.phase === "rito" && permanents.map((card) => <button type="button" className="secondary-button" onClick={() => send({ kind: "activate", card })} key={card}>Ativar</button>)}<button type="button" className="secondary-button" onClick={() => send({ kind: "ultimate" })}>Ultimate</button><button type="button" className="ghost-button" onClick={() => send({ kind: "pass" })}>Passar</button><button type="button" className="danger-text" onClick={() => window.confirm("Conceder esta partida?") && send({ kind: "concede" })}>Conceder</button></div></div>;
}

function DecisionPanel({ state, mySlot, choices, setChoices, send, cards }: { state: BattleState; mySlot: number; choices: string[]; setChoices: (value: string[]) => void; send: (intent: BattleIntent) => void; cards: Map<string, CardDefinition> }) {
  const decision = state.pending!;
  if (decision.player !== mySlot) return <div className="action-panel waiting"><span className="pulse-dot" /><div><strong>Oponente decidindo</strong><small>Uma escolha obrigatória está em resolução.</small></div></div>;
  const max = decision.n || 1;
  const min = decision.min_n ?? max;
  const toggle = (option: string) => setChoices(choices.includes(option) ? choices.filter((item) => item !== option) : choices.length < max ? [...choices, option] : choices);
  return <div className="decision-panel"><div><p className="eyebrow">DECISÃO OBRIGATÓRIA</p><strong>{decisionLabel(decision.kind)}</strong><small>Escolha {min === max ? max : `${min}–${max}`} opção(ões).</small></div><div className="decision-options">{decision.options?.map((option) => { const card = cards.get(state.cards[option]?.def); return <button type="button" className={choices.includes(option) ? "is-selected" : ""} key={option} onClick={() => toggle(option)}>{card?.name ?? option}</button>; })}</div><button className="primary-button" disabled={choices.length < min || choices.length > max} onClick={() => send({ kind: "choose", cards: choices, decision_id: decision.id })}>Confirmar escolha</button></div>;
}

function Deadline({ value }: { value: string | null }) {
  const [seconds, setSeconds] = useState(0);
  useEffect(() => { const update = () => setSeconds(value ? Math.max(0, Math.ceil((new Date(value).getTime() - Date.now()) / 1000)) : 0); update(); const id = window.setInterval(update, 500); return () => window.clearInterval(id); }, [value]);
  return <span className={`battle-timer ${seconds <= 10 ? "is-low" : ""}`} aria-label={`${seconds} segundos restantes`}>◷ {seconds || "—"}</span>;
}

function EventLine({ event, cards, ownSlot }: { event: BattleEvent; cards: Map<string, CardDefinition>; ownSlot: number }) {
  return <article><span>{event.seq + 1}</span><div><strong>{eventTitle(event.kind)}</strong><small>{event.p < 0 ? "Mesa" : event.p === ownSlot ? "Você" : "Oponente"}{event.def ? ` · ${cards.get(event.def)?.name ?? event.def}` : ""}{event.n ? ` · ${event.n}` : ""}</small></div></article>;
}

function expectedActor(state: BattleState) {
  if (state.pending) return state.pending.player;
  const order = [state.initiative, 1 - state.initiative];
  if (state.phase === "mulligan") return order.find((slot) => !state.players[slot].mulligan_done) ?? state.active;
  if (state.phase === "postura") return order.find((slot) => !state.players[slot].stance_committed) ?? state.active;
  if (state.rite_react) return 1 - state.rite_react.caster;
  if (state.guard) return state.guard.defender;
  if (state.extra) return state.extra.player;
  return state.active;
}

const eventLabels: Record<string, string> = { match_started: "Partida iniciada", round_started: "Nova rodada", card_drawn: "Carta comprada", stance_committed: "Postura escolhida", stances_revealed: "Posturas reveladas", card_played: "Carta jogada", damage_dealt: "Dano causado", damage_prevented: "Dano prevenido", healed: "Vitalidade restaurada", ward_gained: "Ward obtido", sigil_added: "Sigilo emitido", eclipse_shifted: "Eclipse deslocado", eclipse_triggered: "Eclipse total", decision_requested: "Decisão aberta", decision_resolved: "Decisão resolvida", pass: "Janela encerrada", match_ended: "Partida encerrada", ultimate: "Ultimate ativada" };
const eventTitle = (kind: string) => eventLabels[kind] ?? kind.replaceAll("_", " ");
const decisionLabel = (kind: string) => ({ discard_n: "Escolha o descarte", pick_top2: "Escolha o destino das cartas", direction: "Escolha a direção do Eclipse", formula_choice: "Escolha a fórmula", reorder_top: "Reordene o topo", scry_bottom: "Mantenha ou envie ao fundo" }[kind] ?? "Resolva o efeito");
const endReason = (reason?: string) => ({ concede: "A partida terminou por concessão.", timeout: "O tempo de ação se esgotou.", vitality: "Uma Vitalidade chegou ao fim." }[reason ?? ""] ?? "A engine confirmou o resultado final.");
