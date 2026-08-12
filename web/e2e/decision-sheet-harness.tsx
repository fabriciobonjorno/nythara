import { useState } from "react";
import { createRoot } from "react-dom/client";
import { DecisionSheet } from "../src/components/DecisionSheet";
import type { BattleState, CardDefinition } from "../src/types";
import "../src/styles.css";
import "../src/battle.css";
import { LanguageSelector } from "../src/components/LanguageSelector";
import { installDocumentLocalization } from "../src/documentLocalization";
import { usePreferencesStore } from "../src/store";
import { useEffect } from "react";

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
const pending = { id: 7, player: 0, kind: "discard_n", options: ["a", "b", "c"], n: 2 } as NonNullable<BattleState["pending"]>;

function Harness() {
  const [confirmed, setConfirmed] = useState<string[]>([]);
  const locale = usePreferencesStore((state) => state.locale);
  useEffect(() => installDocumentLocalization(locale), [locale]);
  return <>
    <LanguageSelector />
    <DecisionSheet pending={pending} cards={cards} byId={byId} busy={false} onConfirm={setConfirmed} />
    <output data-testid="confirmed">{confirmed.join(",")}</output>
  </>;
}

createRoot(document.getElementById("root")!).render(<Harness />);
