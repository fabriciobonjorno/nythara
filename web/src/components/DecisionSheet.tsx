import { useEffect, useState } from "react";
import type { BattleState, CardDefinition } from "../types";
import { DuelCard } from "./DuelCard";
import { translateText } from "../i18n";
import { usePreferencesStore } from "../store";

// A carta pediu uma escolha e a mesa está travada até a resposta. Este
// componente só coleta uma seleção ordenada; toda validação continua no
// servidor authoritative.
export function DecisionSheet({ pending, cards, byId, busy, onConfirm }: {
  pending: NonNullable<BattleState["pending"]>;
  cards: BattleState["cards"];
  byId: Map<string, CardDefinition>;
  busy: boolean;
  onConfirm: (picked: string[]) => void;
}) {
  const locale = usePreferencesStore((state) => state.locale);
  const [picks, setPicks] = useState<string[]>([]);
  useEffect(() => { setPicks([]); }, [pending.id]);
  const options = (pending.options ?? [])
    .map((id) => ({ id, card: byId.get(cards[id]?.def ?? "") }))
    .filter((option): option is { id: string; card: CardDefinition } => Boolean(option.card));
  const need = Math.min(pending.n, options.length);
  const source = pending.source ? byId.get(pending.source)?.name : undefined;
  const sourceTitle = pending.kind === "discard_n"
    ? `Descarte ${need} ${need === 1 ? "carta" : "cartas"}`
    : "Faça sua escolha";
  const title = translateText(sourceTitle, locale);
  const toggle = (id: string) => setPicks((current) => current.includes(id)
    ? current.filter((pick) => pick !== id)
    : current.length < need ? [...current, id] : current);

  return <div className="decision-layer" role="dialog" aria-modal="true" aria-label={title}>
    <div className="decision-sheet">
      <p className="inspector__eyebrow">{source ? `${source.toLocaleUpperCase(locale)} ${translateText("EXIGE", locale)}` : translateText("A CARTA EXIGE", locale)}</p>
      <h2>{title}</h2>
      <p className="decision-sheet__hint">{translateText("Toque para marcar. A mesa segue depois da confirmação.", locale)}</p>
      <ul className="decision-sheet__options">
        {options.map(({ id, card }) => {
          const order = picks.indexOf(id);
          return <li key={id}>
            <button type="button" className={`decision-option ${order >= 0 ? "is-picked" : ""}`}
              onClick={() => toggle(id)} aria-pressed={order >= 0}
              aria-label={`${card.name}${order >= 0 ? `, ${translateText("marcada", locale)}` : ""}`}>
              {order >= 0 && <b className="decision-option__badge">{order + 1}</b>}
              <DuelCard card={card} size="hand" />
            </button>
          </li>;
        })}
      </ul>
      <footer className="decision-sheet__foot">
        <small>{picks.length}/{need} {translateText(need === 1 ? "selecionada" : "selecionadas", locale)}</small>
        <button type="button" className="decision-sheet__confirm" disabled={busy || picks.length !== need}
          onClick={() => onConfirm(picks)}>
          {translateText(busy ? "Enviando…" : "Confirmar escolha", locale)}
        </button>
      </footer>
    </div>
  </div>;
}
