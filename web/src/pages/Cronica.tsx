import { useMemo } from "react";
import { Link, useParams } from "react-router-dom";
import { UiIcon } from "../components/UiIcon";
import { useChampions, useMatchReplay } from "../queries";
import { useSessionStore } from "../store";
import type { BattleEvent, MatchReplayData } from "../types";
import "../arena.css";

// Crônica pós-partida: a engine é determinística e o log de eventos É a
// partida — aqui ele vira história legível: curva de Vitalidade, confrontos
// centrais e os momentos que decidiram o duelo.

interface Moment {
  seq: number;
  round: number;
  icon: "ultimate" | "sigil" | "reaction" | "versus" | "warning";
  text: string;
}

interface Chronicle {
  vitality: [Array<{ seq: number; value: number }>, Array<{ seq: number; value: number }>];
  moments: Moment[];
  rounds: number;
  lastSeq: number;
  confrontations: number;
  shattered: number;
  cardsPlayed: number;
  damage: number;
}

function buildChronicle(replay: MatchReplayData, startVitality: [number, number], names: [string, string]): Chronicle {
  const vitality: Chronicle["vitality"] = [
    [{ seq: 0, value: startVitality[0] }],
    [{ seq: 0, value: startVitality[1] }],
  ];
  const moments: Moment[] = [];
  let rounds = 1;
  let lastSeq = 0;
  let confrontations = 0;
  let shattered = 0;
  let cardsPlayed = 0;
  let damage = 0;
  const seat = (event: BattleEvent) => (event.p === 0 || event.p === 1 ? names[event.p] : "A mesa");
  for (const event of replay.events) {
    lastSeq = Math.max(lastSeq, event.seq);
    if (event.round > rounds) rounds = event.round;
    switch (event.kind) {
      case "damage_dealt":
      case "healed":
      case "fatigue":
      case "vitality_spent":
        if (event.p === 0 || event.p === 1) vitality[event.p].push({ seq: event.seq, value: event.to });
        if (event.kind === "damage_dealt") damage += event.n;
        break;
      case "card_played":
        cardsPlayed++;
        break;
      case "confrontation_resolved": {
        confrontations++;
        const winner = event.s === "guard" ? names[1 - event.p] : names[event.p];
        moments.push({ seq: event.seq, round: event.round, icon: "versus", text: `${winner} venceu o confronto central${event.n > 0 ? ` e causou ${event.n} de dano` : " sem deixar dano passar"}.` });
        break;
      }
      case "card_shattered":
        shattered++;
        break;
      case "card_burned":
        moments.push({ seq: event.seq, round: event.round, icon: "warning", text: `${seat(event)} queimou uma compra por estar com a mão cheia.` });
        break;
      case "status_applied":
        if (event.s?.includes("Pressão")) moments.push({ seq: event.seq, round: event.round, icon: "warning", text: "A Pressão de Nythara começou a consumir os dois lados." });
        break;
      case "match_ended":
        moments.push({ seq: event.seq, round: event.round, icon: "versus", text: "O Véu se fechou." });
        break;
      default:
        break;
    }
  }
  return { vitality, moments, rounds, lastSeq, confrontations, shattered, cardsPlayed, damage };
}

function polyline(points: Array<{ seq: number; value: number }>, lastSeq: number,
  min: number, max: number, width: number, height: number): string {
  const spanX = Math.max(1, lastSeq);
  const spanY = Math.max(1, max - min);
  const path = points.map(({ seq, value }) =>
    `${((seq / spanX) * width).toFixed(1)},${(height - ((value - min) / spanY) * height).toFixed(1)}`);
  // Estende a última leitura até a borda para a curva não "morrer" no meio.
  if (points.length) {
    const last = points[points.length - 1];
    path.push(`${width},${(height - ((last.value - min) / spanY) * height).toFixed(1)}`);
  }
  return path.join(" ");
}

export function CronicaPage() {
  const { matchId = "" } = useParams();
  const { data: replay, error, isLoading } = useMatchReplay(matchId);
  const { data: champions } = useChampions();
  const me = useSessionStore((state) => state.principal);

  const chronicle = useMemo(() => {
    if (!replay) return null;
    const names: [string, string] = [
      replay.players[0].display_name || "Duelista 0",
      replay.players[1].display_name || "Duelista 1",
    ];
    // Mesma base do replay: o primeiro ponto da curva precisa ser o valor com
    // que a partida realmente começou.
    const base = replay.starting_vitality ?? 0;
    const vitality: [number, number] = [
      base || champions?.champions.find((champion) => champion.id === replay.players[0].champion_id)?.vitality || 30,
      base || champions?.champions.find((champion) => champion.id === replay.players[1].champion_id)?.vitality || 30,
    ];
    return buildChronicle(replay, vitality, names);
  }, [champions, replay]);

  if (isLoading) return <div className="page arena-page"><p className="arena-empty">Abrindo a crônica…</p></div>;
  if (error || !replay || !chronicle) {
    return <div className="page arena-page"><p className="arena-empty">
      {error instanceof Error ? error.message : "Crônica indisponível."} <Link to="/arena">Voltar à Arena</Link></p></div>;
  }

  const championName = (id: string) => champions?.champions.find((champion) => champion.id === id)?.name ?? id;
  const mySlot = replay.players.findIndex((player) => player.user_id === me?.user_id);
  const won = replay.winner !== undefined && replay.winner === mySlot;
  const maxVit = Math.max(34, ...chronicle.vitality.flat().map((point) => point.value));

  return <div className="page arena-page cronica-page">
    <header className="arena-head">
      <p className="eyebrow">CRÔNICA DA PARTIDA</p>
      <h1>{championName(replay.players[0].champion_id)} <span className="cronica-vs">×</span> {championName(replay.players[1].champion_id)}</h1>
      <p className="cronica-sub">
        {replay.players[0].display_name} contra {replay.players[1].display_name} · {replay.mode === "practice" ? "treino" : "ranqueada"} · {chronicle.rounds} rodadas ·
        {mySlot >= 0 ? (won ? " vitória sua" : " derrota sua") : ` venceu ${replay.winner !== undefined ? replay.players[replay.winner]?.display_name : "ninguém"}`}
      </p>
    </header>

    <section className="panel cronica-chart" aria-label="Curva de Vitalidade">
      <header><p className="eyebrow">VITALIDADE</p>
        <div className="cronica-legend"><i className="is-p0" /> {replay.players[0].display_name} <i className="is-p1" /> {replay.players[1].display_name}</div>
      </header>
      <svg viewBox="0 0 600 160" preserveAspectRatio="none" role="img" aria-label="Vitalidade dos dois lados ao longo da partida">
        <polyline className="cronica-line is-p0" points={polyline(chronicle.vitality[0], chronicle.lastSeq, 0, maxVit, 600, 160)} />
        <polyline className="cronica-line is-p1" points={polyline(chronicle.vitality[1], chronicle.lastSeq, 0, maxVit, 600, 160)} />
      </svg>
    </section>

    <section className="panel cronica-summary" aria-label="Resumo dos confrontos">
      <header><p className="eyebrow">MODO CONFRONTO</p><h2>Resumo da mesa</h2></header>
      <div className="result-stats">
        <article><small>CONFRONTOS</small><strong>{chronicle.confrontations}</strong></article>
        <article><small>CARTAS ESTILHAÇADAS</small><strong>{chronicle.shattered}</strong></article>
        <article><small>CARTAS JOGADAS</small><strong>{chronicle.cardsPlayed}</strong></article>
        <article><small>DANO TOTAL</small><strong>{chronicle.damage}</strong></article>
      </div>
    </section>

    <section className="panel cronica-moments" aria-label="Momentos decisivos">
      <header><p className="eyebrow">MOMENTOS</p><h2>O que decidiu o duelo</h2></header>
      {chronicle.moments.length ? <ul>
        {chronicle.moments.map((moment) => <li key={moment.seq}>
          <span className="cronica-moment__icon"><UiIcon name={moment.icon} /></span>
          <div><strong>Rodada {moment.round}</strong><small>{moment.text}</small></div>
        </li>)}
      </ul> : <p className="arena-empty">Um duelo direto, sem reviravoltas registradas.</p>}
      <footer><small>{replay.events.length} eventos authoritative · fim: {endReasonLabel(replay.end_reason)}</small>
        <nav aria-label="Navegação da partida"><Link className="secondary-button" to={`/replay/${replay.match_id}`}><UiIcon name="versus" /> Rever duelo</Link><Link className="ghost-button" to="/arena">Voltar à Arena</Link></nav></footer>
    </section>
  </div>;
}

const endReasonLabel = (reason?: string) =>
  ({ concessao: "concessão", concede: "concessão", timeout: "tempo esgotado", vitality: "Vitalidade zerada", vitalidade: "Vitalidade zerada", duplo_nocaute: "nocaute duplo" }[reason ?? ""] ?? reason ?? "—");
