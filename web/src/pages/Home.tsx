import { Link } from "react-router-dom";
import { useDecks, useMe, useSeason } from "../queries";
import { useSessionStore } from "../store";

export function Home() {
  const user = useSessionStore((state) => state.user);
  const activeMatch = useSessionStore((state) => state.activeMatchId);
  const { data: decks } = useDecks();
  const { data: season } = useSeason();
  useMe();
  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Bom dia" : hour < 18 ? "Boa tarde" : "Boa noite";

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
        <article><span className="stat-icon">✦</span><div><small>COLEÇÃO ALPHA</small><strong>Completa</strong></div><Link to="/collection">Explorar</Link></article>
        <article><span className="stat-icon">♜</span><div><small>RANKED</small><strong>Pré-temporada</strong></div><Link to="/profile">Detalhes</Link></article>
      </section>

      <div className="home-columns">
        <section className="panel"><header><div><p className="eyebrow">SEUS ARSENAIS</p><h2>Decks recentes</h2></div><Link to="/decks">Ver todos</Link></header>
          {decks?.decks.length ? <div className="deck-list">{decks.decks.slice(0, 3).map((deck) => <article key={deck.id}><span className="deck-sigil">◈</span><div><strong>{deck.name}</strong><small>{deck.cards.reduce((sum, card) => sum + card.quantity, 0)} cartas · {deck.ruleset_version}</small></div><Link to={`/decks?edit=${deck.id}`}>Editar</Link></article>)}</div> : <div className="empty-state"><span>▤</span><h3>Seu primeiro deck começa aqui</h3><p>Escolha um Campeão e monte 36 cartas em poucos segundos.</p><Link className="secondary-button" to="/decks">Montar deck</Link></div>}
        </section>
        <section className="panel ritual-panel"><p className="eyebrow">RITUAL DIÁRIO</p><h2>Afie uma decisão</h2><p>Revise como Posturas e Ressonância alteram o ritmo antes de entrar na fila.</p><div className="ritual-progress"><span style={{ width: "33%" }} /></div><small>1 de 3 fundamentos revisitados</small><Link className="secondary-button" to="/tutorial">Continuar tutorial</Link></section>
      </div>
    </div>
  );
}
