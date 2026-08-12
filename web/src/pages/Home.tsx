import { useState } from "react";
import { Link } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { api } from "../api";
import { UiIcon } from "../components/UiIcon";
import { useActiveRulesetVersion, useChampions, useDecks, useMe, useProgress, useSeason } from "../queries";
import { useSessionStore } from "../store";
import type { QueueResult } from "../types";

export function Home() {
  const navigate = useNavigate();
  const user = useSessionStore((state) => state.user);
  const activeMatch = useSessionStore((state) => state.activeMatchId);
  const setActiveMatch = useSessionStore((state) => state.setActiveMatch);
  const [practiceBusy, setPracticeBusy] = useState(false);
  const [practiceError, setPracticeError] = useState("");
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
	const account = progress?.account;
  const rulesetVersion = useActiveRulesetVersion();
  const currentDeck = decks?.decks.find((deck) => deck.ruleset_version === rulesetVersion && deck.active)
    ?? decks?.decks.find((deck) => deck.ruleset_version === rulesetVersion);
  const startPractice = async () => {
    if (!currentDeck) { navigate("/decks"); return; }
    setPracticeBusy(true);
    setPracticeError("");
    try {
      const result = await api<QueueResult>("/v1/practice", { method: "POST", body: JSON.stringify({ deck_id: currentDeck.id }) });
      if (result.status !== "matched" || !result.match_id) throw new Error("O treino não abriu uma sala.");
      setActiveMatch(result.match_id);
      navigate(`/battle/${result.match_id}`);
    } catch (caught) {
      setPracticeError(caught instanceof Error ? caught.message : "Não foi possível iniciar o treino.");
      setPracticeBusy(false);
    }
  };

  return (
    <div className="page home-page">
      <section className="home-hero">
        <div><p className="eyebrow">{season?.name ?? "TEMPORADA ALPHA"}</p><h1>{greeting}, {user?.display_name?.split(" ")[0]}.</h1><p>Seu próximo confronto está a uma carta de distância.</p>
          <div className="hero-actions">{activeMatch ? <Link className="primary-button" to={`/battle/${activeMatch}`}>Retomar partida</Link> : <button className="primary-button" type="button" disabled={practiceBusy} onClick={startPractice}>{practiceBusy ? "Abrindo a mesa…" : "Treino instantâneo"}</button>}<Link className="ghost-button" to="/queue">Buscar rival</Link><Link className="ghost-button" to="/tutorial">Como jogar</Link></div>
          {practiceError && <p className="form-error" role="alert">{practiceError}</p>}
        </div>
        <div className="home-eclipse" aria-hidden="true"><img src="/assets/nythara-apocalypse-logo.webp" alt="" /><small>MODO CONFRONTO</small></div>
      </section>

      {decks && !currentDeck && <section className="getting-started" aria-labelledby="getting-started-title">
        <header><div><p className="eyebrow">PRIMEIRO DUELO</p><h2 id="getting-started-title">Comece em três passos</h2></div><Link to="/tutorial">Abrir guia completo</Link></header>
        <div className="getting-started__steps">
          <Link to="/decks"><span>1</span><div><strong>Monte 30 cartas</strong><small>Escolha também um Avatar apenas visual.</small></div><b>Montar <UiIcon name="arrow-right" /></b></Link>
          <Link to="/queue"><span>2</span><div><strong>Treine contra o adversário virtual</strong><small>Aprenda Assalto, Guarda e Rito sem esperar.</small></div><b>Treinar <UiIcon name="arrow-right" /></b></Link>
          <Link to="/arena"><span>3</span><div><strong>Entre no ranqueado</strong><small>O mesmo baralho enfrenta jogadores reais.</small></div><b>Arena <UiIcon name="arrow-right" /></b></Link>
        </div>
      </section>}

      <section className="stat-grid" aria-label="Resumo da conta">
		<article><span className="stat-icon"><UiIcon name="mastery" /></span><div><small>NÍVEL DA CONTA</small><strong>{account ? `Nível ${account.level}` : "—"}</strong>{account && <em>{account.level_xp_required ? `${account.level_xp}/${account.level_xp_required} XP` : "Nível máximo"}</em>}</div><Link to="/collection">Lendárias</Link></article>
		<article><span className="stat-icon"><UiIcon name="deck" /></span><div><small>BARALHO COMPETITIVO</small><strong>{decks ? (currentDeck ? "Pronto" : "Pendente") : "—"}</strong></div><Link to="/decks">Editar</Link></article>
        <article><span className="stat-icon"><UiIcon name="fragment" /></span><div><small>FRAGMENTOS DO VÉU</small><strong>{progress ? progress.fragments : "—"}</strong></div><Link to="/collection">Coleção</Link></article>
        <article><span className="stat-icon"><UiIcon name="rank" /></span><div><small>RANQUEADAS</small><strong>{progress?.ranked && progress.ranked.games > 0 ? `${progress.ranked.rating} · #${progress.ranked.position || "—"}` : "Estreie no ranking"}</strong></div><Link to="/queue">Duelar</Link></article>
      </section>

      <div className="home-columns">
        <section className="panel"><header><div><p className="eyebrow">SEU ARSENAL</p><h2>Baralho competitivo</h2></div><Link to="/decks">Abrir construtor</Link></header>
          {currentDeck ? <div className="deck-list"><article key={currentDeck.id}><span className="deck-sigil"><UiIcon name="deck" /></span><div><strong>{currentDeck.name}</strong><small>{currentDeck.cards.reduce((sum, card) => sum + card.quantity, 0)} cartas · Modo Confronto</small></div><Link to={`/decks?edit=${currentDeck.id}`}>Editar</Link></article></div> : <div className="empty-state"><UiIcon className="empty-state__icon" name="deck" /><h3>Seu baralho começa aqui</h3><p>Escolha um Avatar visual e monte 30 cartas em poucos segundos.</p><Link className="secondary-button" to="/decks">Montar baralho</Link></div>}
          {topMastery && <p className="mastery-note"><UiIcon name="mastery" /> Maestria de Avatar: <strong>{championName(topMastery.champion_id)}</strong> nível {topMastery.level} · {topMastery.wins}V em {topMastery.games} jogos</p>}
        </section>
        <section className="panel ritual-panel"><p className="eyebrow">RITUAIS DIÁRIOS</p><h2>{ritualsDone === progress?.rituals.length && progress?.rituals.length ? "Véu saciado por hoje" : "O Véu exige tributo"}</h2>
          {progress?.rituals.length ? <ul className="ritual-list">{progress.rituals.map((ritual) => {
            const done = Boolean(ritual.completed_at);
            const percent = Math.min(100, Math.round((ritual.progress / Math.max(1, ritual.target)) * 100));
            return <li key={ritual.ritual_id} className={done ? "is-done" : ""}>
              <div className="ritual-head"><strong>{ritual.title}</strong><span className="ritual-reward">{done ? <><UiIcon name="check" /><span className="sr-only">Concluído</span></> : `${ritual.progress}/${ritual.target}`} <i>·</i> {ritual.reward}<UiIcon name="fragment" /></span></div>
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
