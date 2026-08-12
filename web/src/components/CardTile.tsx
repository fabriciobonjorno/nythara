import { useState } from "react";
import { CardSigil, CardZoomIcon } from "./CardSigil";
import { UiIcon } from "./UiIcon";
import type { CardDefinition } from "../types";
import { translateText } from "../i18n";
import { usePreferencesStore } from "../store";

function cardArtworkPath(card: CardDefinition) {
  return `/card-art/${card.id.toLocaleLowerCase("en-US")}.webp`;
}

export function cardConfrontMetric(card: CardDefinition) {
  if (!card.confront?.legal) return "Arquivo";
  if (card.type === "Assalto") return statRange("Poder", card.confront.power, card.confront.power_max);
  if (card.type === "Guarda") return card.confront.prevent_all ? "Prevenção total" : statRange("Prevenção", card.confront.prevention, card.confront.prevention_max);
  if (card.type === "Rito") return card.confront.role ?? card.confront.keywords?.find((keyword) => keyword !== "RITO") ?? "Tática";
  return "Arquivo";
}

function statRange(label: string, minimum = 0, maximum = minimum) {
  return maximum > minimum ? `${label} ${minimum}–${maximum}` : `${label} ${minimum}`;
}

function CardArtwork({ card }: { card: CardDefinition }) {
  return (
    <div className="card-art" aria-hidden="true">
      <span className="card-art__fallback"><CardSigil sigil={card.sigil} /></span>
      <img
        className="card-art__backdrop"
        src={cardArtworkPath(card)}
        alt=""
        loading="lazy"
        decoding="async"
        onError={(event) => { event.currentTarget.hidden = true; }}
      />
      <img
        className="card-art__image"
        src={cardArtworkPath(card)}
        alt=""
        loading="lazy"
        decoding="async"
        onError={(event) => { event.currentTarget.hidden = true; }}
      />
      <span className="card-art__veil" />
    </div>
  );
}

export function CardTile({ card, quantity, selected, compact = false, disabled = false, onSelect }: {
  card: CardDefinition;
  quantity?: number;
  selected?: number;
  compact?: boolean;
  disabled?: boolean;
  onSelect?: () => void;
}) {
  const [zoomed, setZoomed] = useState(false);
  const locale = usePreferencesStore((state) => state.locale);
  const tacticalText = translateText(card.confront?.tactical_text ?? card.rules_text, locale);
  const metric = translateText(cardConfrontMetric(card), locale);
  const body = (
    <article className={`card-tile faction-${card.faction.replaceAll(" ", "-").toLowerCase()} ${compact ? "is-compact" : ""}`}>
      <div className="card-tile__edge" aria-hidden="true" />
      <header>
        <span className="card-cost" aria-label={`Custa ${card.cost} de Vitalidade`}>{card.cost}</span>
        <div><strong>{card.name}</strong><small>{card.faction}</small></div>
        <span className="sigil" aria-label={`Afinidade visual ${card.sigil}`} title="Afinidade visual; sem efeito de regra"><CardSigil sigil={card.sigil} /></span>
      </header>
      <CardArtwork card={card} />
      <div className="card-meta"><span>{translateText(card.type, locale)}</span><span>{translateText(card.rarity, locale)}</span></div>
      {card.confront?.adapted && <span className="card-adapted">ADAPTADA AO CONFRONTO</span>}
      {card.confront?.keywords?.length ? <div className="card-keywords" aria-label="Palavras-chave">{card.confront.keywords.slice(0, 3).map((keyword) => <span key={keyword}>{keyword}</span>)}</div> : null}
      <p className={compact ? "card-rules-preview" : undefined}>{tacticalText}</p>
      <div className="card-tile__bottom">
        <footer><span>{card.id}</span><span>{metric}</span></footer>
        {typeof quantity === "number" && <span className="card-owned" aria-label={typeof selected === "number" ? `${selected} de ${quantity} cópias no deck` : `${quantity} cópias disponíveis`}>
          <small>{typeof selected === "number" ? "NO DECK" : "COLEÇÃO"}</small>
          <strong>{typeof selected === "number" ? `${selected}/${quantity}` : `×${quantity}`}</strong>
        </span>}
      </div>
    </article>
  );

  return (
    <>
      <div className="card-tile-wrap">
        {body}
        <button className="card-tile-main-action" type="button" disabled={disabled} onClick={onSelect ?? (() => setZoomed(true))} aria-label={onSelect ? `${card.name}, adicionar ao deck` : `${card.name}, ampliar`} />
        {onSelect && <button className="card-zoom-button" type="button" onClick={() => setZoomed(true)} aria-label={`Ampliar ${card.name}`}><CardZoomIcon /></button>}
      </div>
      {zoomed && (
        <div className="modal-backdrop" role="presentation" onMouseDown={() => setZoomed(false)}>
          <section className="card-dialog" role="dialog" aria-modal="true" aria-labelledby={`card-${card.id}`} onMouseDown={(event) => event.stopPropagation()}>
            <button className="modal-close" type="button" autoFocus onClick={() => setZoomed(false)} aria-label="Fechar ampliação"><UiIcon name="close" /></button>
            <div className="card-dialog__visual">{body}</div>
            <div className="card-dialog__copy">
              <p className="eyebrow">{translateText(card.confront?.role ?? card.design_role, locale)}</p>
              <h2 id={`card-${card.id}`}>{card.name}</h2>
              <div className="card-dialog__stats"><span><small>CUSTO</small><b>{card.cost} Vitalidade</b></span><span><small>FUNÇÃO</small><b>{metric}</b></span></div>
              {card.confront?.keywords?.length ? <div className="card-dialog__keywords">{card.confront.keywords.map((keyword) => <span key={keyword}>{translateText(keyword, locale)}</span>)}</div> : null}
              <small className="rules-label">COMO RESOLVE</small>
              <p className="rules-copy">{tacticalText}</p>
              <blockquote>“{card.flavor}”</blockquote>
            </div>
          </section>
        </div>
      )}
    </>
  );
}
