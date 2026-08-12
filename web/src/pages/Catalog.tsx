import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { CardTile } from "../components/CardTile";
import { ChampionEmblem } from "../components/ChampionEmblem";
import { DataLoadError } from "../components/DataLoadError";
import { UiIcon } from "../components/UiIcon";
import { useCards, useChampions, useCollection, useProgress, useRuleset } from "../queries";
import type { Champion } from "../types";
import { translateText } from "../i18n";
import { usePreferencesStore } from "../store";

export function CollectionPage() {
  const cardsQuery = useCards();
  const collectionQuery = useCollection();
	const progressQuery = useProgress();
  const { data: catalog, isLoading } = cardsQuery;
  const { data: collection } = collectionQuery;
	const accountLevel = progressQuery.data?.account.level ?? 1;
  const [search, setSearch] = useState("");
  const [type, setType] = useState("Todos");
  const [faction, setFaction] = useState("Todas");
  const [mode, setMode] = useState("Competitivas");
  const [visibleCount, setVisibleCount] = useState(30);
  const owned = new Map(collection?.cards.map((item) => [item.card_id, item.quantity]));
  const cards = useMemo(() => catalog?.cards.filter((card) =>
    (type === "Todos" || card.type === type) && (faction === "Todas" || card.faction === faction) &&
    (mode === "Todas" || (mode === "Competitivas" ? card.confront?.legal : !card.confront?.legal)) &&
    `${card.name} ${card.rules_text} ${card.sigil}`.toLocaleLowerCase("pt-BR").includes(search.toLocaleLowerCase("pt-BR")),
  ) ?? [], [catalog, faction, mode, search, type]);
  const factions = [...new Set(catalog?.cards.map((card) => card.faction) ?? [])];
  useEffect(() => setVisibleCount(30), [faction, mode, search, type]);
  const visibleCards = cards.slice(0, visibleCount);

  const legalCount = catalog?.cards.filter((card) => card.confront?.legal).length ?? 0;
  return <div className="page catalog-page"><PageHeader eyebrow="ARQUIVO DO VÉU" title="Coleção" copy={`${catalog?.cards.length ?? 130} cartas ilustradas no catálogo; ${legalCount} formam o pool competitivo do Modo Confronto.`} count={`${cards.length} exibidas`} />
    <div className="filter-bar"><label className="search-field"><span className="sr-only">Buscar carta</span><input type="search" placeholder="Buscar por nome, ação ou texto…" value={search} onChange={(event) => setSearch(event.target.value)} /></label><label>Modo<select value={mode} onChange={(event) => setMode(event.target.value)}><option value="Competitivas">Competitivas</option><option value="Arquivo">Arquivo</option><option value="Todas">Todas</option></select></label><label>Tipo<select value={type} onChange={(event) => setType(event.target.value)}><option value="Todos">Todos</option>{["Assalto", "Guarda", "Rito", "Relíquia", "Manifestação"].map((item) => <option value={item} key={item}>{item}</option>)}</select></label><label>Facção<select value={faction} onChange={(event) => setFaction(event.target.value)}><option value="Todas">Todas</option>{factions.map((item) => <option value={item} key={item}>{item}</option>)}</select></label></div>
    {cardsQuery.isError || collectionQuery.isError || progressQuery.isError
	  ? <DataLoadError onRetry={() => { void cardsQuery.refetch(); void collectionQuery.refetch(); void progressQuery.refetch(); }} />
	  : isLoading || collectionQuery.isLoading || progressQuery.isLoading ? <LoadingGrid /> : <>
		<div className="card-grid">{visibleCards.map((card) => <CardTile key={card.id} card={card} quantity={owned.get(card.id) ?? 0} currentLevel={accountLevel} />)}</div>
        {visibleCards.length < cards.length && <button className="secondary-button card-load-more" type="button" onClick={() => setVisibleCount((count) => count + 30)}>Carregar mais cartas ({cards.length - visibleCards.length} restantes)</button>}
      </>}
  </div>;
}

export function ChampionsPage() {
  const championsQuery = useChampions();
  const collectionQuery = useCollection();
  const rulesetQuery = useRuleset();
  // A Vitalidade vem do ruleset ativo: repetir o número em texto fixo já
  // prometeu 30 depois que o formato passou a começar em 56.
  const vitality = rulesetQuery.data?.starting_vitality;
  const { data, isLoading } = championsQuery;
  const { data: collection } = collectionQuery;
  const owned = new Set(collection?.champions ?? []);
  return <div className="page champions-page"><PageHeader eyebrow="SUA PRESENÇA NA ARENA" title="Avatares" copy={`Cada Avatar traz um poder próprio e todos começam com${vitality ? ` ${vitality}` : " a mesma"} Vitalidade. O poder muda como você joga, nunca quantas cartas você tem.`} count={`${data?.champions.length ?? 10} visuais`} />
    {championsQuery.isError || collectionQuery.isError
      ? <DataLoadError onRetry={() => { void championsQuery.refetch(); void collectionQuery.refetch(); }} />
      : isLoading || collectionQuery.isLoading ? <LoadingGrid /> : <div className="champion-grid">{data?.champions.map((champion) => <ChampionCard champion={champion} vitality={vitality} available={owned.has(champion.id)} key={champion.id} />)}</div>}
  </div>;
}

function ChampionCard({ champion, vitality, available }: { champion: Champion; vitality?: number; available: boolean }) {
  const locale = usePreferencesStore((state) => state.locale);
  const power = champion.confront_power ? translateText(champion.confront_power, locale) : "Sem poder neste conjunto de regras.";
  const factionClass = champion.faction.normalize("NFD").replace(/[\u0300-\u036f]/g, "").replaceAll(" ", "-").toLowerCase();
  return <article className={`champion-card faction-${factionClass} ${available ? "is-available" : "is-locked"}`}>
    <div className="champion-portrait" aria-hidden="true"><ChampionEmblem id={champion.id} faction={champion.faction} /><span>{champion.name.split(",")[0]}</span></div>
    <div className="champion-card__body">
      <div className="champion-card__title"><div><p className="eyebrow">IDENTIDADE {champion.faction}</p><h2>{champion.name}</h2></div><span className="vitality"><UiIcon name="heart" />{vitality ?? "—"}</span></div>
      <dl><div><dt>Poder</dt><dd>{power}</dd></div><div><dt>Equidade</dt><dd>Todos partem da mesma Vitalidade e do mesmo baralho de 30 cartas.</dd></div><div className="eclipse-form"><dt>Presença</dt><dd>Retrato e nome exibidos na Arena e no histórico.</dd></div></dl>
      <footer className="champion-card__footer"><span className="owned-badge">{available ? "Disponível" : "Bloqueado"}</span>{available && <span className="champion-card__cta">Usar no meu baralho <b><UiIcon name="arrow-right" /></b></span>}</footer>
    </div>
    {available && <Link className="champion-card__main-action" to={`/decks?avatar=${champion.id}`} aria-label={`Usar ${champion.name} como Avatar`}><span className="sr-only">Usar {champion.name} como Avatar</span></Link>}
  </article>;
}

function PageHeader({ eyebrow, title, copy, count }: { eyebrow: string; title: string; copy: string; count: string }) {
  return <header className="page-header"><div><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p>{copy}</p></div><span className="count-badge">{count}</span></header>;
}

function LoadingGrid() { return <div className="loading-grid" aria-label="Carregando"><span /><span /><span /><span /></div>; }
