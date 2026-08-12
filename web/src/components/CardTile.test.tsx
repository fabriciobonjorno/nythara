import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CardDefinition } from "../types";
import { usePreferencesStore } from "../store";
import { CardTile } from "./CardTile";

beforeEach(() => usePreferencesStore.setState({ locale: "pt-BR" }));

const legendary = {
  id: "VR-012",
  name: "Soberania Carmesim",
  faction: "Casa Vhal",
  type: "Assalto",
  rarity: "Lendária",
  unlock_level: 10,
  cost: 5,
  eclipse_shift: 2,
  sigil: "Presa",
  rules_text: "Poder 7.",
  flavor: "O trono reconhece quem sobrevive.",
  design_role: "finalizador",
  confront: { legal: true, power: 7 },
} as CardDefinition;

describe("CardTile progression gate", () => {
  it("mostra o marco e só habilita a seleção no nível exigido", () => {
    const select = vi.fn();
    const { rerender } = render(<CardTile card={legendary} currentLevel={1} quantity={0} disabled onSelect={select} />);
    expect(screen.getByText("LIBERA NO NÍVEL 10")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Soberania Carmesim, adicionar ao baralho" })).toBeDisabled();

    rerender(<CardTile card={legendary} currentLevel={10} quantity={1} onSelect={select} />);
    expect(screen.queryByText("LIBERA NO NÍVEL 10")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Soberania Carmesim, adicionar ao baralho" }));
    expect(select).toHaveBeenCalledOnce();
  });
});
