import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BattleState, CardDefinition } from "../types";
import { DecisionSheet } from "./DecisionSheet";
import { usePreferencesStore } from "../store";

beforeEach(() => {
  usePreferencesStore.setState({ locale: "pt-BR" });
});

const definitions = [
  { id: "VR-A", name: "Cinza", type: "Rito", cost: 1, faction: "Neutra", rarity: "Comum", eclipse_shift: 0, sigil: "Espelho", rules_text: "", flavor: "", design_role: "" },
  { id: "VR-B", name: "Vidro", type: "Guarda", cost: 1, faction: "Neutra", rarity: "Comum", eclipse_shift: 0, sigil: "Espelho", rules_text: "", flavor: "", design_role: "" },
  { id: "VR-C", name: "Rubro", type: "Assalto", cost: 1, faction: "Neutra", rarity: "Comum", eclipse_shift: 0, sigil: "Espelho", rules_text: "", flavor: "", design_role: "" },
] as CardDefinition[];
const byId = new Map(definitions.map((card) => [card.id, card]));
const cards = {
  a: { id: "a", def: "VR-A", owner: 0, zone: "hand" },
  b: { id: "b", def: "VR-B", owner: 0, zone: "hand" },
  c: { id: "c", def: "VR-C", owner: 0, zone: "hand" },
} as BattleState["cards"];

function pending(id = 7) {
  return { id, player: 0, kind: "discard_n", options: ["a", "b", "c"], n: 2 } as NonNullable<BattleState["pending"]>;
}

describe("DecisionSheet", () => {
  it("mantém a ordem do toque, limita N, renumera e confirma exatamente N", () => {
    const confirm = vi.fn();
    render(<DecisionSheet pending={pending()} cards={cards} byId={byId} busy={false} onConfirm={confirm} />);

    const submit = screen.getByRole("button", { name: "Confirmar escolha" });
    const cinza = screen.getByRole("button", { name: "Cinza" });
    const vidro = screen.getByRole("button", { name: "Vidro" });
    const rubro = screen.getByRole("button", { name: "Rubro" });
    expect(submit).toBeDisabled();

    fireEvent.click(cinza);
    fireEvent.click(vidro);
    expect(within(screen.getByRole("button", { name: "Cinza, marcada" })).getByText("1")).toBeTruthy();
    expect(within(screen.getByRole("button", { name: "Vidro, marcada" })).getByText("2")).toBeTruthy();
    expect(submit).toBeEnabled();

    fireEvent.click(rubro);
    expect(rubro).toHaveAttribute("aria-pressed", "false");

    fireEvent.click(screen.getByRole("button", { name: "Cinza, marcada" }));
    expect(within(screen.getByRole("button", { name: "Vidro, marcada" })).getByText("1")).toBeTruthy();
    expect(submit).toBeDisabled();
    fireEvent.click(rubro);
    expect(within(screen.getByRole("button", { name: "Rubro, marcada" })).getByText("2")).toBeTruthy();

    fireEvent.click(submit);
    expect(confirm).toHaveBeenCalledOnce();
    expect(confirm).toHaveBeenCalledWith(["b", "c"]);
  });

  it("limpa a seleção em outra decisão e bloqueia confirmação ocupada", () => {
    const confirm = vi.fn();
    const { rerender } = render(<DecisionSheet pending={pending()} cards={cards} byId={byId} busy={false} onConfirm={confirm} />);
    fireEvent.click(screen.getByRole("button", { name: "Cinza" }));
    fireEvent.click(screen.getByRole("button", { name: "Vidro" }));
    expect(screen.getByRole("button", { name: "Confirmar escolha" })).toBeEnabled();

    rerender(<DecisionSheet pending={pending(8)} cards={cards} byId={byId} busy={true} onConfirm={confirm} />);
    expect(screen.getByText("0/2 selecionadas")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Enviando…" })).toBeDisabled();
  });
});
