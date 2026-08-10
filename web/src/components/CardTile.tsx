import { useState } from "react";
import type { CardDefinition } from "../types";

const sigilGlyph: Record<string, string> = {
  Presa: "◇",
  Sol: "✦",
  Espelho: "◈",
  Garra: "⌁",
  Cinza: "✣",
  Coroa: "♕",
};

export function CardTile({ card, quantity, selected, compact = false, disabled = false, onSelect }: {
  card: CardDefinition;
  quantity?: number;
  selected?: number;
  compact?: boolean;
  disabled?: boolean;
  onSelect?: () => void;
}) {
  const [zoomed, setZoomed] = useState(false);
  const body = (
    <article className={`card-tile faction-${card.faction.replaceAll(" ", "-").toLowerCase()} ${compact ? "is-compact" : ""}`}>
      <div className="card-tile__edge" aria-hidden="true" />
      <header>
        <span className="card-cost" aria-label={`Custo ${card.cost}`}>{card.cost}</span>
        <div><strong>{card.name}</strong><small>{card.faction}</small></div>
        <span className="sigil" aria-label={`Sigilo ${card.sigil}`}>{sigilGlyph[card.sigil] ?? "◆"}</span>
      </header>
      <div className="card-art" aria-hidden="true"><span>{sigilGlyph[card.sigil] ?? "◆"}</span></div>
      <div className="card-meta"><span>{card.type}</span><span>{card.rarity}</span></div>
      {!compact && <p>{card.rules_text}</p>}
      <footer><span>{card.id}</span><span>Eclipse {card.eclipse_shift > 0 ? `+${card.eclipse_shift}` : card.eclipse_shift}</span></footer>
      {typeof quantity === "number" && <span className="card-owned">{selected ?? 0}/{quantity}</span>}
    </article>
  );

  return (
    <>
      <div className="card-tile-wrap">
        {body}
        <button className="card-tile-main-action" type="button" disabled={disabled} onClick={onSelect ?? (() => setZoomed(true))} aria-label={onSelect ? `${card.name}, adicionar ao deck` : `${card.name}, ampliar`} />
        {onSelect && <button className="card-zoom-button" type="button" onClick={() => setZoomed(true)} aria-label={`Ampliar ${card.name}`}>⌕</button>}
      </div>
      {zoomed && (
        <div className="modal-backdrop" role="presentation" onMouseDown={() => setZoomed(false)}>
          <section className="card-dialog" role="dialog" aria-modal="true" aria-labelledby={`card-${card.id}`} onMouseDown={(event) => event.stopPropagation()}>
            <button className="modal-close" type="button" autoFocus onClick={() => setZoomed(false)} aria-label="Fechar ampliação">×</button>
            <div className="card-dialog__visual">{body}</div>
            <div className="card-dialog__copy">
              <p className="eyebrow">{card.design_role}</p>
              <h2 id={`card-${card.id}`}>{card.name}</h2>
              <p className="rules-copy">{card.rules_text}</p>
              <blockquote>“{card.flavor}”</blockquote>
            </div>
          </section>
        </div>
      )}
    </>
  );
}

export function MiniBattleCard({ card, instanceId, selected, onPlay, onZoom }: {
  card: CardDefinition;
  instanceId: string;
  selected?: boolean;
  onPlay?: () => void;
  onZoom: (card: CardDefinition) => void;
}) {
  return (
    <div className={`hand-card ${selected ? "is-selected" : ""}`}>
      <button type="button" className="hand-card__play" onClick={onPlay} disabled={!onPlay} aria-label={`${card.name}, custo ${card.cost}. ${card.rules_text}`}>
        <span className="card-cost">{card.cost}</span>
        <strong>{card.name}</strong>
        <span className="hand-card__sigil">{sigilGlyph[card.sigil] ?? "◆"} {card.sigil}</span>
        <small>{card.type}</small>
      </button>
      <button type="button" className="hand-card__zoom" onClick={() => onZoom(card)} aria-label={`Ampliar ${card.name}`}>⌕</button>
      <span className="sr-only">Instância {instanceId}</span>
    </div>
  );
}
