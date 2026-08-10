import { Link } from "react-router-dom";
import { useChampions, useDecks, useMe, useProgress, useSeason } from "../queries";
import { useSessionStore } from "../store";

export function Home() {
  const user = useSessionStore((state) => state.user);
  const activeMatch = useSessionStore((state) => state.activeMatchId);
  const { data: decks } = useDecks();
  const { data: season } = useSeason();
  const { data: progress } = useProgress();
  const { data: champions } = useChampions();
  useMe();
  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Bom dia" : hour < 18 ? "Boa tarde" : "Boa noite";
  const championName = (id: string) => champions?.champions.find((champion) => champion.id === id)?.name ?? id;
  const ritualsDone = progress?.rituals.filter((ritual) => ritual.completed_at).length ?? 0;
  const topMastery = progress?.mastery?.[0];

  return (
    <div className="page home-page">
      <section className="home-hero">
        <div><p className="eyebrow">{season?.name ?? "TEMPORADA ALPHA"}</p><h1>{greeting}, {user?.display_name?.split(" ")[0]}.</h1><p>O Eclipse espera por sua próxima decisão.</p>
          <div className="hero-actions"><Link className="primary-button" to={activeMatch ? `/battle/${activeMatch}` : "/queue"}>{activeMatch ? "Retomar partida" : "Buscar duelo"}</Link><Link className="ghost-button" to="/tutorial">Rever fundamentos</Link></div>
        </div>
        <div className="home-eclipse" aria-hidden="true"><span>◐</span><small>EQUILÍBRIO INSTÁVEL</small></div>
      </section>

      <section className="stat-grid" aria-label="Resumo da conta">
        <article><span className="stat-icon">▤</span><div><small>DECKS PRONTOS</small><strong>{decks?.decks.length ?? "—"}</strong></div><Link to="/decks">Editar</Link></article>
        <article><span className="stat-icon">✦</span><div><small>FRAGMENTOS DO VÉU</small><strong>{progress ? progress.fragments : "—"}</strong></div><Link to="/collection">Coleção</Link></article>
        <article><span className="stat-icon">♜</span><div><small>RANKED</small><strong>{progress?.ranked && progress.ranked.games > 0 ? `${progress.ranked.rating} · #${progress.ranked.position || "—"}` : "Estreie no ranking"}</strong></div><Link to="/queue">Duelar</Link></article>
      </section>

      <div className="home-columns">
        <section className="panel"><header><div><p className="eyebrow">SEUS ARSENAIS</p><h2>Decks recentes</h2></div><Link to="/decks">Ver todos</Link></header>
          {decks?.decks.length ? <div className="deck-list">{decks.decks.slice(0, 3).map((deck) => <article key={deck.id}><span className="deck-sigil">◈</span><div><strong>{deck.name}</strong><small>{deck.cards.reduce((sum, card) => sum + card.quantity, 0)} cartas · {deck.ruleset_version}</small></div><Link to={`/decks?edit=${deck.id}`}>Editar</Link></article>)}</div> : <div className="empty-state"><span>▤</span><h3>Seu primeiro deck começa aqui</h3><p>Escolha um Campeão e monte 36 cartas em poucos segundos.</p><Link className="secondary-button" to="/decks">Montar deck</Link></div>}
          {topMastery && <p className="mastery-note"><span aria-hidden="true">☩</span> Maestria: <strong>{championName(topMastery.champion_id)}</strong> nível {topMastery.level} · {topMastery.wins}V em {topMastery.games} jogos</p>}
        </section>
        <section className="panel ritual-panel"><p className="eyebrow">RITUAIS DIÁRIOS</p><h2>{ritualsDone === progress?.rituals.length && progress?.rituals.length ? "Véu saciado por hoje" : "O Véu exige tributo"}</h2>
          {progress?.rituals.length ? <ul className="ritual-list">{progress.rituals.map((ritual) => {
            const done = Boolean(ritual.completed_at);
            const percent = Math.min(100, Math.round((ritual.progress / Math.max(1, ritual.target)) * 100));
            return <li key={ritual.ritual_id} className={done ? "is-done" : ""}>
              <div className="ritual-head"><strong>{ritual.title}</strong><span>{done ? "✓" : `${ritual.progress}/${ritual.target}`} · {ritual.reward}✦</span></div>
              <small>{ritual.description}</small>
              <div className="ritual-progress"><span style={{ width: `${done ? 100 : percent}%` }} /></div>
            </li>;
          })}</ul> : <p>Entre em uma partida para revelar os rituais de hoje.</p>}
          <small>{ritualsDone} de {progress?.rituals.length ?? 3} rituais cumpridos · recompensas em Fragmentos do Véu</small>
          <Link className="secondary-button" to="/queue">Cumprir agora</Link></section>
      </div>
    </div>
  );
}
