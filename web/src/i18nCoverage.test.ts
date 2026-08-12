import { readdirSync, readFileSync } from "node:fs";
import { extname, join } from "node:path";
import ts from "typescript";
import { describe, expect, it } from "vitest";
import { translateText } from "./i18n";

const sourceRoot = join(process.cwd(), "src");
const portugueseSignals = /[ãõç]|\b(?:abrir|abrindo|agora|ainda|algum|antes|após|assalto|baralho|baralhos|busca|buscar|carregado|carta|cartas|com|confirmar|confronto|conjunto|conta|criado|dano|depois|duelo|encerrar|escolha|está|fase|fila|fluxo|guarda|iniciar|jogador|jogadores|jogada|jogou|lista|mesa|mesmo|melhor|navegar|nível|níveis|não|nenhum|oponente|ordem|pareamento|partida|para|pontos|próxima|próximo|rascunho|registro|regras|repetição|reproduzir|rito|salvar|sem|senha|sessão|seu|seus|simular|somente|sua|suas|tempo|temporada|treino|turno|uma|validar|versão|vitalidade|você|vocês|voltar)\b/i;
const portugueseOnlySignals = /[ãõç]|\b(?:agora|ainda|assalto|avançou|baralho|baralhos|carregado|confronto|criado|depois|encerrar|escolha|jogador|jogadores|jogou|mão|nível|níveis|não|pareamento|rascunho|regras|reproduzir|salvar|senha|seu|seus|somente|sua|suas|sofreu|treino|uma|você|vocês|voltar)\b/i;
const englishLeakSignals = /\b(?:authoritative|clipboard|decks?|drafts?|headless|language|liveops|log|matchmaking|nickname|pipeline|play|ranked|replay|rulesets?|schema|tokens?)\b|\bbots?(?!\p{L})/iu;
const visibleAttributes = new Set(["alt", "aria-description", "aria-label", "placeholder", "title"]);
const ignoredFiles = new Set(["api.ts", "locales.ts", "seo.ts", "types.ts"]);
const invariantCopy = new Set(["Guarda do Limiar", "Carta de Guarda do Limiar"]);

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return sourceFiles(path);
    if (![".ts", ".tsx"].includes(extname(entry.name)) || entry.name.includes(".test.") || entry.name === "i18n.ts" || ignoredFiles.has(entry.name)) return [];
    return [path];
  });
}

function normalize(value: string) {
  return value.replace(/\s+/g, " ").trim();
}

function collectCandidates(path: string) {
  const source = readFileSync(path, "utf8");
  const file = ts.createSourceFile(path, source, ts.ScriptTarget.Latest, true, path.endsWith("x") ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
  const values: Array<{ value: string; line: number }> = [];
  const add = (value: string, position: number) => {
    const normalized = normalize(value);
    if (normalized.length >= 3 && !normalized.startsWith("{") && !invariantCopy.has(normalized)) values.push({ value: normalized, line: file.getLineAndCharacterOfPosition(position).line + 1 });
  };
  const hiddenAttribute = (node: ts.Node) => {
    let parent = node.parent;
    while (parent && !ts.isSourceFile(parent)) {
      if (ts.isJsxAttribute(parent)) return !visibleAttributes.has(parent.name.getText(file));
      parent = parent.parent;
    }
    return false;
  };
  const expressionVariants = (node: ts.Expression): string[] => {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) return [node.text];
    if (ts.isTemplateExpression(node)) return templateVariants(node);
    if (ts.isConditionalExpression(node)) return [
      ...expressionVariants(node.whenTrue),
      ...expressionVariants(node.whenFalse),
    ];
    return ["2"];
  };
  const templateVariants = (node: ts.TemplateExpression): string[] => node.templateSpans.reduce(
    (values, span) => values.flatMap((value) => expressionVariants(span.expression).map((expression) => `${value}${expression}${span.literal.text}`)),
    [node.head.text],
  );
  const visit = (node: ts.Node) => {
    if (ts.isJsxText(node)) add(node.text, node.getStart(file));
    else if (ts.isTemplateExpression(node) && !hiddenAttribute(node)) templateVariants(node).forEach((value) => add(value, node.getStart(file)));
    else if ((ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) && !hiddenAttribute(node)) add(node.text, node.getStart(file));
    ts.forEachChild(node, visit);
  };
  visit(file);
  return values;
}

describe("cobertura integral de idiomas", () => {
  it("não deixa texto em português aparecer quando o idioma é inglês", () => {
    const missing = sourceFiles(sourceRoot).flatMap((path) => collectCandidates(path).filter(({ value }) => portugueseSignals.test(value)).flatMap(({ value, line }) => {
      const translated = translateText(value, "en");
      return translated === value ? [`src/${path.slice(sourceRoot.length + 1)}:${line} → ${JSON.stringify(value)}`] : [];
    }));

    expect(missing).toEqual([]);
  });

  it("não mantém fragmentos exclusivos do português em espanhol ou inglês", () => {
    const mixed = sourceFiles(sourceRoot).flatMap((path) => collectCandidates(path).filter(({ value }) => portugueseSignals.test(value)).flatMap(({ value, line }) => {
      return (["es", "en"] as const).flatMap((locale) => portugueseOnlySignals.test(translateText(value, locale))
        ? [`${locale} · src/${path.slice(sourceRoot.length + 1)}:${line} → ${JSON.stringify(value)}`]
        : []);
    }));

    expect(mixed).toEqual([]);
  });

  it("não mantém termos de interface em inglês no texto-fonte em português", () => {
    const leaked = sourceFiles(sourceRoot).flatMap((path) => collectCandidates(path).flatMap(({ value, line }) => {
      const looksVisible = !/[\/_{}]/.test(value) && (/\s/.test(value) || /^\p{Lu}/u.test(value));
      return looksVisible && englishLeakSignals.test(value)
        ? [`src/${path.slice(sourceRoot.length + 1)}:${line} → ${JSON.stringify(value)}`]
        : [];
    }));

    expect(leaked).toEqual([]);
  });
});
