import type { Locale } from "./locales";
import { translateText } from "./i18n";

const attributes = ["aria-label", "aria-description", "placeholder", "title"] as const;
type Remembered = { source: string; rendered: string };
const textSources = new WeakMap<Text, Remembered>();
const attributeSources = new WeakMap<Element, Map<string, Remembered>>();

function localizeTextNode(node: Text, locale: Locale) {
  const current = node.nodeValue ?? "";
  const remembered = textSources.get(node);
  const source = remembered && current === remembered.rendered ? remembered.source : current;
  const rendered = translateText(source, locale);
  textSources.set(node, { source, rendered });
  if (rendered !== current) node.nodeValue = rendered;
}

function localizeElement(element: Element, locale: Locale) {
  const remembered = attributeSources.get(element) ?? new Map<string, Remembered>();
  for (const attribute of attributes) {
    const current = element.getAttribute(attribute);
    if (current === null) continue;
    const previous = remembered.get(attribute);
    const source = previous && current === previous.rendered ? previous.source : current;
    const rendered = translateText(source, locale);
    remembered.set(attribute, { source, rendered });
    if (rendered !== current) element.setAttribute(attribute, rendered);
  }
  attributeSources.set(element, remembered);
}

function localizeTree(root: Node, locale: Locale) {
  if (root.nodeType === Node.TEXT_NODE) {
    localizeTextNode(root as Text, locale);
    return;
  }
  if (root instanceof Element) localizeElement(root, locale);
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
  let node = walker.nextNode();
  while (node) {
    if (node.nodeType === Node.TEXT_NODE) localizeTextNode(node as Text, locale);
    else localizeElement(node as Element, locale);
    node = walker.nextNode();
  }
}

export function localizeDocument(locale: Locale) {
  document.documentElement.lang = locale;
  localizeTree(document.body, locale);
}

export function installDocumentLocalization(locale: Locale) {
  localizeDocument(locale);
  let applying = false;
  const observer = new MutationObserver((mutations) => {
    if (applying) return;
    applying = true;
    try {
      for (const mutation of mutations) {
        if (mutation.type === "attributes") localizeElement(mutation.target as Element, locale);
        else if (mutation.type === "characterData") localizeTextNode(mutation.target as Text, locale);
        else mutation.addedNodes.forEach((node) => localizeTree(node, locale));
      }
    } finally {
      queueMicrotask(() => { applying = false; });
    }
  });
  observer.observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: [...attributes] });
  return () => observer.disconnect();
}
