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

/** Números que a lição precisa citar; vêm do ruleset ativo, nunca do código. */
export interface GuidedRules {
  guardLeakCap: number;
}

export function guidedLesson(state: BattleState, mySlot: number, progress: GuidedProgress,
  rules: GuidedRules = { guardLeakCap: 0 }): GuidedLesson {
  if (state.over) return {
    eyebrow: "TREINO CONCLUÍDO",
    title: `${progress.completed}/3 fundamentos praticados`,
    copy: "O resultado mostra o que apareceu nesta partida e o que vale repetir na próxima.",
    tone: "complete",
  };
  // O baralho é o relógio do duelo longo: quando ele encurta, a lição deixa de
  // ser sobre o confronto da vez e passa a ser sobre durar.
  const myDeck = state.players[mySlot]?.deck_count ?? 99;
  if (myDeck > 0 && myDeck <= 6) return {
    eyebrow: "O BARALHO É O RELÓGIO",
    title: `Restam ${myDeck} cartas no seu baralho`,
    copy: "Quando ele acabar, cada compra vira Fadiga e o dano cresce a cada turno. Daqui em diante, gastar carta sem necessidade é gastar tempo de vida.",
    tone: "observe",
  };
  if (state.active !== mySlot) return {
    eyebrow: "OBSERVE O RIVAL",
    title: "Leia a carta que chega ao centro",
    copy: "Você não precisa agir agora. Repare no Poder, no custo e em como a mesa abre a próxima resposta.",
    tone: "observe",
  };
  if (state.phase === "assalto") return {
    eyebrow: "FUNDAMENTO · ASSALTO",
    title: "Crie pressão ou preserve o baralho",
    copy: "A partida é longa: um Assalto que o rival bloqueia custou a você uma carta e a Vitalidade dela. Atacar quando ele não tem resposta vale mais que atacar sempre.",
    tone: "assault",
  };
  if (state.phase === "guarda") {
    const power = state.confront?.power ?? 0;
    return {
      eyebrow: "FUNDAMENTO · GUARDA",
      title: "Defender quase sempre compensa",
      copy: rules.guardLeakCap > 0
        ? `O Assalto chega com Poder ${power}. Uma Guarda comprometida nunca deixa passar mais que ${rules.guardLeakCap} — mesmo contra um golpe grande. Quem não responde leva o valor inteiro.`
        : `O Assalto chega com Poder ${power}. Uma Guarda reduz o dano; aceitar preserva a carta para outro confronto.`,
      tone: "guard",
    };
  }
  if (state.phase === "rito") return {
    eyebrow: "FUNDAMENTO · RITO",
    title: "Prepare o próximo confronto",
    copy: "Ritos compram, curam e aplicam pressão — é onde a partida longa se decide. Se nenhum ajudar agora, encerre o turno e guarde a Vitalidade.",
    tone: "rite",
  };
  return {
    eyebrow: "A MESA ESTÁ RESOLVENDO",
    title: "Acompanhe a sequência",
    copy: "O servidor está concluindo compra, dano ou efeitos antes de abrir outra decisão.",
    tone: "observe",
  };
}
