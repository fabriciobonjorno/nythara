import type { CardDefinition } from "./types";

// Leitura de carta no padrão dos duelos clássicos: a razão CUSTO/EFEITO fica no
// alto da moldura e a descrição declara o preço antes do que a carta faz.
// Nada aqui decide regra — é apresentação do que a engine já publicou.

export interface CardStat {
  value: string;
  label: string;
}

export function cardStat(card: CardDefinition): CardStat {
  const confront = card.confront;
  if (!confront?.legal) return { value: "—", label: "Fora do Confronto" };
  if (card.type === "Assalto") return { value: range(confront.power, confront.power_max), label: "Poder" };
  if (card.type === "Guarda") {
    if (confront.prevent_all) return { value: "∞", label: "Prevenção" };
    return { value: range(confront.prevention, confront.prevention_max), label: "Prevenção" };
  }
  return { value: "◈", label: "Efeito" };
}

function range(minimum = 0, maximum = minimum) {
  return maximum > minimum ? `${minimum}–${maximum}` : `${minimum}`;
}

/** Razão custo/efeito exibida no topo da carta, ex.: "2/5". */
export function cardRatio(card: CardDefinition) {
  const stat = cardStat(card);
  return {
    text: `${card.cost}/${stat.value}`,
    label: `Custa ${card.cost} de Vitalidade. ${stat.label} ${stat.value}.`,
  };
}

/** Descrição da carta: preço primeiro, efeito depois. */
export function cardBrief(card: CardDefinition) {
  const effect = card.confront?.tactical_text ?? card.rules_text;
  return `${cardPrice(card)} ${effect}`;
}

export function cardPrice(card: CardDefinition) {
  if (card.cost <= 0) return "Não custa Vitalidade.";
  return `Custa ${card.cost} ${card.cost === 1 ? "ponto" : "pontos"} de Vitalidade.`;
}

/** Como a carta age na mesa, em uma linha, para selos e placar. */
export function cardRole(card: CardDefinition) {
  if (card.type === "Assalto") return "Ataca. O rival pode responder com uma Guarda.";
  if (card.type === "Guarda") return "Responde a um Assalto e previne dano.";
  if (card.type === "Rito") return "Efeito imediato na sua janela de Rito.";
  return "Não participa do Modo Confronto.";
}
