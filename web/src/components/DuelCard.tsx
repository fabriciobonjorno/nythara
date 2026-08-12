import { CardSigil } from "./CardSigil";
import { cardBrief, cardRatio, cardStat } from "../cardText";
import type { CardDefinition } from "../types";
import { translateText } from "../i18n";
import { usePreferencesStore } from "../store";

// Moldura única de carta do duelo, usada na mão, na mesa e no inspetor.
// A arte é 2:3 e a carta também: assim a ilustração entra inteira, sem corte.
// Nenhum arquivo de arte é alterado — a moldura é que se ajusta a ela.

export type DuelCardSize = "hand" | "table" | "inspect";

const ACCENTS = /[̀-ͯ]/g;

function artworkPath(card: CardDefinition) {
  return `/card-art/${card.id.toLocaleLowerCase("en-US")}.webp`;
}

function typeSlug(card: CardDefinition) {
  return card.type.normalize("NFD").replace(ACCENTS, "").toLocaleLowerCase("pt-BR");
}

export function DuelCard({ card, size, broken, dimmed, banner }: {
  card: CardDefinition;
  size: DuelCardSize;
  broken?: boolean;
  dimmed?: boolean;
  banner?: string;
}) {
  const locale = usePreferencesStore((state) => state.locale);
  const ratio = cardRatio(card);
  const stat = cardStat(card);
  const brief = translateText(cardBrief(card), locale);
  const ratioLabel = translateText(ratio.label, locale);
  return (
    <article
      className={`duel-card size-${size} kind-${typeSlug(card)} ${broken ? "is-broken" : ""} ${dimmed ? "is-dimmed" : ""}`}
      aria-label={`${card.name}. ${ratioLabel} ${brief}`}
    >
      <div className="duel-card__art" aria-hidden="true">
        <span className="duel-card__fallback"><CardSigil sigil={card.sigil} /></span>
        <img src={artworkPath(card)} alt="" loading="lazy" decoding="async" onError={(event) => { event.currentTarget.hidden = true; }} />
      </div>
      {banner && <span className="duel-card__banner">{banner}</span>}
      <header className="duel-card__head">
        <strong>{card.name}</strong>
        <span className="duel-card__ratio" title={ratioLabel}>{ratio.text}</span>
      </header>
      <div className="duel-card__body">
        <p className="duel-card__text">{brief}</p>
        <footer className="duel-card__foot">
          <span className="duel-card__kind">{translateText(card.type, locale)}</span>
          <span className="duel-card__stat">{translateText(stat.label, locale)} <b>{stat.value}</b></span>
        </footer>
      </div>
      <span className="duel-card__crack" aria-hidden="true" />
    </article>
  );
}
