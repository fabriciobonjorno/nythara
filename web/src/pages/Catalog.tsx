import { useMemo, useState } from "react";
import { CardTile } from "../components/CardTile";
import { useCards, useChampions, useCollection } from "../queries";

export function CollectionPage() {
  const { data: catalog, isLoading } = useCards();
  const { data: collection } = useCollection();
  const [search, setSearch] = useState("");
  const [type, setType] = useState("Todos");
  const [faction, setFaction] = useState("Todas");
  const owned = new Map(collection?.cards.map((item) => [item.card_id, item.quantity]));
  const cards = useMemo(() => catalog?.cards.filter((card) =>
    (type === "Todos" || card.type === type) && (faction === "Todas" || card.faction === faction) &&
    `${card.name} ${card.rules_text} ${card.sigil}`.toLocaleLowerCase("pt-BR").includes(search.toLocaleLowerCase("pt-BR")),
  ) ?? [], [catalog, faction, search, type]);
  const factions = [...new Set(catalog?.cards.map((card) => card.faction) ?? [])];

  return <div className="page catalog-page"><PageHeader eyebrow="ARQUIVO DO VÉU" title="Coleção" copy="Todas as 80 cartas do Alpha estão disponíveis para competir." count={`${cards.length} cartas`} />
    <div className="filter-bar"><label className="search-field"><span className="sr-only">Buscar carta</span><input type="search" placeholder="Buscar por nome, texto ou Sigilo…" value={search} onChange={(event) => setSearch(event.target.value)} /></label><label>Tipo<select value={type} onChange={(event) => setType(event.target.value)}><option>Todos</option>{["Assalto", "Guarda", "Rito", "Relíquia", "Manifestação"].map((item) => <option key={item}>{item}</option>)}</select></label><label>Facção<select value={faction} onChange={(event) => setFaction(event.target.value)}><option>Todas</option>{factions.map((item) => <option key={item}>{item}</option>)}</select></label></div>
    {isLoading ? <LoadingGrid /> : <div className="card-grid">{cards.map((card) => <CardTile key={card.id} card={card} quantity={owned.get(card.id) ?? 0} />)}</div>}
  </div>;
}

export function ChampionsPage() {
  const { data, isLoading } = useChampions();
  const { data: collection } = useCollection();
  const owned = new Set(collection?.champions ?? []);
  return <div className="page champions-page"><PageHeader eyebrow="PORTADORES DO DESTINO" title="Campeões" copy="Cada Campeão transforma o Eclipse em uma estratégia diferente." count={`${data?.champions.length ?? 10} campeões`} />
    {isLoading ? <LoadingGrid /> : <div className="champion-grid">{data?.champions.map((champion, index) => <article className="champion-card" key={champion.id}><div className="champion-portrait" aria-hidden="true"><span>{["♜", "♞", "☀", "✧", "◈", "◎", "⌁", "◒", "✣", "♙"][index]}</span></div><div className="champion-card__body"><div className="champion-card__title"><div><p className="eyebrow">{champion.faction}</p><h2>{champion.name}</h2></div><span className="vitality">♥ {champion.vitality}</span></div><dl><div><dt>Passiva</dt><dd>{champion.passive}</dd></div><div><dt>Ultimate</dt><dd>{champion.ultimate}</dd></div><div className="eclipse-form"><dt>Forma de Eclipse</dt><dd>{champion.eclipse_form}</dd></div></dl><span className="owned-badge">{owned.has(champion.id) ? "Disponível" : "Bloqueado"}</span></div></article>)}</div>}
  </div>;
}

function PageHeader({ eyebrow, title, copy, count }: { eyebrow: string; title: string; copy: string; count: string }) {
  return <header className="page-header"><div><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p>{copy}</p></div><span className="count-badge">{count}</span></header>;
}

function LoadingGrid() { return <div className="loading-grid" aria-label="Carregando"><span /><span /><span /><span /></div>; }
