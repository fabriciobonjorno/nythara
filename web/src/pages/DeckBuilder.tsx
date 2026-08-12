import { useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { api, mutationHeaders } from "../api";
import { CardTile, cardConfrontMetric } from "../components/CardTile";
import { ChampionEmblem } from "../components/ChampionEmblem";
import { DataLoadError } from "../components/DataLoadError";
import { UiIcon } from "../components/UiIcon";
import { analyzeDeck } from "../deckAdvisor";
import { useActiveRulesetVersion, useCards, useChampions, useCollection, useDecks, useRuleset } from "../queries";
import type { CardDefinition, Deck } from "../types";
import { formatNumber } from "../i18n";
import { usePreferencesStore } from "../store";

const DECK_SIZE = 30;
const TYPE_TARGETS: Record<"Assalto" | "Guarda" | "Rito", number> = { Assalto: 10, Guarda: 10, Rito: 10 };
const DEFAULT_TYPE_MINIMUMS: Record<"Assalto" | "Guarda" | "Rito", number> = { Assalto: 8, Guarda: 10, Rito: 4 };

function remainingLock(deck: Deck | undefined, now: number) {
  if (!deck?.locked_until || deck.system_provided) return 0;
  return Math.max(0, new Date(deck.locked_until).getTime() - now);
}

function formatLock(milliseconds: number) {
  const totalMinutes = Math.ceil(milliseconds / 60_000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return `${hours}h ${String(minutes).padStart(2, "0")}min`;
}

function balancedDeck(cards: CardDefinition[], maxCopies: (card: CardDefinition) => number) {
  const next: Record<string, number> = {};
  for (const cardType of Object.keys(TYPE_TARGETS) as Array<keyof typeof TYPE_TARGETS>) {
    const pool = cards.filter((card) => card.type === cardType).sort((left, right) => left.cost - right.cost || left.id.localeCompare(right.id));
    let total = 0;
    let cursor = 0;
    while (total < TYPE_TARGETS[cardType] && pool.length) {
      const card = pool[cursor % pool.length];
      if ((next[card.id] ?? 0) < maxCopies(card)) {
        next[card.id] = (next[card.id] ?? 0) + 1;
        total++;
      }
      cursor++;
      if (cursor > pool.length * 3) break;
    }
  }
  return next;
}

export function DeckBuilderPage() {
  const locale = usePreferencesStore((state) => state.locale);
  const [params] = useSearchParams();
  const cardsQuery = useCards();
  const championsQuery = useChampions();
  const collectionQuery = useCollection();
  const decksQuery = useDecks();
  const rulesetQuery = useRuleset();
  const queryClient = useQueryClient();
  const cards = cardsQuery.data?.cards ?? [];
  const avatars = championsQuery.data?.champions ?? [];
  const collection = collectionQuery.data;
  const rulesetVersion = useActiveRulesetVersion();
  const typeMinimums = {
    Assalto: rulesetQuery.data?.min_assaults ?? DEFAULT_TYPE_MINIMUMS.Assalto,
    Guarda: rulesetQuery.data?.min_guards ?? DEFAULT_TYPE_MINIMUMS.Guarda,
    Rito: rulesetQuery.data?.min_rites ?? DEFAULT_TYPE_MINIMUMS.Rito,
  };
  const activeDeck = decksQuery.data?.decks.find((deck) => deck.ruleset_version === rulesetVersion && deck.active)
    ?? decksQuery.data?.decks.find((deck) => deck.ruleset_version === rulesetVersion);
  const legalCards = useMemo(() => cards.filter((card) => card.confront?.legal), [cards]);
  const archivedCount = cards.length - legalCards.length;
  const owned = useMemo(() => new Map(collection?.cards.map((item) => [item.card_id, item.quantity]) ?? []), [collection]);

  const [name, setName] = useState("Meu Baralho");
  const [avatarId, setAvatarId] = useState("");
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [search, setSearch] = useState("");
  const [type, setType] = useState<"Todos" | "Assalto" | "Guarda" | "Rito">("Todos");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [now, setNow] = useState(Date.now());
  const [loadedId, setLoadedId] = useState("");
  const lockLeft = remainingLock(activeDeck, now);
  const locked = lockLeft > 0;
  const requestedAvatar = params.get("avatar");

  useEffect(() => {
    if (!activeDeck || loadedId === activeDeck.id) return;
    setLoadedId(activeDeck.id);
    setName(activeDeck.name);
    setAvatarId(activeDeck.champion_id);
    setCounts(Object.fromEntries(activeDeck.cards.map((card) => [card.card_id, card.quantity])));
  }, [activeDeck, loadedId]);

  useEffect(() => {
    if (!avatarId && avatars[0]) setAvatarId(avatars[0].id);
  }, [avatarId, avatars]);

  useEffect(() => {
    if (!requestedAvatar || locked || !avatars.some((avatar) => avatar.id === requestedAvatar)) return;
    if (avatarId !== requestedAvatar) setAvatarId(requestedAvatar);
  }, [avatarId, avatars, locked, requestedAvatar]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, []);

  const total = Object.values(counts).reduce((sum, quantity) => sum + quantity, 0);
  const maxCopies = (card: CardDefinition) => Math.min(owned.get(card.id) ?? 0, card.rarity === "Lendária" ? 1 : 2);
  const visible = legalCards.filter((card) => (type === "Todos" || card.type === type)
    && `${card.name} ${card.rules_text} ${card.faction}`.toLocaleLowerCase("pt-BR").includes(search.toLocaleLowerCase("pt-BR")));
  const typeCounts = (Object.keys(TYPE_TARGETS) as Array<keyof typeof TYPE_TARGETS>).map((cardType) => ({
    type: cardType,
    count: legalCards.reduce((sum, card) => sum + (card.type === cardType ? counts[card.id] ?? 0 : 0), 0),
  }));
  const compositionValid = typeCounts.every((item) => item.count >= typeMinimums[item.type]);
  const analysis = useMemo(() => analyzeDeck(legalCards, counts), [counts, legalCards]);

  const add = (card: CardDefinition) => {
    if (locked || total >= DECK_SIZE || (counts[card.id] ?? 0) >= maxCopies(card)) return;
    setCounts((current) => ({ ...current, [card.id]: (current[card.id] ?? 0) + 1 }));
    setMessage("");
  };
  const remove = (id: string) => {
    if (locked) return;
    setCounts((current) => {
      const next = { ...current };
      if ((next[id] ?? 0) <= 1) delete next[id]; else next[id] -= 1;
      return next;
    });
  };
  const autoBuild = () => {
    if (locked) return;
    const next = balancedDeck(legalCards, maxCopies);
    setCounts(next);
    const amount = Object.values(next).reduce((sum, quantity) => sum + quantity, 0);
    setMessage(amount === DECK_SIZE ? "Baralho equilibrado: 10 Assaltos, 10 Guardas e 10 Ritos. Revise e salve." : "A coleção não tem cartas legais suficientes.");
  };
  const save = async () => {
    if (locked) { setMessage(`Este baralho poderá ser alterado em ${formatLock(lockLeft)}.`); return; }
    if (!avatarId || total !== DECK_SIZE || !name.trim()) { setMessage("Defina o nome e escolha exatamente 30 cartas."); return; }
    if (!compositionValid) { setMessage(`O baralho precisa de pelo menos ${typeMinimums.Assalto} Assaltos, ${typeMinimums.Guarda} Guardas e ${typeMinimums.Rito} Ritos.`); return; }
    setBusy(true); setMessage("");
    const body = {
      name: name.trim(), champion_id: avatarId,
      cards: Object.entries(counts).sort().map(([card_id, quantity]) => ({ card_id, quantity })),
      ...(activeDeck ? { version: activeDeck.version } : {}),
    };
    try {
      const saved = await api<Deck>(activeDeck ? `/v1/decks/${activeDeck.id}` : "/v1/decks", {
        method: activeDeck ? "PUT" : "POST", headers: mutationHeaders(), body: JSON.stringify(body),
      });
      await queryClient.invalidateQueries({ queryKey: ["decks"] });
      setLoadedId(saved.id);
      setMessage("Baralho salvo e protegido por 24 horas. Ele já está pronto para a Arena.");
      setNow(Date.now());
    } catch (caught) {
      setMessage(caught instanceof Error ? caught.message : "Não foi possível salvar o baralho.");
    } finally { setBusy(false); }
  };

  if (cardsQuery.isError || championsQuery.isError || collectionQuery.isError || decksQuery.isError) {
    return <div className="deck-builder-page"><header className="deck-builder-top"><div><p className="eyebrow">SEU ÚNICO BARALHO</p><h1>Montar baralho</h1></div></header><DataLoadError onRetry={() => { void cardsQuery.refetch(); void championsQuery.refetch(); void collectionQuery.refetch(); void decksQuery.refetch(); }} /></div>;
  }

  return <div className="deck-builder-page confront-builder">
    <header className="deck-builder-top"><div><p className="eyebrow">MODO CONFRONTO · 30 CARTAS</p><h1>Seu baralho de duelo</h1><p>Um baralho, três tipos. Seu Avatar acrescenta um poder próprio, sem alterar sua coleção.</p></div><button className="primary-button deck-save-top" type="button" disabled={busy || locked || total !== DECK_SIZE || !compositionValid} onClick={save}>{busy ? "Salvando…" : locked ? `Travado · ${formatLock(lockLeft)}` : "Salvar baralho"}</button></header>

    <section className={`deck-lock-banner ${locked ? "is-locked" : ""}`} role="status"><span><UiIcon name={locked ? "clock" : "info"} /></span><div><strong>{locked ? "Baralho competitivo protegido" : "Você pode editar agora"}</strong><small>{locked ? `Nova alteração liberada em ${formatLock(lockLeft)}. Você pode jogar normalmente durante a trava.` : "Depois de salvar, a composição fica fixa por 24 horas para dar peso às escolhas."}</small></div></section>

    <div className="deck-builder-layout">
      <section className="deck-workbench">
        <div className="builder-section avatar-cosmetic"><div className="builder-heading"><div><span>1</span><h2>Escolha o Avatar</h2></div><small>Poder próprio · mesma coleção.</small></div><div className="champion-picker">{avatars.map((avatar) => <button type="button" disabled={locked} className={avatar.id === avatarId ? "is-selected" : ""} onClick={() => setAvatarId(avatar.id)} key={avatar.id}><span className="champion-picker__emblem"><ChampionEmblem id={avatar.id} faction={avatar.faction} /></span><strong>{avatar.name.split(",")[0]}</strong><small>{avatar.confront_power ?? "Sem poder neste conjunto de regras."}</small></button>)}</div></div>

        <div className="builder-section"><div className="builder-heading"><div><span>2</span><h2>Escolha 30 cartas</h2></div><button className="secondary-button" type="button" disabled={locked} onClick={autoBuild}>Montar 10 / 10 / 10</button></div><p className="builder-pool-note"><strong>{legalCards.length} cartas jogáveis</strong> neste modo. {archivedCount} cartas do ruleset antigo ficam no arquivo com o motivo explícito e não podem entrar no duelo.</p><div className="filter-bar compact builder-filters"><input aria-label="Buscar cartas" type="search" placeholder="Buscar por nome, ação ou facção…" value={search} onChange={(event) => setSearch(event.target.value)} /><select aria-label="Filtrar por tipo" value={type} onChange={(event) => setType(event.target.value as typeof type)}><option value="Todos">Todos</option><option value="Assalto">Assalto</option><option value="Guarda">Guarda</option><option value="Rito">Rito</option></select></div><div className="builder-card-grid">{visible.map((card) => <CardTile compact key={card.id} card={card} quantity={maxCopies(card)} selected={counts[card.id] ?? 0} disabled={locked || total >= DECK_SIZE || (counts[card.id] ?? 0) >= maxCopies(card)} onSelect={() => add(card)} />)}</div></div>
      </section>

      <aside className="deck-summary confront-summary"><div className="deck-count"><span style={{ "--progress": `${Math.min(100, total / DECK_SIZE * 100)}%` } as React.CSSProperties}><b>{total}</b><small>/ {DECK_SIZE}</small></span><div><strong>{total === DECK_SIZE && compositionValid ? "Baralho completo" : `${Math.max(0, DECK_SIZE - total)} restantes`}</strong><small>Máx. 2 cópias · Lendária 1</small></div></div><div className="deck-type-meter">{typeCounts.map((item) => <div className={item.count >= typeMinimums[item.type] ? "is-valid" : "is-missing"} key={item.type}><span>{item.type} <small>mín. {typeMinimums[item.type]}</small></span><b>{item.count}</b><i><em style={{ width: `${Math.min(100, item.count / 10 * 100)}%` }} /></i></div>)}</div><label>Nome do baralho<input disabled={locked} value={name} maxLength={64} onChange={(event) => setName(event.target.value)} /></label><details className={`deck-advisor ${analysis.ready ? "is-ready" : ""}`} open><summary><span><UiIcon name="balance" /><strong>Leitura do baralho</strong></span><small>Consultiva · não prevê vitória</small></summary><div className="deck-advisor__metrics"><span><small>CUSTO MÉDIO</small><b>{formatNumber(analysis.averageCost, locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}</b></span><span><small>PODER MÉDIO</small><b>{analysis.averagePower === null ? "—" : formatNumber(analysis.averagePower, locale, { maximumFractionDigits: 1 })}</b></span><span><small>PREVENÇÃO</small><b>{analysis.averagePrevention === null ? "—" : formatNumber(analysis.averagePrevention, locale, { maximumFractionDigits: 1 })}</b></span><span><small>PAPÉIS DE RITO</small><b>{analysis.riteRoles.length || "—"}</b></span></div><div className="deck-advisor__signals">{analysis.signals.map((signal) => <article className={`is-${signal.tone}`} key={signal.title}><i aria-hidden="true" /><span><strong>{signal.title}</strong><small>{signal.copy}</small></span></article>)}</div><p>{analysis.lowCostCards} cartas leves · {analysis.highCostCards} de custo alto{analysis.totalPreventionCards ? ` · ${analysis.totalPreventionCards} com prevenção total` : ""}</p></details><div className="selected-cards"><header><strong>Lista escolhida</strong><small>{Object.keys(counts).length} únicas</small></header>{Object.entries(counts).sort().map(([id, quantity]) => { const card = cards.find((item) => item.id === id); return card ? <div key={id}><span className="mini-cost">{card.cost}</span><span><strong>{card.name}</strong><small>{card.type} · {cardConfrontMetric(card)}</small></span><b>×{quantity}</b><button type="button" disabled={locked} onClick={() => remove(id)} aria-label={`Remover ${card.name}`}><UiIcon name="minus" /></button></div> : null; })}</div><div className="deck-summary-actions">{message && <p className="builder-message" role="status">{message}</p>}<button className="primary-button" type="button" disabled={busy || locked || total !== DECK_SIZE || !compositionValid} onClick={save}>{busy ? "Salvando…" : locked ? "Baralho protegido" : "Salvar e usar na Arena"}</button></div></aside>
    </div>
  </div>;
}
