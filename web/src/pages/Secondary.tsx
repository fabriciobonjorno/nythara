import { useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../api";
import { AlphaNote } from "../components/AlphaNote";
import { ChampionEmblem } from "../components/ChampionEmblem";
import { NytharaMark } from "../components/NytharaMark";
import { UiIcon } from "../components/UiIcon";
import { VeilGlyph } from "../components/VeilGlyph";
import { LanguageSelector } from "../components/LanguageSelector";
import { buildGuidedProgress } from "../guidedTraining";
import { useActiveRulesetVersion, useCards, useChampions, useDecks, useMe, useProgress, useSeason } from "../queries";
import { usePreferencesStore, useSessionStore } from "../store";
import type { QueueResult } from "../types";

export function ResultPage() {
  const battle = useSessionStore((state) => state.lastBattle);
  const setActiveMatch = useSessionStore((state) => state.setActiveMatch);
  const guidedMatchId = useSessionStore((state) => state.guidedMatchId);
  const setGuidedMatch = useSessionStore((state) => state.setGuidedMatch);
  const navigate = useNavigate();
  const { data: deckData } = useDecks();
  const { data: cardData } = useCards();
  const { data: championData } = useChampions();
  const [practiceBusy, setPracticeBusy] = useState(false);
  const [practiceError, setPracticeError] = useState("");
  const [shareStatus, setShareStatus] = useState("");
  const cardsById = useMemo(() => new Map(cardData?.cards.map((card) => [card.id, card]) ?? []), [cardData]);
  const trainingProgress = useMemo(() => battle
    ? buildGuidedProgress(battle.events, battle.slot, cardsById)
    : { assault: false, guard: false, rite: false, completed: 0 }, [battle, cardsById]);
  const guided = Boolean(battle && guidedMatchId === battle.matchId);
  const rulesetVersion = useActiveRulesetVersion();
  const currentDeck = deckData?.decks.find((deck) => deck.ruleset_version === rulesetVersion && deck.active)
    ?? deckData?.decks.find((deck) => deck.ruleset_version === rulesetVersion);
  const playAgainNow = async (guidedNext = false) => {
    if (!currentDeck) { navigate("/decks"); return; }
    setPracticeBusy(true);
    setPracticeError("");
    try {
      const result = await api<QueueResult>("/v1/practice", { method: "POST", body: JSON.stringify({ deck_id: currentDeck.id }) });
      if (result.status !== "matched" || !result.match_id) throw new Error("O treino não abriu uma sala.");
      setActiveMatch(result.match_id);
      setGuidedMatch(guidedNext ? result.match_id : null);
      navigate(`/battle/${result.match_id}`);
    } catch (caught) {
      setPracticeError(caught instanceof Error ? caught.message : "Não foi possível iniciar outra partida.");
      setPracticeBusy(false);
    }
  };
  if (!battle) return <Missing title="Nenhum resultado nesta sessão" copy="Conclua uma partida para ver o resumo." action="Buscar duelo" to="/queue" />;
  const victory = battle.state.winner === battle.slot;
  const mine = battle.state.players[battle.slot];
  const rival = battle.state.players[1 - battle.slot];
  const cardsPlayed = battle.events.filter((event) => event.kind === "card_played");
  const damage = battle.events.filter((event) => event.kind === "damage_dealt" && event.p === 1 - battle.slot).reduce((sum, event) => sum + event.n, 0);
  const biggestHit = battle.events.filter((event) => event.kind === "damage_dealt" && event.p === 1 - battle.slot).reduce((best, event) => Math.max(best, event.n), 0);
  const vitalitySpent = battle.events.filter((event) => event.kind === "vitality_spent" && event.p === battle.slot).reduce((sum, event) => sum + event.n, 0);
  const prevented = battle.events.filter((event) => event.kind === "damage_prevented" && event.p === battle.slot).reduce((sum, event) => sum + event.n, 0);
  const myCardsPlayed = cardsPlayed.filter((event) => event.p === battle.slot).length;
  const rivalCardsPlayed = cardsPlayed.filter((event) => event.p === 1 - battle.slot).length;
  const myChampion = championData?.champions.find((champion) => champion.id === mine.champion);
  const rivalChampion = championData?.champions.find((champion) => champion.id === rival.champion);
  const resultReason = resultEndReason(battle.state.end_reason);
  const lesson = battle.state.end_reason === "timeout"
    ? "O relógio competitivo encerrou a partida. Use 1–7 para jogar rapidamente ou Espaço para passar; uma decisão simples preserva a disputa. No treino, a expiração apenas avança a janela."
    : battle.state.end_reason === "concede"
      ? "A concessão preserva tempo quando não há saída, mas a Crônica ainda mostra quais recursos poderiam ter prolongado o duelo."
      : victory
        ? biggestHit >= 6 ? `Seu golpe de ${biggestHit} foi o ponto de ruptura. Você transformou pressão em dano antes que o custo das cartas cobrasse a partida.` : `Você venceu pela soma de decisões: ${damage} de dano distribuído e ${prevented} prevenido no momento certo.`
        : vitalitySpent >= 12 ? `Você investiu ${vitalitySpent} de Vitalidade nas próprias cartas. Na próxima, preserve margem para responder à Guarda ou ao golpe decisivo.` : `O duelo ficou a ${Math.max(0, rival.vitality)} de Vitalidade. Reveja na Crônica onde uma Guarda ou um passe teria mudado o confronto.`;
  const insightTitle = battle.state.end_reason === "timeout" ? "Tempo também é recurso" : victory ? "Pressão convertida" : "Margem de Vitalidade";
  const shareResult = async () => {
    const text = `Nythara — ${victory ? "vitória" : "derrota"} em ${battle.state.round} ${battle.state.round === 1 ? "rodada" : "rodadas"}. Causei ${damage} de dano, preveni ${prevented} e meu maior golpe foi ${biggestHit}. #Nythara`;
    try {
      await copyText(text);
      setShareStatus("Resumo copiado. Nenhuma carta ou informação do rival foi incluída.");
    } catch {
      setShareStatus("O navegador bloqueou a cópia. Tente novamente após permitir acesso ao clipboard.");
    }
  };
  return <div className={`page result-page ${victory ? "victory" : "defeat"}`}>
    <section className="result-hero" aria-labelledby="result-title">
      <div className="result-orbit" aria-hidden="true"><span>{victory ? <UiIcon name="ultimate" /> : <NytharaMark />}</span></div>
      <header className="result-heading"><p className="eyebrow">CONFRONTO ENCERRADO · {resultReason.toLocaleUpperCase("pt-BR")}</p><h1 id="result-title">{victory ? "Vitória confirmada" : "Derrota na Arena"}</h1><p>{victory ? "Seu baralho sustentou a pressão até o último confronto." : "O resultado está registrado. Agora transforme a derrota em leitura para a próxima mesa."}</p></header>
      <div className="result-duelists" aria-label="Placar final do duelo">
        <article className={`result-duelist ${victory ? "is-winner" : ""}`}>
          <span className="result-duelist__emblem">{myChampion ? <ChampionEmblem id={myChampion.id} faction={myChampion.faction} /> : <UiIcon name="champion" />}</span>
          <div><small>VOCÊ</small><strong>{myChampion?.name?.split(",")[0] ?? "Seu Avatar"}</strong><em>{currentDeck?.name ?? "Baralho competitivo"}</em></div>
          <span className="result-duelist__vitality"><UiIcon name="heart" /><b>{Math.max(0, mine.vitality)}</b><small>DE {mine.max_vitality}</small><i><i style={{ width: `${Math.max(0, Math.min(100, (mine.vitality / Math.max(1, mine.max_vitality)) * 100))}%` }} /></i></span>
        </article>
        <div className="result-verdict"><span>{victory ? "VITÓRIA" : "DERROTA"}</span><UiIcon name="versus" /><small>RODADA {battle.state.round}</small></div>
        <article className={`result-duelist is-rival ${!victory ? "is-winner" : ""}`}>
          <span className="result-duelist__emblem">{rivalChampion ? <ChampionEmblem id={rivalChampion.id} faction={rivalChampion.faction} /> : <UiIcon name="champion" />}</span>
          <div><small>RIVAL</small><strong>{rivalChampion?.name?.split(",")[0] ?? "Avatar rival"}</strong><em>Baralho oculto</em></div>
          <span className="result-duelist__vitality"><UiIcon name="heart" /><b>{Math.max(0, rival.vitality)}</b><small>DE {rival.max_vitality}</small><i><i style={{ width: `${Math.max(0, Math.min(100, (rival.vitality / Math.max(1, rival.max_vitality)) * 100))}%` }} /></i></span>
        </article>
      </div>
    </section>

    <div className="result-content">
      <main className="result-analysis">
        <section className="result-metrics" aria-label="Números da partida">
          <article><span><UiIcon name="clock" /></span><small>RODADAS</small><strong>{battle.state.round}</strong><em>{resultReason}</em></article>
          <article><span><UiIcon name="deck" /></span><small>SUAS CARTAS</small><strong>{myCardsPlayed}</strong><em>rival jogou {rivalCardsPlayed}</em></article>
          <article><span><UiIcon name="duel" /></span><small>DANO CAUSADO</small><strong>{damage}</strong><em>maior golpe {biggestHit}</em></article>
          <article><span><UiIcon name="ward" /></span><small>PREVENIDO</small><strong>{prevented}</strong><em>{prevented ? "defesas efetivas" : "nenhum dano barrado"}</em></article>
        </section>
        <section className="result-insight"><span><UiIcon name="balance" /></span><div><p className="eyebrow">LEITURA TÁTICA</p><h2>{insightTitle}</h2><p>{lesson}</p><div className="result-insight__facts"><span><b>{vitalitySpent}</b> Vitalidade investida</span><span><b>{damage}</b> dano confirmado</span><span><b>{cardsPlayed.length}</b> cartas na mesa</span></div></div></section>
        {guided && <section className="guided-result" aria-label="Fundamentos praticados"><header><span><UiIcon name="guide" /></span><div><p className="eyebrow">TREINO GUIADO</p><h2>{trainingProgress.completed}/3 fundamentos praticados</h2><small>Itens vêm dos eventos confirmados desta partida.</small></div></header><div>{([["Assalto", "Declarou uma carta no centro", trainingProgress.assault], ["Guarda", "Respondeu a um ataque rival", trainingProgress.guard], ["Rito", "Usou um efeito antes de encerrar", trainingProgress.rite]] as const).map(([title, copy, done]) => <article className={done ? "is-done" : ""} key={title}><i>{done ? "✓" : "—"}</i><span><strong>{title}</strong><small>{done ? copy : `${copy} · pratique na próxima partida`}</small></span></article>)}</div></section>}
      </main>

      <aside className="result-command" aria-label="Próximos passos">
        <header><p className="eyebrow">SUA PRÓXIMA DECISÃO</p><h2>{victory ? "Mantenha o ritmo" : "Volte mais preparado"}</h2><p>Jogue novamente agora ou estude a partida antes de retornar.</p></header>
        <button className="primary-button result-rematch" type="button" disabled={practiceBusy} onClick={() => playAgainNow(guided)}><UiIcon name="duel" />{practiceBusy ? "Abrindo a mesa…" : guided ? "Repetir treino guiado" : "Jogar contra o bot"}</button>
        <Link className="secondary-button result-queue" to="/queue"><UiIcon name="versus" /> Buscar rival</Link>
        {practiceError && <p className="form-error" role="alert">{practiceError}</p>}
        <div className="result-review"><p className="eyebrow">ESTUDAR O DUELO</p><Link to={`/replay/${battle.matchId}`}><span><UiIcon name="history" /></span><div><strong>Replay visual</strong><small>Reveja cartas, confrontos e dano evento por evento.</small></div><UiIcon name="arrow-right" /></Link><Link to={`/cronica/${battle.matchId}`}><span><UiIcon name="balance" /></span><div><strong>Crônica tática</strong><small>Leia a curva de Vitalidade e os momentos decisivos.</small></div><UiIcon name="arrow-right" /></Link></div>
        <footer className="result-actions"><button type="button" onClick={shareResult}><UiIcon name="fragment" /> Copiar resultado</button><Link to="/app"><UiIcon name="home" /> Início</Link></footer>
        {shareStatus && <p className="share-status" role="status">{shareStatus}</p>}
        <AlphaNote matchId={battle.matchId} />
      </aside>
    </div>
  </div>;
}

const resultEndReason = (reason?: string) => ({
  timeout: "tempo esgotado", concede: "concessão", concessao: "concessão",
  vitality: "Vitalidade zerada", vitalidade: "Vitalidade zerada",
  duplo_nocaute: "nocaute duplo", pressao_de_nythara: "Pressão de Nythara",
}[reason ?? ""] ?? reason?.replaceAll("_", " ") ?? "partida encerrada");

async function copyText(text: string) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const field = document.createElement("textarea");
  field.value = text;
  field.setAttribute("readonly", "");
  field.style.position = "fixed";
  field.style.opacity = "0";
  document.body.appendChild(field);
  field.select();
  const copied = document.execCommand("copy");
  field.remove();
  if (!copied) throw new Error("clipboard_unavailable");
}

export function ProfilePage() {
  const rulesetVersion = useActiveRulesetVersion();
  const { data: me } = useMe();
  const { data: season } = useSeason();
  const { data: decks } = useDecks();
	const { data: progress } = useProgress();
	return <div className="page profile-page"><section className="profile-hero"><div className="profile-avatar">{me?.display_name?.slice(0, 1).toUpperCase() ?? "V"}</div><div><p className="eyebrow">PERFIL DO DUELISTA</p><h1>{me?.display_name ?? "Viajante"}</h1><p>Participante do Alpha · {me?.role === "admin" ? "Guardião" : "Jogador"}</p></div><span className="season-seal"><NytharaMark /><small>{season?.name ?? "Alpha"}</small></span></section><section className="profile-grid"><article className="rank-card"><p className="eyebrow">NÍVEL DA CONTA</p><h2>Nível {progress?.account.level ?? 1}</h2><div className="rank-emblem"><UiIcon name="mastery" /></div><strong>{progress?.account.level_xp_required ? `${progress.account.level_xp}/${progress.account.level_xp_required} XP` : "Nível máximo"}</strong><p>Suba no PvP contra jogadores para liberar Lendárias. Treinos e bots não concedem XP.</p></article><article className="panel"><h2>Conta competitiva</h2><dl className="profile-details"><div><dt>Ruleset</dt><dd>{season?.ruleset_version ?? rulesetVersion}</dd></div><div><dt>Baralho atual</dt><dd>{decks?.decks.some((deck) => deck.ruleset_version === rulesetVersion) ? "Pronto" : "Pendente"}</dd></div><div><dt>Catálogo</dt><dd>130 cartas · Lendárias por nível</dd></div><div><dt>Temporada</dt><dd>{season?.name ?? "Alpha"}</dd></div></dl></article></section></div>;
}

const tutorialSteps = [
  { phase: "Começar", navLabel: "Começo", icon: "journey", title: "O caminho até o primeiro duelo", body: "Abra Baralho, escolha uma aparência, salve 30 cartas e entre em Jogar. Treino e ranqueada usam a mesma regra; só muda quem ocupa o outro lado da mesa.", tip: "Se uma partida estiver ativa, o botão Retomar partida aparece no início." },
  { phase: "Aparência", navLabel: "Avatar", icon: "champion", title: "Escolha o Avatar pelo poder", body: "Cada um dos dez traz um poder próprio — curar ao acertar, ganhar Ward ao apanhar, pagar menos quando a Vitalidade cai. Todos partem da mesma Vitalidade e do mesmo baralho de 30 cartas; o poder muda como você joga, não quanto você tem.", tip: "Os poderes são medidos em simulação: nenhum passa de alguns pontos de vantagem sobre outro." },
  { phase: "Preparar", navLabel: "30 cartas", icon: "deck", title: "Monte um baralho legal", body: "Use exatamente 30 cartas, com no mínimo 8 Assaltos, 10 Guardas e 4 Ritos. São permitidas até 2 cópias da mesma carta e apenas 1 de cada Lendária.", tip: "Montar 10 / 10 / 10 cria uma base equilibrada; depois você pode personalizar os oito espaços livres." },
  { phase: "Preparar", navLabel: "Salvar", icon: "validation", title: "Salve uma escolha que importa", body: "O servidor valida composição, cópias e posse. Depois de salvar, o baralho competitivo fica protegido por 24 horas e continua disponível para jogar durante esse período.", tip: "O contador e o botão de salvar permanecem visíveis na lateral do construtor." },
  { phase: "Duelar", navLabel: "Treino", icon: "duel", title: "Treine ou encontre um rival", body: "Treinar contra o bot abre uma partida imediatamente. Buscar oponente entra na fila 1v1; quando houver par, os dois clientes recebem somente a informação que podem ver.", tip: "A mão rival aparece como versos de carta. O servidor nunca envia suas identidades ao oponente." },
  { phase: "Mesa", navLabel: "Compra", icon: "mulligan", title: "Compre e leia sua mão", body: "Você começa com cinco cartas e compra no início de cada turno. Sua mão comporta sete; compras acima do limite são queimadas. Se o baralho acabar, a Fadiga cobra Vitalidade crescente.", tip: "Cartas que podem ser jogadas ficam iluminadas. O custo sempre é pago com Vitalidade e nunca pode deixá-lo abaixo de 1." },
  { phase: "Mesa", navLabel: "Assalto", icon: "duel", title: "Envie um Assalto ao centro", body: "Na fase de Assalto, escolha uma carta ofensiva ou passe. Ela viaja até a zona central e revela seu Poder. O primeiro Assalto da partida recebe a compensação de iniciativa indicada pela regra.", tip: "Observe o Poder no centro antes de decidir quanto risco aceita no turno." },
  { phase: "Mesa", navLabel: "Guarda", icon: "guard", title: "Defenda ou aceite o golpe", body: "O defensor pode responder com uma Guarda. Poder menos Prevenção é o dano final: se o dano passar, a Guarda se estilhaça; se for bloqueado, o Assalto se estilhaça. Sem Guarda, o golpe é direto.", tip: "Passar pode preservar uma Guarda forte, mas a Vitalidade perdida não volta sozinha." },
  { phase: "Mesa", navLabel: "Rito", icon: "rite", title: "Use um Rito e encerre", body: "Depois do confronto, o atacante pode jogar até um Rito legal ou encerrar o turno. Ritos curam, compram cartas ou aplicam efeitos compatíveis com o Modo Confronto.", tip: "Relíquias e Manifestações permanecem no arquivo e não entram no baralho competitivo atual." },
  { phase: "Vencer", navLabel: "Final", icon: "decision", title: "Leve a Vitalidade rival a zero", body: "O duelo termina quando um lado chega a zero. Se a partida se arrastar até o fim do relógio, a Pressão de Nythara passa a reduzir a Vitalidade dos dois lados em escala crescente até haver um vencedor — a mesa avisa quando esse turno se aproxima.", tip: "A Arena mostra dano, descarte, Fadiga e cada carta estilhaçada; o histórico preserva o resultado autoritativo." },
] as const;

export function TutorialPage() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const { data: deckData } = useDecks();
  const setActiveMatch = useSessionStore((state) => state.setActiveMatch);
  const setGuidedMatch = useSessionStore((state) => state.setGuidedMatch);
  const [guidedBusy, setGuidedBusy] = useState(false);
  const [guidedError, setGuidedError] = useState("");
  const requestedStep = Number(params.get("step"));
  const step = Number.isInteger(requestedStep) && requestedStep >= 0 && requestedStep < tutorialSteps.length ? requestedStep : 0;
  const restartOnboarding = usePreferencesStore((state) => state.restartOnboarding);
  const rulesetVersion = useActiveRulesetVersion();
  const currentDeck = deckData?.decks.find((deck) => deck.ruleset_version === rulesetVersion && deck.active)
    ?? deckData?.decks.find((deck) => deck.ruleset_version === rulesetVersion);
  const startGuidedPractice = async () => {
    if (!currentDeck) { navigate("/decks"); return; }
    setGuidedBusy(true);
    setGuidedError("");
    try {
      const result = await api<QueueResult>("/v1/practice", { method: "POST", body: JSON.stringify({ deck_id: currentDeck.id }) });
      if (result.status !== "matched" || !result.match_id) throw new Error("O treino não abriu uma sala.");
      setActiveMatch(result.match_id);
      setGuidedMatch(result.match_id);
      navigate(`/battle/${result.match_id}`);
    } catch (caught) {
      setGuidedError(caught instanceof Error ? caught.message : "Não foi possível iniciar o treino guiado.");
      setGuidedBusy(false);
    }
  };
  const setStep = (next: number) => setParams(next ? { step: String(next) } : {}, { replace: true });
  const current = tutorialSteps[step];
  return <div className="page tutorial-page"><header><p className="eyebrow">GUIA COMPLETO · {step + 1}/{tutorialSteps.length}</p><h1>Da escolha ao duelo</h1><p>Aprenda o fluxo do app e as decisões essenciais da mesa.</p><div className="tutorial-entry-actions"><button className="primary-button" type="button" disabled={guidedBusy} onClick={startGuidedPractice}>{guidedBusy ? "Preparando a mesa…" : "Iniciar treino guiado"}</button><button className="ghost-button tutorial-restart" type="button" onClick={restartOnboarding}>Reabrir introdução rápida</button></div>{guidedError && <p className="form-error" role="alert">{guidedError}</p>}<div className="tutorial-progress" aria-hidden="true">{tutorialSteps.map((item, index) => <span className={index === step ? "is-current" : index < step ? "is-complete" : ""} key={item.title} />)}</div></header><nav className="tutorial-phases" aria-label="Etapas do guia">{tutorialSteps.map((item, index) => <button type="button" aria-label={`Etapa ${index + 1}: ${item.title}`} aria-current={index === step ? "step" : undefined} className={index === step ? "is-current" : index < step ? "is-complete" : ""} onClick={() => setStep(index)} key={item.title}><span>{String(index + 1).padStart(2, "0")}</span>{item.navLabel}</button>)}</nav><section className="tutorial-card"><div className="tutorial-illustration" aria-hidden="true"><VeilGlyph variant={current.icon} /><small>{current.phase}</small></div><div><p className="eyebrow">{current.phase} · ETAPA {step + 1}</p><h2>{current.title}</h2><p>{current.body}</p><div className="tutorial-tip"><strong>Como usar</strong><span>{current.tip}</span></div></div></section><footer><button className="ghost-button" type="button" onClick={() => setStep(Math.max(0, step - 1))} disabled={step === 0}><UiIcon name="arrow-left" />Voltar</button>{step < tutorialSteps.length - 1 ? <button className="primary-button" type="button" onClick={() => setStep(step + 1)}>Continuar<UiIcon name="arrow-right" /></button> : <Link className="primary-button" to="/decks">Montar meu deck</Link>}</footer></div>;
}

export function SettingsPage() {
  const rulesetVersion = useActiveRulesetVersion();
  const preferences = usePreferencesStore();
  const options = useMemo(() => [
    ["sound", "Som", "Efeitos sonoros da interface e batalha."],
    ["ambience", "Trilha ambiente", "Paisagem sonora discreta que reage ao turno e ao perigo; começa após sua primeira ação."],
    ["haptics", "Resposta tátil", "Vibrações curtas em confrontos e golpes, quando o dispositivo permitir."],
    ["combatHints", "Dicas durante o duelo", "Mostra custo, Vitalidade restante e orientação da fase atual."],
    ["reducedMotion", "Reduzir movimento", "Desativa rotações, pulsos e transições decorativas."],
    ["highContrast", "Alto contraste", "Reforça bordas e contraste dos controles."],
    ["largeText", "Texto ampliado", "Aumenta o tamanho-base da interface."],
  ] as const, []);
  return <div className="page settings-page"><header className="page-header"><div><p className="eyebrow">PREFERÊNCIAS LOCAIS</p><h1>Configurações</h1><p>Ajustes ficam somente neste dispositivo.</p></div></header><section className="settings-panel"><h2>Idioma</h2><div className="setting-select"><span><strong>Idioma</strong><small>Português do Brasil, Español ou English. A troca é imediata.</small></span><LanguageSelector /></div><h2>Acessibilidade e experiência</h2><label className="setting-select"><span><strong>Ritmo das animações</strong><small>Cinemático deixa voo, comparação e estilhaço mais fáceis de acompanhar.</small></span><select value={preferences.animationPace} onChange={(event) => preferences.setAnimationPace(event.target.value as "cinematic" | "normal" | "quick")}><option value="cinematic">Cinemático · mais legível</option><option value="normal">Normal</option><option value="quick">Rápido</option></select></label>{options.map(([key, title, copy]) => <label className="setting-row" key={key}><span><strong>{title}</strong><small>{copy}</small></span><input type="checkbox" checked={preferences[key]} onChange={(event) => preferences.set(key, event.target.checked)} /><i aria-hidden="true" /></label>)}</section><section className="settings-panel"><h2>Sobre o Alpha</h2><dl className="profile-details"><div><dt>Cliente</dt><dd>Modo Confronto</dd></div><div><dt>Ruleset</dt><dd>{rulesetVersion}</dd></div><div><dt>Privacidade</dt><dd>Tokens na sessão</dd></div><div><dt>Conteúdo</dt><dd>IP original</dd></div></dl></section></div>;
}

export function Missing({ title = "Caminho não encontrado", copy = "Esta área não existe ou foi movida.", action = "Ir ao início", to = "/app" }: { title?: string; copy?: string; action?: string; to?: string }) { return <div className="missing-page"><NytharaMark /><h1>{title}</h1><p>{copy}</p><Link className="primary-button" to={to}>{action}</Link></div>; }
