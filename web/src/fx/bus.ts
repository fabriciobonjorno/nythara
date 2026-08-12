// Barramento minúsculo entre a leitura de eventos (useBattleFx) e o palco de
// partículas. Desacopla de propósito: quem interpreta o log não precisa saber
// onde os alvos estão na tela, e o canvas não precisa conhecer o jogo.

export type FxKind = "sparks" | "ring" | "shards" | "motes" | "glint";

export interface FxSpawn {
  kind: FxKind;
  /** Nome de um alvo marcado com data-fx no DOM da mesa. */
  target: string;
  /** Intensidade 0..1 — escala contagem, velocidade e tamanho. */
  power?: number;
  color?: string;
}

type Listener = (spawn: FxSpawn) => void;
const listeners = new Set<Listener>();

export function onFx(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function emitFx(spawn: FxSpawn) {
  for (const listener of listeners) listener(spawn);
}
