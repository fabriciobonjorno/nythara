import type { CardDefinition } from "./types";

type DeckSignalTone = "good" | "attention" | "neutral";

export interface DeckSignal {
  tone: DeckSignalTone;
  title: string;
  copy: string;
}

export interface DeckAnalysis {
  ready: boolean;
  averageCost: number;
  lowCostCards: number;
  highCostCards: number;
  averagePower: number | null;
  averagePrevention: number | null;
  totalPreventionCards: number;
  riteRoles: string[];
  signals: DeckSignal[];
}

const MINIMUMS: Record<"Assalto" | "Guarda" | "Rito", number> = {
  Assalto: 8,
  Guarda: 8,
  Rito: 4,
};

function normalize(value: string) {
  return value.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLocaleLowerCase("pt-BR");
}

function riteRole(card: CardDefinition) {
  const text = normalize(`${card.rules_text} ${card.design_role}`);
  if (/compre|mao|descarte/.test(text)) return "Mão";
  if (/cure|restaure/.test(text)) return "Recuperação";
  if (/sangramento|maldicao|exposto/.test(text)) return "Pressão";
  if (/ward|previna|prevencao/.test(text)) return "Proteção";
  if (/copie|devolva|recupere|exile|controle/.test(text)) return "Controle";
  return "Utilidade";
}

/**
 * Leitura local, explicativa e deliberadamente conservadora. Não estima taxa
 * de vitória e não replica a validação de legalidade do servidor.
 */
export function analyzeDeck(cards: CardDefinition[], counts: Record<string, number>): DeckAnalysis {
  const selected = cards
    .map((card) => ({ card, quantity: Math.max(0, counts[card.id] ?? 0) }))
    .filter((entry) => entry.quantity > 0);
  const total = selected.reduce((sum, entry) => sum + entry.quantity, 0);
  const totalCost = selected.reduce((sum, entry) => sum + entry.card.cost * entry.quantity, 0);
  const typeCounts = { Assalto: 0, Guarda: 0, Rito: 0 };
  for (const entry of selected) {
    if (entry.card.type in typeCounts) typeCounts[entry.card.type as keyof typeof typeCounts] += entry.quantity;
  }

  const weightedAverage = (type: "Assalto" | "Guarda", metric: "power" | "prevention") => {
    const entries = selected.filter((entry) => entry.card.type === type && typeof entry.card.confront?.[metric] === "number");
    const amount = entries.reduce((sum, entry) => sum + entry.quantity, 0);
    if (!amount) return null;
    return entries.reduce((sum, entry) => sum + (entry.card.confront?.[metric] ?? 0) * entry.quantity, 0) / amount;
  };

  const averageCost = total ? totalCost / total : 0;
  const lowCostCards = selected.filter((entry) => entry.card.cost <= 2).reduce((sum, entry) => sum + entry.quantity, 0);
  const highCostCards = selected.filter((entry) => entry.card.cost >= 4).reduce((sum, entry) => sum + entry.quantity, 0);
  const totalPreventionCards = selected
    .filter((entry) => entry.card.type === "Guarda" && entry.card.confront?.prevent_all)
    .reduce((sum, entry) => sum + entry.quantity, 0);
  const riteRoles = Array.from(new Set(selected.filter((entry) => entry.card.type === "Rito").map((entry) => riteRole(entry.card)))).sort();
  const ready = total === 30 && (Object.keys(MINIMUMS) as Array<keyof typeof MINIMUMS>)
    .every((type) => typeCounts[type] >= MINIMUMS[type]);

  const signals: DeckSignal[] = [];
  if (total !== 30) {
    signals.push({ tone: "neutral", title: `${total}/30 cartas`, copy: total < 30 ? `Escolha mais ${30 - total} para completar a lista.` : `Retire ${total - 30} para voltar ao limite.` });
  }
  const missingTypes = (Object.keys(MINIMUMS) as Array<keyof typeof MINIMUMS>)
    .filter((type) => typeCounts[type] < MINIMUMS[type])
    .map((type) => `${MINIMUMS[type] - typeCounts[type]} ${type}`);
  if (missingTypes.length) {
    signals.push({ tone: "attention", title: "Composição incompleta", copy: `Ainda faltam ${missingTypes.join(", ")} para a validação competitiva.` });
  }
  if (total >= 20 && highCostCards > total * 0.35) {
    signals.push({ tone: "attention", title: "Curva exigente", copy: `${highCostCards} cartas custam 4 ou mais de Vitalidade. Passar cedo pode ser necessário para não se consumir antes do golpe decisivo.` });
  } else if (total >= 20 && lowCostCards < Math.ceil(total * 0.2)) {
    signals.push({ tone: "attention", title: "Poucas respostas leves", copy: `Só ${lowCostCards} cartas custam até 2. Uma mão cara reduz escolhas quando sua Vitalidade já está baixa.` });
  }
  if (typeCounts.Rito >= MINIMUMS.Rito && riteRoles.length < 2) {
    signals.push({ tone: "neutral", title: "Ritos concentrados", copy: "Seus Ritos cumprem um único papel. Variar função aumenta as linhas de decisão, sem garantir vantagem." });
  }
  if (ready && !signals.some((signal) => signal.tone === "attention")) {
    signals.push({ tone: "good", title: "Estrutura pronta", copy: "A lista fecha a regra competitiva e não apresenta alerta estrutural. O próximo passo é testá-la na Arena." });
  }

  return {
    ready,
    averageCost,
    lowCostCards,
    highCostCards,
    averagePower: weightedAverage("Assalto", "power"),
    averagePrevention: weightedAverage("Guarda", "prevention"),
    totalPreventionCards,
    riteRoles,
    signals: signals.slice(0, 3),
  };
}
