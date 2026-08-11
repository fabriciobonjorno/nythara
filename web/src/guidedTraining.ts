import type { BattleEvent, BattleState, CardDefinition } from "./types";

export interface GuidedProgress {
  assault: boolean;
  guard: boolean;
  rite: boolean;
  completed: number;
}

export interface GuidedLesson {
  eyebrow: string;
  title: string;
  copy: string;
  tone: "observe" | "assault" | "guard" | "rite" | "complete";
}

export function buildGuidedProgress(events: BattleEvent[], mySlot: number,
  cards: Map<string, CardDefinition>): GuidedProgress {
  const assault = events.some((event) => event.kind === "confrontation_opened" && event.p === mySlot);
  const guard = events.some((event) => event.kind === "guard_committed" && event.p === mySlot);
  const rite = events.some((event) => event.kind === "card_played" && event.p === mySlot
    && Boolean(event.def && cards.get(event.def)?.type === "Rito"));
  return { assault, guard, rite, completed: Number(assault) + Number(guard) + Number(rite) };
}

export function guidedLesson(state: BattleState, mySlot: number, progress: GuidedProgress): GuidedLesson {
  if (state.over) return {
    eyebrow: "TREINO CONCLUÍDO",
    title: `${progress.completed}/3 fundamentos praticados`,
    copy: "O resultado mostrará o que apareceu nesta partida e o que vale repetir.",
    tone: "complete",
  };
  if (state.active !== mySlot) return {
    eyebrow: "OBSERVE O RIVAL",
    title: "Leia a carta que chega ao centro",
    copy: "Você não precisa agir agora. Repare no Poder, no custo e em como a mesa abre a próxima resposta.",
    tone: "observe",
  };
  if (state.phase === "assalto") return {
    eyebrow: "FUNDAMENTO · ASSALTO",
    title: "Crie pressão ou preserve Vitalidade",
    copy: "Arraste uma carta iluminada ao centro. Passar também é uma decisão válida quando o custo deixaria você vulnerável.",
    tone: "assault",
  };
  if (state.phase === "guarda") return {
    eyebrow: "FUNDAMENTO · GUARDA",
    title: "Compare Poder e Prevenção",
    copy: `O Assalto chega com Poder ${state.confront?.power ?? 0}. Uma Guarda reduz o dano; aceitar preserva a carta para outro confronto.`,
    tone: "guard",
  };
  if (state.phase === "rito") return {
    eyebrow: "FUNDAMENTO · RITO",
    title: "Prepare o próximo confronto",
    copy: "Ritos podem comprar, curar ou aplicar pressão. Se nenhum ajudar agora, encerre o turno sem gastar Vitalidade.",
    tone: "rite",
  };
  return {
    eyebrow: "A MESA ESTÁ RESOLVENDO",
    title: "Acompanhe a sequência",
    copy: "O servidor está concluindo compra, dano ou efeitos antes de abrir outra decisão.",
    tone: "observe",
  };
}
