import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useDecks, useMe, useSeason } from "../queries";
import { usePreferencesStore, useSessionStore } from "../store";

export function ResultPage() {
  const battle = useSessionStore((state) => state.lastBattle);
  if (!battle) return <Missing title="Nenhum resultado nesta sessão" copy="Conclua uma partida para ver o resumo." action="Buscar duelo" to="/queue" />;
  const victory = battle.state.winner === battle.slot;
  const mine = battle.state.players[battle.slot];
  const rival = battle.state.players[1 - battle.slot];
  const cardsPlayed = battle.events.filter((event) => event.kind === "card_played");
  const damage = battle.events.filter((event) => event.kind === "damage_dealt" && event.p === 1 - battle.slot).reduce((sum, event) => sum + event.n, 0);
  return <div className={`page result-page ${victory ? "victory" : "defeat"}`}><div className="result-sun" aria-hidden="true">{victory ? "✦" : "◐"}</div><p className="eyebrow">PARTIDA ENCERRADA</p><h1>{victory ? "Vitória sob o Véu" : "O Véu se fechou"}</h1><p>{victory ? "Suas decisões sobreviveram ao Eclipse." : "Cada derrota preserva um mapa para o próximo duelo."}</p><div className="result-score"><div><span>VOCÊ</span><strong>♥ {mine.vitality}</strong></div><b>×</b><div><span>RIVAL</span><strong>♥ {rival.vitality}</strong></div></div><section className="result-stats"><article><small>RODADAS</small><strong>{battle.state.round}</strong></article><article><small>CARTAS JOGADAS</small><strong>{cardsPlayed.length}</strong></article><article><small>DANO CAUSADO</small><strong>{damage}</strong></article><article><small>ECLIPSE FINAL</small><strong>{battle.state.eclipse > 0 ? `+${battle.state.eclipse}` : battle.state.eclipse}</strong></article></section><div className="result-actions"><Link className="primary-button" to="/queue">Jogar novamente</Link><Link className="secondary-button" to="/replay">Ver replay da sessão</Link><Link className="ghost-button" to="/app">Voltar ao início</Link></div></div>;
}

export function ReplayPage() {
  const battle = useSessionStore((state) => state.lastBattle);
  const [cursor, setCursor] = useState(0);
  if (!battle) return <Missing title="Replay indisponível" copy="O replay da sessão aparece após uma partida concluída." action="Voltar" to="/app" />;
  const visible = battle.events.slice(0, cursor + 1);
  const current = battle.events[cursor];
  return <div className="page replay-page"><header className="page-header"><div><p className="eyebrow">REGISTRO DETERMINÍSTICO</p><h1>Replay da sessão</h1><p>Explore a linha do tempo redigida recebida durante a partida.</p></div><span className="count-badge">{battle.events.length} eventos</span></header><div className="replay-layout"><section className="replay-stage"><div className="replay-eclipse">◐</div><p className="eyebrow">RODADA {current?.round ?? 0}</p><h2>{current ? eventCopy(current.kind) : "Início da partida"}</h2><p>{current?.def ? `Carta ${current.def}` : current?.s || "A engine registrou cada transição em ordem."}</p><div className="replay-controls"><button type="button" onClick={() => setCursor(Math.max(0, cursor - 1))} disabled={cursor === 0}>← Anterior</button><input aria-label="Posição do replay" type="range" min="0" max={Math.max(0, battle.events.length - 1)} value={cursor} onChange={(event) => setCursor(Number(event.target.value))} /><button type="button" onClick={() => setCursor(Math.min(battle.events.length - 1, cursor + 1))} disabled={cursor >= battle.events.length - 1}>Próximo →</button></div><small>Evento {cursor + 1} de {battle.events.length}</small></section><aside className="replay-timeline"><h2>Linha do tempo</h2>{visible.slice().reverse().map((event) => <button type="button" className={event.seq === current?.seq ? "is-current" : ""} onClick={() => setCursor(battle.events.indexOf(event))} key={event.seq}><span>{event.seq + 1}</span><div><strong>{eventCopy(event.kind)}</strong><small>Rodada {event.round}</small></div></button>)}</aside></div><p className="honesty-note">Este Alpha preserva aqui o replay da sessão atual. Consulta a partidas antigas será ligada quando a API de histórico for aberta.</p></div>;
}

export function ProfilePage() {
  const { data: me } = useMe();
  const { data: season } = useSeason();
  const { data: decks } = useDecks();
  return <div className="page profile-page"><section className="profile-hero"><div className="profile-avatar">{me?.display_name?.slice(0, 1).toUpperCase() ?? "V"}</div><div><p className="eyebrow">PERFIL DO DUELISTA</p><h1>{me?.display_name ?? "Viajante"}</h1><p>Participante do Alpha · {me?.role === "admin" ? "Guardião" : "Jogador"}</p></div><span className="season-seal">◐<small>{season?.name ?? "Alpha"}</small></span></section><section className="profile-grid"><article className="rank-card"><p className="eyebrow">RANKED</p><h2>Pré-temporada</h2><div className="rank-emblem">♜</div><strong>Sem colocação</strong><p>O Alpha não inventa MMR antes do módulo ranked registrar partidas e critérios oficiais.</p></article><article className="panel"><h2>Conta competitiva</h2><dl className="profile-details"><div><dt>Ruleset</dt><dd>{season?.ruleset_version ?? "alpha-0.4.0"}</dd></div><div><dt>Decks salvos</dt><dd>{decks?.decks.length ?? 0}</dd></div><div><dt>Coleção</dt><dd>Competitiva completa</dd></div><div><dt>Temporada</dt><dd>{season?.name ?? "Alpha"}</dd></div></dl></article></section></div>;
}

const tutorialSteps = [
  { icon: "⚔", title: "Cause dano", body: "Assaltos ferem a Vitalidade rival no Confronto. Leve-a a zero para vencer. A Postura Predação dá +1 ao seu primeiro Assalto da rodada." },
  { icon: "⬡", title: "Responda com Guarda", body: "Quando um Assalto é anunciado, o defensor recebe uma janela clara: jogar no máximo uma Guarda ou passar. Vigília barateia a primeira Guarda." },
  { icon: "❖", title: "Prepare Ritos", body: "Antes do Confronto vem a fase de Rito: cura, compra, Maldições e manipulação do Eclipse. Essência paga tudo — cresce por rodada, e a temporária expira no Crepúsculo. Arcano barateia o primeiro Rito sem dano." },
  { icon: "◐", title: "Dispute o Eclipse", body: "Cada carta pode mover o medidor entre Aurora (−3) e Noite (+3). Atingir um extremo desperta efeitos totais até o fim da rodada — negue o pico do rival ou provoque o seu." },
  { icon: "✦", title: "Encadeie Sigilos", body: "Cartas emitem Presa, Sol, Espelho, Garra, Cinza ou Coroa. A ordem da sua Trilha de Ressonância abre bônus únicos — combos são sequências, não solitários." },
  { icon: "◆", title: "Assente Relíquias", body: "Relíquias são permanentes (máximo de 2 ativas) que disparam efeitos a cada rodada. A terceira exige abrir mão de uma existente." },
  { icon: "☽", title: "Convoque Manifestações", body: "Manifestações são aliados persistentes (máximo de 2) com gatilhos próprios. Não combatem sozinhas — amplificam suas jogadas e podem ser silenciadas." },
  { icon: "▤", title: "Monte seu deck", body: "36 cartas, no máximo 2 cópias (1 de Lendária), com pelo menos 24 da facção do seu Campeão ou neutras. Comece por um precon oficial e ajuste ao seu estilo." },
];

export function TutorialPage() {
  const [step, setStep] = useState(0);
  const current = tutorialSteps[step];
  return <div className="page tutorial-page"><header><p className="eyebrow">RITO DE INICIAÇÃO · {step + 1}/{tutorialSteps.length}</p><h1>Fundamentos do Véu</h1><div className="tutorial-progress">{tutorialSteps.map((_, index) => <button type="button" aria-label={`Ir à etapa ${index + 1}`} className={index <= step ? "is-complete" : ""} onClick={() => setStep(index)} key={index} />)}</div></header><section className="tutorial-card"><div className="tutorial-illustration" aria-hidden="true"><span>{current.icon}</span></div><div><p className="eyebrow">FUNDAMENTO {step + 1}</p><h2>{current.title}</h2><p>{current.body}</p><div className="tutorial-tip"><strong>Na mesa</strong><span>{step === 3 ? "O medidor permanece sempre no centro e anuncia quando um estado total foi ativado." : step === 1 ? "A faixa de Guarda pulsa e o cronômetro permanece visível." : step === 7 ? "No construtor, use um precon como ponto de partida e o validador aponta qualquer regra violada." : "Passe o foco pelas cartas para ler o texto completo antes de agir."}</span></div></div></section><footer><button className="ghost-button" type="button" onClick={() => setStep(Math.max(0, step - 1))} disabled={step === 0}>← Voltar</button>{step < tutorialSteps.length - 1 ? <button className="primary-button" type="button" onClick={() => setStep(step + 1)}>Continuar →</button> : <Link className="primary-button" to="/decks">Montar meu deck</Link>}</footer></div>;
}

export function SettingsPage() {
  const preferences = usePreferencesStore();
  const options = useMemo(() => [
    ["sound", "Som", "Efeitos sonoros da interface e batalha."],
    ["reducedMotion", "Reduzir movimento", "Desativa rotações, pulsos e transições decorativas."],
    ["highContrast", "Alto contraste", "Reforça bordas e contraste dos controles."],
    ["largeText", "Texto ampliado", "Aumenta o tamanho-base da interface."],
  ] as const, []);
  return <div className="page settings-page"><header className="page-header"><div><p className="eyebrow">PREFERÊNCIAS LOCAIS</p><h1>Configurações</h1><p>Ajustes ficam somente neste dispositivo.</p></div></header><section className="settings-panel"><h2>Acessibilidade e experiência</h2>{options.map(([key, title, copy]) => <label className="setting-row" key={key}><span><strong>{title}</strong><small>{copy}</small></span><input type="checkbox" checked={preferences[key]} onChange={(event) => preferences.set(key, event.target.checked)} /><i aria-hidden="true" /></label>)}</section><section className="settings-panel"><h2>Sobre o Alpha</h2><dl className="profile-details"><div><dt>Cliente</dt><dd>PWA Fase 5</dd></div><div><dt>Ruleset</dt><dd>alpha-0.4.0</dd></div><div><dt>Privacidade</dt><dd>Tokens na sessão</dd></div><div><dt>Conteúdo</dt><dd>IP original</dd></div></dl></section></div>;
}

export function Missing({ title = "Caminho não encontrado", copy = "Esta área não existe ou foi movida.", action = "Ir ao início", to = "/app" }: { title?: string; copy?: string; action?: string; to?: string }) { return <div className="missing-page"><span>◐</span><h1>{title}</h1><p>{copy}</p><Link className="primary-button" to={to}>{action}</Link></div>; }

const eventCopy = (kind: string) => ({ match_started: "Partida iniciada", round_started: "Nova rodada", card_played: "Carta jogada", damage_dealt: "Dano causado", damage_prevented: "Dano prevenido", eclipse_shifted: "Eclipse deslocado", eclipse_triggered: "Eclipse total", sigil_added: "Ressonância ampliada", match_ended: "Partida encerrada" }[kind] ?? kind.replaceAll("_", " "));
