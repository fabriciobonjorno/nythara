import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api";
import { useDecks } from "../queries";
import { useSessionStore } from "../store";
import type { QueueResult } from "../types";

export function QueuePage() {
  const { data } = useDecks();
  const navigate = useNavigate();
  const setActiveMatch = useSessionStore((store) => store.setActiveMatch);
  const [deckId, setDeckId] = useState("");
  const [queue, setQueue] = useState<QueueResult>({ status: "idle" });
  const [seconds, setSeconds] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => { if (!deckId && data?.decks[0]) setDeckId(data.decks[0].id); }, [data, deckId]);
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

  const selected = data?.decks.find((deck) => deck.id === deckId);
  return <div className="page queue-page"><div className="queue-stage">
    <div className={`queue-orbit ${queue.status === "queued" ? "is-searching" : ""}`} aria-hidden="true"><span className="orbit orbit-a" /><span className="orbit orbit-b" /><strong>◐</strong></div>
    <p className="eyebrow">CÍRCULO DE CONFRONTO</p><h1>{queue.status === "queued" ? "Procurando um rival…" : "Entre no duelo"}</h1><p>{queue.status === "queued" ? "O servidor está alinhando um oponente com o mesmo ruleset." : "Escolha um deck validado e atravesse o Véu."}</p>
    {queue.status === "queued" ? <div className="queue-search"><strong>{String(Math.floor(seconds / 60)).padStart(2, "0")}:{String(seconds % 60).padStart(2, "0")}</strong><small>TEMPO DE BUSCA</small><button className="ghost-button" type="button" onClick={cancel} disabled={busy}>Cancelar busca</button></div> : <div className="queue-controls"><label>Deck para o duelo<select value={deckId} onChange={(event) => setDeckId(event.target.value)}><option value="">Selecione</option>{data?.decks.map((deck) => <option key={deck.id} value={deck.id}>{deck.name}</option>)}</select></label>{selected && <article className="selected-deck"><span>◈</span><div><strong>{selected.name}</strong><small>{selected.cards.reduce((sum, card) => sum + card.quantity, 0)} cartas · {selected.ruleset_version}</small></div><b>Pronto</b></article>}<button className="primary-button" type="button" onClick={enter} disabled={!deckId || busy}>{busy ? "Entrando…" : "Buscar oponente"}</button>{!data?.decks.length && <p>Você precisa <a href="/decks">montar um deck</a> antes de jogar.</p>}</div>}
    {error && <p className="form-error" role="alert">{error}</p>}
    <div className="queue-note"><span>⚖</span><p><strong>Competição justa</strong><br />Todas as contas Alpha têm a mesma coleção competitiva.</p></div>
  </div></div>;
}
