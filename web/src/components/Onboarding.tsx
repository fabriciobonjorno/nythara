import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api";
import { useActiveRulesetVersion, useDecks } from "../queries";
import { usePreferencesStore, useSessionStore } from "../store";
import type { QueueResult } from "../types";
import { UiIcon } from "./UiIcon";
import { VeilGlyph } from "./VeilGlyph";

const introSteps = [
  {
    icon: "veil",
    eyebrow: "BEM-VINDO A NYTHARA",
    title: "Um objetivo. Trinta cartas.",
    body: "Reduza a Vitalidade rival a zero. Sua própria Vitalidade também paga as cartas, então toda jogada forte deixa uma abertura.",
    points: ["O primeiro duelo começa em poucos segundos", "A engine mostra somente jogadas válidas", "Cada Avatar tem um poder próprio, todos equilibrados"],
  },
  {
    icon: "duel",
    eyebrow: "A RODADA EM UMA IMAGEM",
    title: "Assalto. Guarda. Rito.",
    body: "Ataque no centro. O rival bloqueia ou aceita o dano. Depois, um Rito pode curar, comprar ou mudar o próximo confronto.",
    points: ["Poder menos Prevenção vira dano", "A carta perdedora se estilhaça", "Espaço passa a fase; 1–7 joga da mão"],
  },
  {
    icon: "journey",
    eyebrow: "APRENDA JOGANDO",
    title: "A mesa guia você",
    body: "Seu baralho inicial já está pronto. No treino, cartas válidas acendem e a prévia mostra custo, Vitalidade restante e o resultado base antes da decisão.",
    points: ["Treino usa a mesma regra do PvP", "Dicas podem ser ocultadas a qualquer momento", "Som, vibração e movimento são configuráveis"],
  },
] as const;

export function Onboarding() {
  const userId = useSessionStore((state) => state.user?.id ?? "");
  const completedFor = usePreferencesStore((state) => state.onboardingUserId);
  const completed = Boolean(userId && completedFor === userId);
  const complete = usePreferencesStore((state) => state.completeOnboarding);
  const setActiveMatch = useSessionStore((state) => state.setActiveMatch);
  const setGuidedMatch = useSessionStore((state) => state.setGuidedMatch);
  const navigate = useNavigate();
  const { data: decks } = useDecks();
  const [step, setStep] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const titleRef = useRef<HTMLHeadingElement>(null);

  const finish = (destination?: string) => {
    complete(userId);
    if (destination) navigate(destination);
  };
  const rulesetVersion = useActiveRulesetVersion();
  const currentDeck = decks?.decks.find((deck) => deck.ruleset_version === rulesetVersion && deck.active)
    ?? decks?.decks.find((deck) => deck.ruleset_version === rulesetVersion);
  const startPractice = async () => {
    if (!currentDeck) { finish("/decks"); return; }
    setBusy(true);
    setError("");
    try {
      const result = await api<QueueResult>("/v1/practice", { method: "POST", body: JSON.stringify({ deck_id: currentDeck.id }) });
      if (result.status !== "matched" || !result.match_id) throw new Error("O treino não abriu uma sala.");
      complete(userId);
      setActiveMatch(result.match_id);
      setGuidedMatch(result.match_id);
      navigate(`/battle/${result.match_id}`);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Não foi possível iniciar o treino.");
      setBusy(false);
    }
  };
  useEffect(() => {
    if (completed) return;
    titleRef.current?.focus();
    const closeOnEscape = (event: KeyboardEvent) => { if (event.key === "Escape") complete(userId); };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [complete, completed, step, userId]);

  if (completed) return null;
  const current = introSteps[step];

  return <div className="onboarding-backdrop" role="presentation">
    <section className="onboarding-dialog" role="dialog" aria-modal="true" aria-labelledby="onboarding-title">
      <button className="onboarding-skip" type="button" onClick={() => finish()}>Pular por agora</button>
      <div className="onboarding-visual" aria-hidden="true">{step === 1 ? <OnboardingDuelDemo /> : <VeilGlyph variant={current.icon} />}<small>{step + 1} / {introSteps.length}</small></div>
      <div className="onboarding-copy">
        <p className="eyebrow">{current.eyebrow}</p>
        <h2 id="onboarding-title" ref={titleRef} tabIndex={-1}>{current.title}</h2>
        <p>{current.body}</p>
        <ul>{current.points.map((point) => <li key={point}><UiIcon name="check" />{point}</li>)}</ul>
        <div className="onboarding-progress" aria-label={`Etapa ${step + 1} de ${introSteps.length}`}>
          {introSteps.map((item, index) => <button type="button" aria-label={`Ir para ${item.title}`} aria-current={index === step ? "step" : undefined} className={index === step ? "is-current" : index < step ? "is-complete" : ""} onClick={() => setStep(index)} key={item.title} />)}
        </div>
        <footer>
          <button className="ghost-button" type="button" onClick={() => setStep(Math.max(0, step - 1))} disabled={step === 0}><UiIcon name="arrow-left" />Voltar</button>
          {step < introSteps.length - 1
            ? <button className="primary-button" type="button" onClick={() => setStep(step + 1)}>Continuar<UiIcon name="arrow-right" /></button>
            : <div className="onboarding-final-actions"><button className="primary-button" type="button" disabled={busy} onClick={startPractice}>{busy ? "Abrindo a mesa…" : currentDeck ? "Treinar agora" : "Montar meu baralho"}</button><button className="ghost-button" type="button" onClick={() => finish("/decks")}>Revisar baralho</button></div>}
        </footer>
        {error && <p className="form-error" role="alert">{error}</p>}
      </div>
    </section>
  </div>;
}

function OnboardingDuelDemo() {
  return <div className="onboarding-duel-demo">
    <article className="is-assault"><span>ASSALTO</span><strong>7</strong><small>PODER</small></article>
    <div><b>7</b><i>−</i><b>4</b><strong>3 DANO</strong></div>
    <article className="is-guard"><span>GUARDA</span><strong>4</strong><small>PREVENÇÃO</small></article>
    <em>ASSALTO → GUARDA → RITO</em>
  </div>;
}
