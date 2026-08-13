const TEXT_ENTRY_SELECTOR = "input, textarea, select, [contenteditable='true']";

export function releaseTextEntryFocus() {
  const active = document.activeElement;
  if (!(active instanceof HTMLElement) || !active.matches(TEXT_ENTRY_SELECTOR)) return false;
  active.blur();
  return true;
}

export function hasFinePointer() {
  return typeof window.matchMedia === "function" && window.matchMedia("(hover: hover) and (pointer: fine)").matches;
}

function focusRouteHeading() {
  if (document.querySelector('[role="dialog"][aria-modal="true"]')) return;
  const target = document.querySelector<HTMLElement>("#main-content h1, main h1");
  if (!target) return;
  const ownedTabIndex = !target.hasAttribute("tabindex");
  if (ownedTabIndex) target.tabIndex = -1;
  target.focus({ preventScroll: true });
  if (ownedTabIndex) target.addEventListener("blur", () => target.removeAttribute("tabindex"), { once: true });
}

// O teclado móvel recompõe o visual viewport de forma assíncrona depois do
// blur. Repetir o ajuste em dois frames e após a animação do teclado evita que
// a nova rota herde a altura reduzida ou a posição rolada do formulário.
export function settleRouteViewport() {
  releaseTextEntryFocus();
  let secondFrame = 0;
  const reset = () => window.scrollTo({ top: 0, left: 0, behavior: "auto" });
  const firstFrame = window.requestAnimationFrame(() => {
    secondFrame = window.requestAnimationFrame(() => {
      reset();
      focusRouteHeading();
    });
  });
  const keyboardTimer = window.setTimeout(reset, 360);
  const viewport = window.visualViewport;
  viewport?.addEventListener("resize", reset, { once: true });

  return () => {
    window.cancelAnimationFrame(firstFrame);
    window.cancelAnimationFrame(secondFrame);
    window.clearTimeout(keyboardTimer);
    viewport?.removeEventListener("resize", reset);
  };
}
