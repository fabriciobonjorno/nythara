import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { AlphaNote } from "../components/AlphaNote";
import { ChampionEmblem } from "../components/ChampionEmblem";
import { NytharaMark } from "../components/NytharaMark";
import { UiIcon } from "../components/UiIcon";
import { VeilGlyph } from "../components/VeilGlyph";
import { LanguageSelector } from "../components/LanguageSelector";
import { buildGuidedProgress } from "../guidedTraining";
import { useActiveRulesetVersion, useCards, useChampions, useDecks, useMatchReplay, useMe, useProgress, useSeason } from "../queries";
import { TUTORIAL_STEP_IDS, type TutorialStepId, usePreferencesStore, useSessionStore } from "../store";
import type { QueueResult } from "../types";

export function ResultPage() {
  const battle = useSessionStore((state) => state.lastBattle);
  const setActiveMatch = useSessionStore((state) => state.setActiveMatch);
  const guidedMatchId = useSessionStore((state) => state.guidedMatchId);
  const setGuidedMatch = useSessionStore((state) => state.setGuidedMatch);
  const userId = useSessionStore((state) => state.user?.id ?? "");
  const completeTutorialStep = usePreferencesStore((state) => state.completeTutorialStep);
  const navigate = useNavigate();
  const { data: deckData } = useDecks();
  const { data: cardData } = useCards();
  const { data: championData } = useChampions();
	const { data: me } = useMe();
	const { data: replay } = useMatchReplay(battle?.matchId ?? "");
  const [practiceBusy, setPracticeBusy] = useState(false);
  const [practiceError, setPracticeError] = useState("");
  const [shareStatus, setShareStatus] = useState("");
  const cardsById = useMemo(() => new Map(cardData?.cards.map((card) => [card.id, card]) ?? []), [cardData]);
  const trainingProgress = useMemo(() => battle
    ? buildGuidedProgress(battle.events, battle.slot, cardsById)
    : { assault: false, guard: false, rite: false, completed: 0 }, [battle, cardsById]);
  const guided = Boolean(battle && guidedMatchId === battle.matchId);
  useEffect(() => {
    if (guided && userId) completeTutorialStep(userId, "training");
  }, [completeTutorialStep, guided, userId]);
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
	const myNickname = replay?.players[battle.slot]?.display_name ?? me?.display_name ?? "Você";
	const rivalNickname = replay?.players[1 - battle.slot]?.display_name ?? "Rival";
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
		  <div><small>VOCÊ</small><strong>{myNickname}</strong><em>{myChampion?.name?.split(",")[0] ?? "Seu Avatar"} · {currentDeck?.name ?? "Baralho competitivo"}</em></div>
          <span className="result-duelist__vitality"><UiIcon name="heart" /><b>{Math.max(0, mine.vitality)}</b><small>DE {mine.max_vitality}</small><i><i style={{ width: `${Math.max(0, Math.min(100, (mine.vitality / Math.max(1, mine.max_vitality)) * 100))}%` }} /></i></span>
        </article>
        <div className="result-verdict"><span>{victory ? "VITÓRIA" : "DERROTA"}</span><UiIcon name="versus" /><small>RODADA {battle.state.round}</small></div>
        <article className={`result-duelist is-rival ${!victory ? "is-winner" : ""}`}>
          <span className="result-duelist__emblem">{rivalChampion ? <ChampionEmblem id={rivalChampion.id} faction={rivalChampion.faction} /> : <UiIcon name="champion" />}</span>
		  <div><small>RIVAL</small><strong>{rivalNickname}</strong><em>{rivalChampion?.name?.split(",")[0] ?? "Avatar rival"} · Baralho oculto</em></div>
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
        {guided && <Link className="primary-button tutorial-result-continue" to="/tutorial?step=5"><UiIcon name="guide" /> Continuar tutorial</Link>}
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
	const { data: champions } = useChampions();
  const { data: season } = useSeason();
  const { data: decks } = useDecks();
	const { data: progress } = useProgress();
	const queryClient = useQueryClient();
	const setPrincipal = useSessionStore((state) => state.setPrincipal);
	const clear = useSessionStore((state) => state.clear);
	const [avatarID, setAvatarID] = useState("");
	const [profileBusy, setProfileBusy] = useState(false);
	const [profileStatus, setProfileStatus] = useState("");
	const [profileError, setProfileError] = useState("");
	const [currentPassword, setCurrentPassword] = useState("");
	const [newPassword, setNewPassword] = useState("");
	const [confirmPassword, setConfirmPassword] = useState("");
	const [passwordBusy, setPasswordBusy] = useState(false);
	const [passwordError, setPasswordError] = useState("");
	const chosenAvatarID = avatarID || me?.avatar_id || "";
	const chosenAvatar = champions?.champions.find((champion) => champion.id === chosenAvatarID);
	const saveAvatar = async () => {
		if (!chosenAvatarID) return;
		setProfileBusy(true); setProfileError(""); setProfileStatus("");
		try {
			const updated = await api<NonNullable<typeof me>>("/v1/me/profile", { method: "PUT", body: JSON.stringify({ avatar_id: chosenAvatarID }) });
			setPrincipal(updated); queryClient.setQueryData(["me"], updated); setProfileStatus("Imagem de perfil atualizada em todo o jogo.");
		} catch (caught) { setProfileError(caught instanceof Error ? caught.message : "Não foi possível salvar a imagem."); }
		finally { setProfileBusy(false); }
	};
	const changePassword = async (event: FormEvent) => {
		event.preventDefault(); setPasswordError("");
		if (newPassword !== confirmPassword) { setPasswordError("A confirmação não corresponde à nova senha."); return; }
		setPasswordBusy(true);
		try {
			await api<void>("/v1/me/password", { method: "PUT", body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }) });
			sessionStorage.setItem("nythara-password-changed", "1"); clear();
		} catch (caught) { setPasswordError(caught instanceof Error ? caught.message : "Não foi possível trocar a senha."); setPasswordBusy(false); }
	};
	return <div className="page profile-page">
		<section className="profile-hero"><div className="profile-avatar">{chosenAvatar ? <ChampionEmblem id={chosenAvatar.id} faction={chosenAvatar.faction} /> : me?.display_name?.slice(0, 1).toUpperCase() ?? "V"}</div><div><p className="eyebrow">PERFIL DO DUELISTA</p><h1>{me?.display_name ?? "Viajante"}</h1><p>Este é o nickname mostrado na Arena · {me?.role === "player" ? "Jogador" : me?.role === "owner" ? "Proprietário" : "Guardião"}</p></div><span className="season-seal"><NytharaMark /><small>{season?.name ?? "Alpha"}</small></span></section>
		<section className="profile-grid"><article className="rank-card"><p className="eyebrow">NÍVEL DA CONTA</p><h2>Nível {progress?.account.level ?? 1}</h2><div className="rank-emblem"><UiIcon name="mastery" /></div><strong>{progress?.account.level_xp_required ? `${progress.account.level_xp}/${progress.account.level_xp_required} XP` : "Nível máximo"}</strong><p>Suba no PvP contra jogadores para liberar Lendárias. Treinos e bots não concedem XP.</p></article><article className="panel"><h2>Conta competitiva</h2><dl className="profile-details"><div><dt>Nickname público</dt><dd>{me?.display_name ?? "—"}</dd></div><div><dt>Ruleset</dt><dd>{season?.ruleset_version ?? rulesetVersion}</dd></div><div><dt>Baralho atual</dt><dd>{decks?.decks.some((deck) => deck.ruleset_version === rulesetVersion) ? "Pronto" : "Pendente"}</dd></div><div><dt>Temporada</dt><dd>{season?.name ?? "Alpha"}</dd></div></dl></article></section>
		<section className="profile-settings-grid">
		  <article className="panel profile-editor"><header><p className="eyebrow">SUA IDENTIDADE VISUAL</p><h2>Imagem de perfil</h2><p>Escolha um emblema original. Ele aparece no topo e acompanha sua identidade, sem alterar o Avatar do baralho.</p></header><div className="profile-avatar-options">{champions?.champions.map((champion) => <button type="button" className={chosenAvatarID === champion.id ? "is-selected" : ""} aria-pressed={chosenAvatarID === champion.id} aria-label={`Usar ${champion.name} como imagem de perfil`} onClick={() => { setAvatarID(champion.id); setProfileStatus(""); }} key={champion.id}><ChampionEmblem id={champion.id} faction={champion.faction} /><span>{champion.name.split(",")[0]}</span></button>)}</div>{profileError && <p className="form-error" role="alert">{profileError}</p>}{profileStatus && <p className="form-success" role="status">{profileStatus}</p>}<button className="primary-button" type="button" disabled={profileBusy || !chosenAvatarID || chosenAvatarID === me?.avatar_id} onClick={saveAvatar}>{profileBusy ? "Salvando…" : "Salvar imagem"}</button></article>
		  <form className="panel profile-editor password-editor" onSubmit={changePassword}><header><p className="eyebrow">SEGURANÇA DA CONTA</p><h2>{me?.password_set ? "Trocar senha" : "Criar uma senha"}</h2><p>{me?.password_set ? "Confirme a credencial atual. Depois da troca, todas as sessões serão encerradas." : "Sua conta entrou pelo Google. Crie uma senha se também quiser usar e-mail e senha."}</p></header>{me?.password_set && <label>Senha atual<input required type="password" autoComplete="current-password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} /></label>}<label>Nova senha<input required minLength={12} maxLength={256} type="password" autoComplete="new-password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} /><small>Use de 12 a 256 caracteres.</small></label><label>Confirmar nova senha<input required minLength={12} maxLength={256} type="password" autoComplete="new-password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} /></label>{passwordError && <p className="form-error" role="alert">{passwordError}</p>}<button className="secondary-button" type="submit" disabled={passwordBusy}>{passwordBusy ? "Protegendo a conta…" : me?.password_set ? "Trocar senha e sair" : "Criar senha e sair"}</button></form>
		</section>
	</div>;
}

const tutorialSteps: Array<{
  id: TutorialStepId; phase: string; navLabel: string; icon: "journey" | "champion" | "mulligan" | "deck" | "duel" | "decision";
  title: string; body: string; tip: string; action: string; to?: string;
}> = [
  { id: "goal", phase: "Entender", navLabel: "Objetivo", icon: "journey", title: "Vença sem se perder no caminho", body: "Reduza a Vitalidade rival a zero. Cada carta custa sua própria Vitalidade: Assalto cria pressão, Guarda reduz o golpe e Rito prepara a próxima decisão.", tip: "A mesa ilumina somente ações válidas. Você pode passar uma fase quando preservar cartas for melhor.", action: "Entendi o objetivo" },
  { id: "avatars", phase: "Escolher", navLabel: "Avatares", icon: "champion", title: "Conheça os dez Avatares", body: "Todos começam com a mesma Vitalidade e o mesmo limite de 30 cartas. O poder do Avatar muda seu estilo — cura, Ward, custo ou ritmo — sem comprar vantagem.", tip: "Abra a tela de Avatares, compare os poderes e escolha aquele cuja forma de jogar parece natural para você.", action: "Abrir Avatares", to: "/champions" },
  { id: "collection", phase: "Explorar", navLabel: "Coleção", icon: "mulligan", title: "Leia as três famílias de carta", body: "Assaltos pressionam, Guardas respondem e Ritos ajustam a mão ou o próximo confronto. A coleção permite filtrar cada tipo e entender sua função antes de montar.", tip: "Comece pelas cartas competitivas. Relíquias e Manifestações ficam no arquivo e não entram neste modo.", action: "Explorar Coleção", to: "/collection" },
  { id: "deck", phase: "Preparar", navLabel: "Baralho", icon: "deck", title: "Revise seu baralho de 30 cartas", body: "O construtor mostra os mínimos de Assalto, Guarda e Rito e impede uma composição inválida. Sua conta já pode receber uma base pronta; revise o Avatar e salve quando estiver satisfeito.", tip: "A opção 10 / 10 / 10 cria um ponto de partida equilibrado. Depois você pode personalizar sem adivinhar os limites.", action: "Revisar Baralho", to: "/decks" },
  { id: "training", phase: "Praticar", navLabel: "Treino", icon: "duel", title: "Complete um treino guiado", body: "Na mesa, faça um Assalto, responda com Guarda quando puder e use um Rito. O coach explica a fase atual sem escolher a carta por você.", tip: "Esta etapa só recebe o check quando a partida terminar. O resultado mostra quais fundamentos realmente apareceram nos eventos do duelo.", action: "Iniciar treino guiado" },
  { id: "pvp", phase: "Duelar", navLabel: "PvP", icon: "decision", title: "Escolha seu próximo confronto", body: "Treino abre imediatamente contra o bot. Buscar rival entra na fila 1v1 e usa o mesmo baralho e as mesmas regras; só muda quem está do outro lado.", tip: "Se ainda houver uma etapa pendente, a faixa do tutorial continuará visível e trará você de volta ao ponto certo.", action: "Abrir tela Jogar", to: "/queue" },
];

export function TutorialPage() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const { data: deckData } = useDecks();
  const setActiveMatch = useSessionStore((state) => state.setActiveMatch);
  const setGuidedMatch = useSessionStore((state) => state.setGuidedMatch);
  const userId = useSessionStore((state) => state.user?.id ?? "");
  const journey = usePreferencesStore((state) => state.tutorialByUser[userId]);
  const beginTutorial = usePreferencesStore((state) => state.beginTutorial);
  const completeStep = usePreferencesStore((state) => state.completeTutorialStep);
  const restartTutorial = usePreferencesStore((state) => state.restartTutorial);
  const [guidedBusy, setGuidedBusy] = useState(false);
  const [guidedError, setGuidedError] = useState("");
  const requestedStep = Number(params.get("step"));
  const step = Number.isInteger(requestedStep) && requestedStep >= 0 && requestedStep < tutorialSteps.length ? requestedStep : 0;
  const rulesetVersion = useActiveRulesetVersion();
  const currentDeck = deckData?.decks.find((deck) => deck.ruleset_version === rulesetVersion && deck.active)
    ?? deckData?.decks.find((deck) => deck.ruleset_version === rulesetVersion);
  useEffect(() => { if (userId) beginTutorial(userId); }, [beginTutorial, userId]);
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
  const completed = journey?.completed ?? [];
  const isComplete = completed.includes(current.id);
  const completeAndOpen = () => {
    if (!userId) return;
    if (current.id === "training") { void startGuidedPractice(); return; }
    completeStep(userId, current.id);
    if (current.to) { navigate(current.to); return; }
    setStep(Math.min(tutorialSteps.length - 1, step + 1));
  };
  const reset = () => { if (userId) restartTutorial(userId); setStep(0); };
  return <div className="page tutorial-page"><header><p className="eyebrow"><span>PRIMEIRO DUELO</span> · {completed.length}/{TUTORIAL_STEP_IDS.length} <span>CONCLUÍDAS</span></p><h1>Você sempre saberá o próximo passo.</h1><p>Conclua no seu ritmo. Cada botão abre a tela certa e o guia permanece disponível até a jornada terminar.</p><div className="tutorial-entry-actions"><button className="ghost-button tutorial-restart" type="button" onClick={reset}>Reiniciar progresso</button></div>{guidedError && <p className="form-error" role="alert">{guidedError}</p>}<div className="tutorial-progress" aria-hidden="true">{tutorialSteps.map((item, index) => <span className={completed.includes(item.id) ? "is-complete" : index === step ? "is-current" : ""} key={item.id} />)}</div></header><nav className="tutorial-phases" aria-label="Etapas do guia">{tutorialSteps.map((item, index) => <button type="button" aria-label={`Etapa ${index + 1}: ${item.title}${completed.includes(item.id) ? " · concluída" : ""}`} aria-current={index === step ? "step" : undefined} className={`${index === step ? "is-current" : ""} ${completed.includes(item.id) ? "is-complete" : ""}`} onClick={() => setStep(index)} key={item.id}><span>{completed.includes(item.id) ? <UiIcon name="check" /> : String(index + 1).padStart(2, "0")}</span>{item.navLabel}</button>)}</nav><section className={`tutorial-card ${isComplete ? "is-complete" : ""}`}><div className="tutorial-illustration" aria-hidden="true"><VeilGlyph variant={current.icon} /><small>{current.phase}</small></div><div><p className="eyebrow">{current.phase} · ETAPA {step + 1}</p><h2>{current.title}</h2><p>{current.body}</p><div className="tutorial-tip"><strong>Como concluir</strong><span>{current.tip}</span></div><div className="tutorial-step-action"><span className={isComplete ? "is-done" : ""}><UiIcon name={isComplete ? "check" : "guide"} />{isComplete ? "Etapa concluída" : "Etapa pendente"}</span><button className="primary-button" type="button" disabled={guidedBusy && current.id === "training"} onClick={completeAndOpen}>{guidedBusy && current.id === "training" ? "Preparando a mesa…" : current.action}<UiIcon name="arrow-right" /></button></div></div></section><footer><button className="ghost-button" type="button" onClick={() => setStep(Math.max(0, step - 1))} disabled={step === 0}><UiIcon name="arrow-left" />Voltar</button><button className="ghost-button" type="button" onClick={() => setStep(Math.min(tutorialSteps.length - 1, step + 1))} disabled={step === tutorialSteps.length - 1}>Ver próxima etapa<UiIcon name="arrow-right" /></button></footer></div>;
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
