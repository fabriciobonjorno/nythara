import { useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { api, mutationHeaders } from "../api";
import { CardTile } from "../components/CardTile";
import { useCards, useChampions, useCollection, useDecks } from "../queries";
import type { CardDefinition, Deck } from "../types";

const DECK_SIZE = 36;

export function DeckBuilderPage() {
  const { data: cardsData } = useCards();
  const { data: championsData } = useChampions();
  const { data: collection } = useCollection();
  const { data: decksData } = useDecks();
  const queryClient = useQueryClient();
  const [params, setParams] = useSearchParams();
  const [editing, setEditing] = useState<Deck | null>(null);
  const [name, setName] = useState("Novo Círculo");
  const [championId, setChampionId] = useState("");
  const [allyFaction, setAllyFaction] = useState("");
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [search, setSearch] = useState("");
  const [type, setType] = useState("Todos");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const loadedRef = useRef("");

  const cards = cardsData?.cards ?? [];
  const champions = championsData?.champions ?? [];
  const selectedChampion = champions.find((champion) => champion.id === championId);
  const factions = [...new Set(cards.map((card) => card.faction))].filter((faction) => faction !== "Errantes" && faction !== selectedChampion?.faction);
  const owned = new Map(collection?.cards.map((item) => [item.card_id, item.quantity]));
  const total = Object.values(counts).reduce((sum, quantity) => sum + quantity, 0);
  const alliedTotal = cards.reduce((sum, card) => sum + (card.faction === allyFaction ? counts[card.id] ?? 0 : 0), 0);
  const eligible = useMemo(() => cards.filter((card) => (!selectedChampion || card.faction === selectedChampion.faction || card.faction === "Errantes" || card.faction === allyFaction) &&
    (type === "Todos" || card.type === type) && `${card.name} ${card.rules_text}`.toLocaleLowerCase("pt-BR").includes(search.toLocaleLowerCase("pt-BR"))), [allyFaction, cards, search, selectedChampion, type]);

  useEffect(() => {
    const id = params.get("edit");
    if (!id || loadedRef.current === id || !decksData) return;
    const deck = decksData.decks.find((item) => item.id === id);
    if (!deck) return;
    loadedRef.current = id;
    setEditing(deck);
    setName(deck.name);
    setChampionId(deck.champion_id);
    setCounts(Object.fromEntries(deck.cards.map((card) => [card.card_id, card.quantity])));
    const champion = champions.find((item) => item.id === deck.champion_id);
    const allied = deck.cards.map((item) => cards.find((card) => card.id === item.card_id)?.faction).find((faction) => faction && faction !== "Errantes" && faction !== champion?.faction);
    setAllyFaction(allied ?? "");
  }, [cards, champions, decksData, params]);

  const chooseChampion = (id: string) => {
    if (id === championId) return;
    setChampionId(id);
    setAllyFaction("");
    setCounts({});
    setMessage("");
  };

  const maxCopies = (card: CardDefinition) => Math.min(owned.get(card.id) ?? 0, card.rarity === "Lendária" ? 1 : 2);
  const add = (card: CardDefinition) => {
    if (!selectedChampion) { setMessage("Escolha um Campeão primeiro."); return; }
    if (total >= DECK_SIZE || (counts[card.id] ?? 0) >= maxCopies(card)) return;
    if (card.faction === allyFaction && alliedTotal >= 12) { setMessage("A facção aliada atingiu o limite de 12 cartas."); return; }
    setCounts((current) => ({ ...current, [card.id]: (current[card.id] ?? 0) + 1 }));
  };
  const remove = (id: string) => setCounts((current) => {
    const next = { ...current };
    if ((next[id] ?? 0) <= 1) delete next[id]; else next[id] -= 1;
    return next;
  });

  const autoBuild = () => {
    if (!selectedChampion) { setMessage("Escolha um Campeão para montar automaticamente."); return; }
    const pool = cards.filter((card) => card.faction === selectedChampion.faction || card.faction === "Errantes")
      .sort((a, b) => a.cost - b.cost || a.id.localeCompare(b.id));
    const next: Record<string, number> = {};
    let filled = 0;
    for (const card of pool) {
      const quantity = Math.min(maxCopies(card), DECK_SIZE - filled);
      if (quantity > 0) next[card.id] = quantity;
      filled += quantity;
      if (filled === DECK_SIZE) break;
    }
    setCounts(next);
    setMessage(filled === DECK_SIZE ? "Deck de 36 cartas montado. Você pode refiná-lo antes de salvar." : "Não há cartas suficientes para este Campeão.");
  };

  const save = async () => {
    if (!selectedChampion || total !== DECK_SIZE || name.trim().length < 2) { setMessage("Defina nome, Campeão e exatamente 36 cartas."); return; }
    setBusy(true);
    setMessage("");
    const body = { name: name.trim(), champion_id: championId, cards: Object.entries(counts).sort().map(([card_id, quantity]) => ({ card_id, quantity })), ...(editing ? { version: editing.version } : {}) };
    try {
      const saved = await api<Deck>(editing ? `/v1/decks/${editing.id}` : "/v1/decks", { method: editing ? "PUT" : "POST", headers: mutationHeaders(), body: JSON.stringify(body) });
      setEditing(saved);
      loadedRef.current = saved.id;
      setParams({ edit: saved.id });
      await queryClient.invalidateQueries({ queryKey: ["decks"] });
      setMessage("Deck validado pela engine e salvo.");
    } catch (caught) { setMessage(caught instanceof Error ? caught.message : "Não foi possível salvar."); }
    finally { setBusy(false); }
  };

  const newDeck = () => { loadedRef.current = ""; setEditing(null); setName("Novo Círculo"); setChampionId(""); setAllyFaction(""); setCounts({}); setParams({}); setMessage(""); };
  const deleteDeck = async () => {
    if (!editing || !window.confirm(`Excluir “${editing.name}”?`)) return;
    setBusy(true);
    try {
      await api<void>(`/v1/decks/${editing.id}?version=${editing.version}`, { method: "DELETE", headers: mutationHeaders() });
      await queryClient.invalidateQueries({ queryKey: ["decks"] });
      newDeck();
    } catch (caught) { setMessage(caught instanceof Error ? caught.message : "Não foi possível excluir."); }
    finally { setBusy(false); }
  };

  return <div className="deck-builder-page">
    <header className="deck-builder-top"><div><p className="eyebrow">OFICINA DE ESTRATÉGIAS</p><h1>Construtor de deck</h1></div><div className="deck-picker"><label>Deck<select value={editing?.id ?? ""} onChange={(event) => setParams(event.target.value ? { edit: event.target.value } : {})}><option value="">Novo deck</option>{decksData?.decks.map((deck) => <option value={deck.id} key={deck.id}>{deck.name}</option>)}</select></label><button className="ghost-button" type="button" onClick={newDeck}>Novo</button></div></header>
    <div className="deck-builder-layout">
      <section className="deck-workbench">
        <div className="builder-section"><div className="builder-heading"><div><span>1</span><h2>Escolha o Campeão</h2></div><small>A facção define o arsenal disponível.</small></div><div className="champion-picker">{champions.map((champion) => <button type="button" className={champion.id === championId ? "is-selected" : ""} onClick={() => chooseChampion(champion.id)} key={champion.id}><span>♙</span><strong>{champion.name.split(",")[0]}</strong><small>{champion.faction} · ♥ {champion.vitality}</small></button>)}</div></div>
        <div className="builder-section"><div className="builder-heading"><div><span>2</span><h2>Selecione as cartas</h2></div><button className="secondary-button" type="button" onClick={autoBuild}>Montar automaticamente</button></div><div className="filter-bar compact builder-filters"><input type="search" placeholder="Buscar cartas…" value={search} onChange={(event) => setSearch(event.target.value)} /><select value={type} onChange={(event) => setType(event.target.value)}><option>Todos</option>{["Assalto", "Guarda", "Rito", "Relíquia", "Manifestação"].map((item) => <option key={item}>{item}</option>)}</select><select aria-label="Facção aliada" value={allyFaction} onChange={(event) => { const next = event.target.value; setAllyFaction(next); setCounts((current) => Object.fromEntries(Object.entries(current).filter(([id]) => { const faction = cards.find((card) => card.id === id)?.faction; return !faction || faction === selectedChampion?.faction || faction === "Errantes" || faction === next; }))); }}><option value="">Sem facção aliada</option>{factions.map((faction) => <option key={faction}>{faction}</option>)}</select></div>
          {!selectedChampion ? <div className="empty-state"><span>♙</span><h3>Escolha quem lidera</h3><p>As cartas compatíveis aparecerão aqui.</p></div> : <div className="builder-card-grid">{eligible.map((card) => <CardTile compact key={card.id} card={card} quantity={maxCopies(card)} selected={counts[card.id] ?? 0} disabled={total >= DECK_SIZE || (counts[card.id] ?? 0) >= maxCopies(card)} onSelect={() => add(card)} />)}</div>}
        </div>
      </section>
      <aside className="deck-summary"><div className="deck-count"><span style={{ "--progress": `${Math.min(100, total / DECK_SIZE * 100)}%` } as React.CSSProperties}><b>{total}</b><small>/ {DECK_SIZE}</small></span><div><strong>{total === DECK_SIZE ? "Deck completo" : `${DECK_SIZE - total} restantes`}</strong><small>{selectedChampion?.faction ?? "Sem facção"}</small></div></div><label>Nome do deck<input value={name} maxLength={60} onChange={(event) => setName(event.target.value)} /></label><div className="selected-cards"><header><strong>Lista</strong><small>{Object.keys(counts).length} únicas</small></header>{Object.entries(counts).sort().map(([id, quantity]) => { const card = cards.find((item) => item.id === id); return card ? <div key={id}><span className="mini-cost">{card.cost}</span><span><strong>{card.name}</strong><small>{card.type}</small></span><b>×{quantity}</b><button type="button" onClick={() => remove(id)} aria-label={`Remover ${card.name}`}>−</button></div> : null; })}</div>{message && <p className="builder-message" role="status">{message}</p>}<button className="primary-button" type="button" disabled={busy || total !== DECK_SIZE} onClick={save}>{busy ? "Salvando…" : editing ? "Salvar alterações" : "Salvar deck"}</button>{editing && <button className="danger-button" type="button" onClick={deleteDeck}>Excluir deck</button>}</aside>
    </div>
  </div>;
}
