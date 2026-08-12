import { useMemo } from "react";
import { Link } from "react-router-dom";
import { UiIcon } from "../components/UiIcon";
import { useChampions, useLeaderboard, useMatchHistory, useProgress } from "../queries";
import { usePreferencesStore, useSessionStore } from "../store";
import { formatDate } from "../i18n";
import "../arena.css";

// Arena — a casa do duelista: patente da temporada, escada, histórico com
// crônicas e códigos de deck para compartilhar listas.

export function ArenaPage() {
  const locale = usePreferencesStore((state) => state.locale);
  const { data: progress } = useProgress();
  const { data: board } = useLeaderboard();
  const { data: history } = useMatchHistory();
  const { data: champions } = useChampions();
  const me = useSessionStore((state) => state.principal);
  const championName = (id: string) => champions?.champions.find((champion) => champion.id === id)?.name ?? id;
  const ranked = progress?.ranked;
  const tier = ranked?.tier;
  const recentMatches = history?.matches ?? [];
  const rankedRate = ranked?.games ? Math.round((ranked.wins / ranked.games) * 100) : 0;
  const tierProgress = useMemo(() => {
    if (!ranked || !tier?.next_at) return 100;
    const span = tier.next_at - tier.min_rating;
    return Math.min(100, Math.max(4, Math.round(((ranked.rating - tier.min_rating) / Math.max(1, span)) * 100)));
  }, [ranked, tier]);

  return <div className="page arena-page">
    <header className="arena-head"><div><p className="eyebrow">CÍRCULO COMPETITIVO</p><h1>Arena</h1><p>Suba na temporada, reveja seus confrontos e volte à mesa.</p></div><nav className="arena-head__actions" aria-label="Ações da Arena"><Link className="primary-button" to="/queue"><UiIcon name="duel" /> Buscar rival</Link><Link className="secondary-button" to="/decks"><UiIcon name="deck" /> Ajustar baralho</Link></nav></header>

    <section className="arena-grid">
      <article className="panel arena-tier" aria-label="Sua patente">
        <span className="arena-tier__mark"><UiIcon name="rank" /></span>
        <p className="eyebrow">SUA PATENTE</p>
        <h2>{tier?.name ?? "Errante do Crepúsculo"}</h2>
        <strong className="arena-tier__rating">{ranked?.rating ?? 1000}</strong>
        <div className="ritual-progress arena-tier__bar"><span style={{ width: `${tierProgress}%` }} /></div>
        <small>{tier?.next_at
          ? `${tier.next_at - (ranked?.rating ?? 0)} de rating até a próxima patente`
          : "Você alcançou a patente mais alta da temporada"}
          {ranked && ranked.games > 0 ? ` · ${ranked.wins}V/${ranked.games}J` : " · dispute ranqueadas para subir"}</small>
      </article>

      <article className="panel arena-board" aria-label="Escada da temporada">
        <header><p className="eyebrow">ESCADA DA TEMPORADA</p><h2>Vozes do topo</h2></header>
        {board?.entries?.length ? <ol className="arena-board__list">
          {board.entries.map((entry) => <li key={entry.user_id} className={entry.user_id === me?.user_id ? "is-me" : ""}>
            <span className="arena-board__pos">{entry.position}</span>
            <div><strong>{entry.display_name || "Duelista velado"}</strong><small>{entry.tier}</small></div>
            <b>{entry.rating}</b>
          </li>)}
        </ol> : <p className="arena-empty">A escada abre com o primeiro duelo ranqueado da temporada.</p>}
      </article>
    </section>

    <section className="arena-pulse" aria-label="Desempenho competitivo">
      <article><small>RANQUEADAS</small><strong>{ranked?.games ?? 0}</strong><span>partidas na temporada</span></article>
      <article><small>VITÓRIAS</small><strong>{ranked?.wins ?? 0}</strong><span>confirmadas no ranking</span></article>
      <article><small>APROVEITAMENTO</small><strong>{rankedRate}%</strong><span>somente partidas ranqueadas</span></article>
      <article><small>MEMÓRIA</small><strong>{recentMatches.length}</strong><span>duelos recentes disponíveis</span></article>
    </section>

    <section className="panel arena-history" aria-label="Histórico de partidas">
      <header><p className="eyebrow">SUAS PARTIDAS</p><h2>Histórico</h2></header>
      {recentMatches.length ? <ul className="arena-history__list">
        {recentMatches.map((match) => <li key={match.match_id}>
          <span className={`arena-history__result ${match.won ? "is-win" : match.draw ? "" : "is-loss"}`}>
            {match.won ? "VITÓRIA" : match.draw ? "EMPATE" : "DERROTA"}</span>
          <div>
            <strong>{championName(match.my_champion)} × {championName(match.opponent_champion)}</strong>
            <small>{match.opponent || "Duelista velado"} · {match.mode === "practice" ? "treino" : "ranqueada"}
              {match.end_reason ? ` · ${endReasonLabel(match.end_reason)}` : ""}
              {match.finished_at ? ` · ${formatDate(match.finished_at, locale, { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" })}` : ""}</small>
          </div>
          <nav className="arena-history__actions" aria-label={`Abrir registros da partida contra ${match.opponent || "Duelista velado"}`}><Link className="secondary-button" to={`/replay/${match.match_id}`}><UiIcon name="versus" /> Rever duelo</Link><Link className="ghost-button" to={`/cronica/${match.match_id}`}><UiIcon name="history" /> Crônica</Link></nav>
        </li>)}
      </ul> : <p className="arena-empty">Nenhuma partida encerrada ainda. <Link to="/queue">Atravesse o Véu</Link>.</p>}
    </section>
  </div>;
}

const endReasonLabel = (reason: string) => ({
  concessao: "concessão", concede: "concessão", timeout: "tempo esgotado",
  vitality: "Vitalidade zerada", vitalidade: "Vitalidade zerada",
  assalto: "golpe final", rito: "Rito decisivo", guarda: "resposta decisiva",
  sangramento: "Sangramento", maldicao: "Maldição", pressao_de_nythara: "Pressão de Nythara",
  duplo_nocaute: "nocaute duplo",
}[reason] ?? reason.replaceAll("_", " "));
