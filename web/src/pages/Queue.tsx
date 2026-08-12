import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api";
import { NytharaMark } from "../components/NytharaMark";
import { UiIcon } from "../components/UiIcon";
import { useActiveRulesetVersion, useDecks, useProgress } from "../queries";
import { useSessionStore } from "../store";
import type { QueueResult } from "../types";

export function QueuePage() {
  const { data } = useDecks();
	const { data: progress } = useProgress();
	const accountLevel = progress?.account.level ?? 1;
  const rulesetVersion = useActiveRulesetVersion();
  const decks = data?.decks.filter((deck) => deck.ruleset_version === rulesetVersion && deck.active) ?? [];
  const navigate = useNavigate();
  const setActiveMatch = useSessionStore((store) => store.setActiveMatch);
  const [deckId, setDeckId] = useState("");
  const [queue, setQueue] = useState<QueueResult>({ status: "idle" });
  const [seconds, setSeconds] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => { if (!deckId && decks[0]) setDeckId(decks[0].id); }, [deckId, decks]);
  useEffect(() => {
    let cancelled = false;
    void api<QueueResult>("/v1/matchmaking").then((result) => { if (!cancelled) setQueue(result); }).catch(() => undefined);
    return () => { cancelled = true; };
  }, []);
  useEffect(() => {
    if (queue.status !== "queued") return;
    const started = Date.now() - seconds * 1000;
    const timer = window.setInterval(async () => {
      setSeconds(Math.floor((Date.now() - started) / 1000));
      try {
        const result = await api<QueueResult>("/v1/matchmaking");
        setQueue(result);
        if (result.status === "matched" && result.match_id) {
          setActiveMatch(result.match_id);
          navigate(`/battle/${result.match_id}`);
        }
      } catch (caught) { setError(caught instanceof Error ? caught.message : "A fila não respondeu."); }
    }, 1200);
    return () => window.clearInterval(timer);
  }, [navigate, queue.status, seconds, setActiveMatch]);

  const enter = async () => {
    if (!deckId) return;
    setBusy(true); setError("");
    try {
      const result = await api<QueueResult>("/v1/matchmaking", { method: "POST", body: JSON.stringify({ deck_id: deckId }) });
      setQueue(result);
      if (result.status === "matched" && result.match_id) {
        setActiveMatch(result.match_id);
        navigate(`/battle/${result.match_id}`);
      }
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Não foi possível entrar na fila."); }
    finally { setBusy(false); }
  };
  const cancel = async () => {
    setBusy(true);
    try { await api<void>("/v1/matchmaking", { method: "DELETE" }); setQueue({ status: "idle" }); setSeconds(0); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "Não foi possível sair da fila."); }
    finally { setBusy(false); }
  };
  const practice = async () => {
    if (!deckId) return;
    setBusy(true); setError("");
    try {
      const result = await api<QueueResult>("/v1/practice", { method: "POST", body: JSON.stringify({ deck_id: deckId }) });
      if (result.status === "matched" && result.match_id) {
        setActiveMatch(result.match_id);
        navigate(`/battle/${result.match_id}`);
      }
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Não foi possível iniciar o treino."); }
    finally { setBusy(false); }
  };
  // Fila pequena: oferece treino cedo sem cancelar automaticamente a busca.
  const practiceFromQueue = async () => {
    if (!deckId) return;
    setBusy(true); setError("");
    try {
      await api<void>("/v1/matchmaking", { method: "DELETE" });
      setQueue({ status: "idle" });
      setSeconds(0);
      const result = await api<QueueResult>("/v1/practice", { method: "POST", body: JSON.stringify({ deck_id: deckId }) });
      if (result.status === "matched" && result.match_id) {
        setActiveMatch(result.match_id);
        navigate(`/battle/${result.match_id}`);
      }
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Não foi possível migrar para o treino."); }
    finally { setBusy(false); }
  };

  const selected = decks.find((deck) => deck.id === deckId);
  return <div className="page queue-page"><div className="queue-stage">
    <div className={`queue-orbit ${queue.status === "queued" ? "is-searching" : ""}`} aria-hidden="true"><span className="orbit orbit-a" /><span className="orbit orbit-b" /><NytharaMark /></div>
	<p className="eyebrow">CÍRCULO DE CONFRONTO</p><h1>{queue.status === "queued" ? "Procurando um rival…" : "Entre no duelo"}</h1><p>{queue.status === "queued" ? `Buscando níveis ${Math.max(1, accountLevel - 5)} a ${Math.min(50, accountLevel + 5)} no mesmo ruleset.` : "Escolha um deck validado e atravesse o Véu."}</p>
    {queue.status === "queued" ? <div className="queue-search"><strong>{String(Math.floor(seconds / 60)).padStart(2, "0")}:{String(seconds % 60).padStart(2, "0")}</strong><small>TEMPO DE BUSCA</small><button className="ghost-button" type="button" onClick={cancel} disabled={busy}>Cancelar busca</button>{seconds >= 15 && <aside className="queue-suggest" role="status"><p><strong>A busca continua em segundo plano.</strong><br />Se preferir jogar agora, abra um treino com o mesmo baralho e as mesmas regras.</p><button className="primary-button" type="button" onClick={practiceFromQueue} disabled={busy || !deckId}>Jogar agora contra o bot</button></aside>}</div> : <div className="queue-controls"><label>Seu baralho ativo<select value={deckId} onChange={(event) => setDeckId(event.target.value)}><option value="">Selecione</option>{decks.map((deck) => <option key={deck.id} value={deck.id}>{deck.name}</option>)}</select></label>{selected && <article className="selected-deck"><span className="selected-deck__icon"><UiIcon name="deck" /></span><div><strong>{selected.name}</strong><small>{selected.cards.reduce((sum, card) => sum + card.quantity, 0)} cartas · Modo Confronto</small></div><b>Pronto</b></article>}<div className="queue-mode-actions"><button className="primary-button" type="button" onClick={enter} disabled={!deckId || busy}><small>DUELO PVP</small>{busy ? "Entrando…" : "Buscar oponente"}</button><button className="secondary-button" type="button" onClick={practice} disabled={!deckId || busy}><small>SEM ESPERA</small>Treino instantâneo</button></div>{!decks.length && <p>Você precisa <Link to="/decks">montar seu baralho de 30 cartas</Link> antes de jogar.</p>}</div>}
    {error && <p className="form-error" role="alert">{error}</p>}
    {queue.status === "idle" && <section className="queue-how"><p className="eyebrow">COMO DUELAR</p><ol><li><span>1</span><div><strong>Assalto</strong><small>Envie uma carta ao centro.</small></div></li><li><span>2</span><div><strong>Guarda</strong><small>O rival bloqueia ou aceita o dano.</small></div></li><li><span>3</span><div><strong>Rito</strong><small>Use um efeito e encerre o turno.</small></div></li></ol><Link to="/tutorial?step=4">Ver guia do duelo <UiIcon name="arrow-right" /></Link></section>}
	<div className="queue-note"><span><UiIcon name="balance" /></span><p><strong>Faixa equilibrada</strong><br />Você está no nível {accountLevel}. O servidor só forma partidas com diferença máxima de 5 níveis. Somente PvP contra jogadores concede XP.</p></div>
  </div></div>;
}
