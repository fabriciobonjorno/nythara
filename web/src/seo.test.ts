import { afterEach, describe, expect, it } from "vitest";
import { applyRouteMetadata } from "./seo";

function addMeta(attribute: "name" | "property", key: string) {
  const meta = document.createElement("meta");
  meta.setAttribute(attribute, key);
  document.head.append(meta);
}

afterEach(() => {
  document.head.querySelectorAll("meta").forEach((meta) => meta.remove());
  document.title = "";
});

describe("metadados de busca", () => {
  it("mantém somente a entrada pública indexável", () => {
    addMeta("name", "description");
    addMeta("name", "robots");
    applyRouteMetadata("/", "", "pt-BR");
    expect(document.title).toBe("Nythara — Jogo de cartas online PvP");
    expect(document.querySelector('meta[name="description"]')?.getAttribute("content")).toContain("30 cartas");
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute("content", expect.stringContaining("index, follow"));

    applyRouteMetadata("/profile", "", "pt-BR");
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute("content", "noindex, nofollow, noarchive");
  });

  it("evita indexar convites e localiza o título público", () => {
    addMeta("name", "robots");
    applyRouteMetadata("/", "?admin_invite=opaco", "pt-BR");
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute("content", "noindex, nofollow, noarchive");

    applyRouteMetadata("/", "", "en");
    expect(document.title).toBe("Nythara — Online PvP card game");
  });
});
