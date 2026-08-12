import { useCallback, useEffect, useState } from "react";
import { api } from "../api";
import { CURRENT_RULESET } from "../rules";
import { useSessionStore } from "../store";
import "../admin.css";

// Painel LiveOps (Fase 7 no backend; dívida de UI da Fase 10). Toda mutação
// é auditada no servidor ("sem auditoria, sem mudança") — o painel é uma
// casca fina e honesta sobre a API admin.

interface RulesetInfo { version: string; active: boolean; created_at: string }
interface CardBan { id: string; card_id: string; reason: string; created_by: string; created_at: string; lifted_at?: string }
interface Season { id: string; name: string; ruleset_version: string; starts_at: string }
interface AuditEntry { id: number; actor: string; action: string; subject: string; created_at: string }
interface ChampionStat { champion_id: string; games: number; wins: number; win_rate: number }
interface RhythmStats {
  sample_matches: number;
  average_duration_seconds: number;
  p50_duration_seconds: number;
  p95_duration_seconds: number;
  average_rounds: number;
  p50_rounds: number;
  p95_rounds: number;
  over_thirty_minutes: number;
}
interface Telemetry {
  total_matches: number;
  finished_matches: number;
  by_champion: ChampionStat[] | null;
  rhythm: RhythmStats;
}
interface Draft {
  id: string; card_id: string; status: string; note: string;
  card: unknown; effects: unknown; last_validation?: unknown; published_version?: string; updated_at: string;
}

interface AdminOverview {
  total_players: number; total_admins: number; banned_players: number; new_players_7d: number;
  active_players_7d: number; active_players_30d: number; total_matches: number; active_matches: number;
  finished_matches: number; cancelled_matches: number; pvp_matches: number; practice_matches: number;
  can_invite_admins: boolean;
}
interface AdminPlayer {
  id: string; email: string; display_name: string; role: "player" | "admin" | "owner";
  account_level: number; created_at: string; last_session_at?: string; match_count: number; wins: number;
  banned_at?: string; banned_reason?: string;
  deactivated_at?: string;
}
interface AdminMatchPlayer { user_id: string; display_name: string; slot: number }
interface AdminMatch {
  id: string; mode: string; ruleset_version: string; status: string; players: AdminMatchPlayer[];
  winner_slot?: number; end_reason?: string; created_at: string; started_at?: string; ended_at?: string;
  duration_seconds: number;
}
interface AdminInvite {
  id: string; email: string; created_by: string; expires_at: string; used_at?: string; used_by?: string;
  created_at: string; token?: string;
}

export function AdminPage() {
  const principal = useSessionStore((state) => state.principal);
  if (principal?.role !== "admin" && principal?.role !== "owner") {
    return <main className="page admin-page">
      <header className="admin-head">
        <p className="eyebrow">ACESSO RESTRITO</p>
        <h1>Salão de Controle</h1>
        <p className="admin-empty">Esta área é exclusiva para a equipe de operação.</p>
      </header>
    </main>;
  }
  return <div className="page admin-page">
    <header className="admin-head admin-hero"><div><p className="eyebrow">SALÃO DE CONTROLE</p><h1>Centro de Operações</h1>
      <p className="admin-sub">Jogadores, partidas, moderação e saúde do jogo em uma única visão.</p></div>
      <span className="admin-hero__identity"><small>Sessão administrativa</small><strong>{principal.display_name}</strong><b>{principal.role === "owner" ? "PROPRIETÁRIO" : "ADMIN"}</b></span></header>
    <OverviewPanel />
    <PlayersPanel />
    <MatchesPanel />
    {principal.role === "owner" && <AdminInvitesPanel />}
    <header className="admin-section-title"><div><p className="eyebrow">CONFIGURAÇÃO DO JOGO</p><h2>LiveOps</h2></div><p>Mudanças auditadas no servidor.</p></header>
    <RulesetsPanel />
    <div className="admin-grid">
      <BansPanel />
      <SeasonPanel />
    </div>
    <TelemetryPanel />
    <AlphaNotesPanel />
    <DraftsPanel />
    <AuditPanel />
  </div>;
}

function OverviewPanel() {
  const [overview, setOverview] = useState<AdminOverview | null>(null);
  const [error, setError] = useState("");
  const load = useCallback(() => {
    setError("");
    void api<AdminOverview>("/v1/admin/overview").then(setOverview).catch((caught) => setError(errorText(caught)));
  }, []);
  useEffect(load, [load]);
  const stats = overview ? [
    ["Jogadores", overview.total_players, `${overview.new_players_7d} novos em 7 dias`],
    ["Ativos · 7 dias", overview.active_players_7d, `${overview.active_players_30d} ativos em 30 dias`],
    ["Partidas", overview.total_matches, `${overview.active_matches} em andamento`],
    ["Banidos", overview.banned_players, `${overview.total_admins} contas administrativas`],
  ] as const : [];
  return <section className="admin-overview" aria-label="Visão geral operacional">
    <header><div><p className="eyebrow">AGORA</p><h2>Visão operacional</h2></div><button type="button" className="admin-refresh" onClick={load}>Atualizar</button></header>
    {overview ? <><div className="admin-stat-grid">{stats.map(([label, value, detail]) => <article key={label}>
      <span>{label}</span><strong>{value.toLocaleString("pt-BR")}</strong><small>{detail}</small>
    </article>)}</div><div className="admin-health-row">
      <span><b>{overview.finished_matches}</b> encerradas</span><span><b>{overview.pvp_matches}</b> PvP</span>
      <span><b>{overview.practice_matches}</b> treinos</span><span><b>{overview.cancelled_matches}</b> canceladas</span>
    </div></> : <p className={error ? "admin-feedback is-error" : "admin-empty"}>{error || "Carregando indicadores…"}</p>}
  </section>;
}

function PlayersPanel() {
  const [players, setPlayers] = useState<AdminPlayer[]>([]);
  const [query, setQuery] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [selected, setSelected] = useState<AdminPlayer | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [feedback, notify, isError] = useFeedback();
  const load = useCallback((search = query) => {
    setLoaded(false);
    void api<AdminPlayer[]>(`/v1/admin/players?limit=100&q=${encodeURIComponent(search.trim())}`)
      .then((list) => setPlayers(list ?? [])).catch((caught) => notify("erro", errorText(caught))).finally(() => setLoaded(true));
  }, [query]);
  useEffect(() => { load(""); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const moderate = async (player: AdminPlayer, action: "ban" | "unban") => {
    if (action === "ban" && !window.confirm(`Banir ${player.display_name}? As sessões abertas serão encerradas imediatamente.`)) return;
    if (action === "unban" && !window.confirm(`Remover o banimento de ${player.display_name}?`)) return;
    setBusy(true);
    try {
      await api(`/v1/admin/players/${player.id}/${action}`, { method: "POST",
        ...(action === "ban" ? { body: JSON.stringify({ reason: reason.trim() }) } : {}) });
      notify("ok", action === "ban" ? `${player.display_name} foi banido e teve as sessões revogadas.` : `${player.display_name} foi liberado.`);
      setSelected(null); setReason(""); load(query);
    } catch (caught) { notify("erro", errorText(caught)); }
    finally { setBusy(false); }
  };

  return <section className="panel admin-panel admin-players" aria-label="Jogadores">
    <header><div><p className="eyebrow">CONTAS</p><h2>Jogadores</h2></div>
      <form className="admin-search" onSubmit={(event) => { event.preventDefault(); load(query); }}>
        <input type="search" placeholder="Nome ou e-mail" value={query} onChange={(event) => setQuery(event.target.value)} />
        <button type="submit">Buscar</button>
      </form></header>
    <div className="admin-table-wrap"><table className="admin-table admin-player-table"><thead><tr>
      <th>Conta</th><th>Papel</th><th>Nível</th><th>Partidas</th><th>Último acesso</th><th>Estado</th><th />
    </tr></thead><tbody>{players.map((player) => <tr key={player.id} className={player.banned_at ? "is-banned" : ""}>
      <td><strong>{player.display_name}</strong><small>{player.email}<br />desde {new Date(player.created_at).toLocaleDateString("pt-BR")}</small></td>
      <td><span className={`admin-role is-${player.role}`}>{player.role === "owner" ? "Proprietário" : player.role === "admin" ? "Admin" : "Jogador"}</span></td>
      <td>{player.account_level}</td><td>{player.match_count} <small>· {player.wins} vitórias</small></td>
      <td>{player.last_session_at ? new Date(player.last_session_at).toLocaleString("pt-BR") : "Nunca"}</td>
	  <td>{player.banned_at ? <span className="admin-status is-danger">BANIDO</span> : player.deactivated_at ? <span className="admin-status is-muted">DESATIVADA</span> : <span className="admin-status is-ok">ATIVO</span>}{player.banned_reason && <small>{player.banned_reason}</small>}</td>
      <td className="admin-actions">{player.role === "player" && (player.banned_at
        ? <button type="button" disabled={busy} onClick={() => moderate(player, "unban")}>Liberar</button>
        : <button type="button" className="danger" disabled={busy} onClick={() => { setSelected(player); setReason(""); }}>Banir</button>)}</td>
    </tr>)}</tbody></table></div>
    {!loaded ? <p className="admin-empty">Carregando contas…</p> : players.length === 0 && <p className="admin-empty">Nenhuma conta encontrada.</p>}
    {selected && <div className="admin-moderation" role="dialog" aria-label={`Banir ${selected.display_name}`}>
      <div><p className="eyebrow">AÇÃO DE MODERAÇÃO</p><h3>Banir {selected.display_name}</h3><p>{selected.email}</p></div>
      <label>Motivo auditável<textarea rows={3} minLength={4} maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Descreva objetivamente a violação…" /></label>
      <div><button type="button" onClick={() => setSelected(null)}>Cancelar</button><button type="button" className="danger" disabled={busy || reason.trim().length < 4} onClick={() => moderate(selected, "ban")}>Banir e encerrar sessões</button></div>
    </div>}
    <Feedback text={feedback} error={isError} />
  </section>;
}

function MatchesPanel() {
  const [matches, setMatches] = useState<AdminMatch[]>([]);
  const [loaded, setLoaded] = useState(false);
  const load = useCallback(() => { setLoaded(false); void api<AdminMatch[]>("/v1/admin/matches?limit=50").then((list) => setMatches(list ?? [])).catch(() => undefined).finally(() => setLoaded(true)); }, []);
  useEffect(load, [load]);
  return <section className="panel admin-panel" aria-label="Partidas recentes">
    <header><div><p className="eyebrow">ATIVIDADE</p><h2>Partidas recentes</h2></div><button type="button" className="admin-refresh" onClick={load}>Atualizar</button></header>
    <div className="admin-table-wrap"><table className="admin-table"><thead><tr><th>Partida</th><th>Jogadores</th><th>Modo</th><th>Estado</th><th>Duração</th><th>Início</th></tr></thead>
      <tbody>{matches.map((match) => <tr key={match.id}><td><a href={`/cronica/${match.id}`}><code>{match.id.slice(0, 8)}…</code></a><small>{match.ruleset_version}</small></td>
        <td>{match.players.map((player) => <span className={match.winner_slot === player.slot ? "admin-winner" : ""} key={player.user_id}>{player.display_name}</span>)}</td>
        <td>{match.mode === "practice" ? "Treino" : "PvP"}</td><td><span className={`admin-status is-${match.status}`}>{statusLabel(match.status)}</span>{match.end_reason && <small>{match.end_reason}</small>}</td>
        <td>{match.duration_seconds ? formatDuration(match.duration_seconds) : "—"}</td><td>{new Date(match.created_at).toLocaleString("pt-BR")}</td></tr>)}</tbody></table></div>
    {!loaded ? <p className="admin-empty">Carregando partidas…</p> : matches.length === 0 && <p className="admin-empty">Nenhuma partida registrada.</p>}
  </section>;
}

function statusLabel(status: string) {
  return ({ waiting_ready: "AGUARDANDO", active: "EM JOGO", finished: "ENCERRADA", cancelled: "CANCELADA" } as Record<string, string>)[status] ?? status.toUpperCase();
}

function AdminInvitesPanel() {
  const [email, setEmail] = useState("");
  const [invites, setInvites] = useState<AdminInvite[]>([]);
  const [link, setLink] = useState("");
  const [busy, setBusy] = useState(false);
  const [feedback, notify, isError] = useFeedback();
  const load = useCallback(() => { void api<AdminInvite[]>("/v1/admin/invites?limit=30").then((list) => setInvites(list ?? [])).catch(() => undefined); }, []);
  useEffect(load, [load]);
  const issue = async () => {
    setBusy(true); setLink("");
    try {
      const invite = await api<AdminInvite>("/v1/admin/invites", { method: "POST", body: JSON.stringify({ email: email.trim() }) });
      const url = `${window.location.origin}/?admin_invite=${encodeURIComponent(invite.token ?? "")}`;
      setLink(url); setEmail(""); notify("ok", "Convite criado. O segredo é exibido somente agora."); load();
    } catch (caught) { notify("erro", errorText(caught)); }
    finally { setBusy(false); }
  };
  return <section className="panel admin-panel admin-invites" aria-label="Convites administrativos">
    <header><div><p className="eyebrow">ACESSO PRIVILEGIADO</p><h2>Administradores por convite</h2></div><span className="admin-status is-owner">SOMENTE PROPRIETÁRIO</span></header>
    <p className="admin-callout">O cadastro público nunca cria administradores. Cada convite fica preso a um e-mail, expira em 24 horas e só pode ser usado uma vez.</p>
    <div className="admin-form"><input type="email" placeholder="novo.admin@exemplo.com" value={email} onChange={(event) => setEmail(event.target.value)} /><button type="button" disabled={busy || !email.trim()} onClick={issue}>Emitir convite</button></div>
    {link && <div className="admin-secret"><label>Link único<input readOnly value={link} onFocus={(event) => event.currentTarget.select()} /></label><button type="button" onClick={() => { void navigator.clipboard.writeText(link).then(() => notify("ok", "Link copiado.")); }}>Copiar</button></div>}
    {invites.length > 0 && <ul className="admin-invite-list">{invites.map((invite) => { const expired = !invite.used_at && new Date(invite.expires_at).getTime() <= Date.now(); return <li key={invite.id}>
      <div><strong>{invite.email}</strong><small>emitido em {new Date(invite.created_at).toLocaleString("pt-BR")} · expira {new Date(invite.expires_at).toLocaleString("pt-BR")}</small></div>
      <span className={`admin-status ${invite.used_at ? "is-ok" : expired ? "is-danger" : "is-waiting_ready"}`}>{invite.used_at ? "USADO" : expired ? "EXPIRADO" : "PENDENTE"}</span>
    </li>; })}</ul>}
    <Feedback text={feedback} error={isError} />
  </section>;
}

interface AlphaNote {
  id: string;
  user_id: string;
  match_id?: string;
  ruleset_version: string;
  message: string;
  created_at: string;
}

// Recados do Alpha. Pedir opinião e não ler é pior que não pedir: este painel
// existe para que o convite da tela de resultado tenha destino.
function AlphaNotesPanel() {
  const [notes, setNotes] = useState<AlphaNote[]>([]);
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    void api<{ feedback: AlphaNote[] | null }>("/v1/admin/feedback?limit=100")
      .then((data) => setNotes(data.feedback ?? []))
      .catch(() => undefined)
      .finally(() => setLoaded(true));
  }, []);
  return <section className="panel admin-panel" aria-label="Recados do Alpha">
    <header><p className="eyebrow">VOZ DE QUEM JOGA</p><h2>Recados do Alpha</h2></header>
    {!loaded ? <p className="admin-empty">Carregando…</p>
      : notes.length === 0 ? <p className="admin-empty">Nenhum recado ainda. O convite aparece na tela de resultado e é opcional.</p>
      : <ul className="alpha-notes">
          {notes.map((note) => <li key={note.id}>
            <header>
              <time dateTime={note.created_at}>{new Date(note.created_at).toLocaleString("pt-BR")}</time>
              <span className="alpha-notes__ruleset">{note.ruleset_version}</span>
              {note.match_id && <a href={`/cronica/${note.match_id}`}>ver a partida</a>}
            </header>
            <p>{note.message}</p>
          </li>)}
        </ul>}
  </section>;
}

function useFeedback(): [string, (tone: "ok" | "erro", text: string) => void, boolean] {
  const [text, setText] = useState("");
  const [isError, setIsError] = useState(false);
  return [text, (tone, value) => { setText(value); setIsError(tone === "erro"); }, isError];
}

function Feedback({ text, error }: { text: string; error: boolean }) {
  if (!text) return null;
  return <p className={`admin-feedback ${error ? "is-error" : ""}`} role="status">{text}</p>;
}

const errorText = (caught: unknown) => (caught instanceof Error ? caught.message : "A operação falhou.");

function RulesetsPanel() {
  const [rulesets, setRulesets] = useState<RulesetInfo[]>([]);
  const [feedback, notify, isError] = useFeedback();
  const [busy, setBusy] = useState("");
  const reload = useCallback(() => { void api<RulesetInfo[]>("/v1/admin/rulesets").then(setRulesets).catch(() => undefined); }, []);
  useEffect(reload, [reload]);

  const act = async (kind: "activate" | "rotate", version: string) => {
    const warning = kind === "activate"
      ? `Ativar ${version}? O matchmaking passa a usar esta versão imediatamente.`
      : `Rotacionar para ${version}? Concede a coleção da versão a todos e clona decks válidos.`;
    if (!window.confirm(warning)) return;
    setBusy(version + kind);
    try {
      if (kind === "activate") {
        await api<void>(`/v1/admin/rulesets/${version}/activate`, { method: "POST" });
        notify("ok", `${version} ativada — matchmaking repontado.`);
      } else {
        const result = await api<{ granted: number; decks_cloned: number }>(`/v1/admin/rulesets/${version}/rotate`, { method: "POST" });
        notify("ok", `Rotação para ${version}: ${result.granted} concessões, ${result.decks_cloned} decks clonados.`);
      }
      reload();
    } catch (caught) { notify("erro", errorText(caught)); }
    finally { setBusy(""); }
  };

  return <section className="panel admin-panel" aria-label="Rulesets">
    <header><p className="eyebrow">VERSÕES DE REGRA</p><h2>Rulesets</h2></header>
    <table className="admin-table"><thead><tr><th>Versão</th><th>Estado</th><th>Publicada</th><th /></tr></thead>
      <tbody>{rulesets.map((ruleset) => <tr key={ruleset.version}>
        <td><code>{ruleset.version}</code></td>
        <td>{ruleset.active ? <b className="admin-badge is-active">ATIVA</b> : <span className="admin-badge">inativa</span>}</td>
        <td>{new Date(ruleset.created_at).toLocaleDateString("pt-BR")}</td>
        <td className="admin-actions">
          {!ruleset.active && <button type="button" disabled={busy !== ""} onClick={() => act("activate", ruleset.version)}>Ativar</button>}
          <button type="button" disabled={busy !== ""} onClick={() => act("rotate", ruleset.version)}>Rotacionar</button>
        </td>
      </tr>)}</tbody></table>
    <small>Ordem operacional: publicar → rotacionar → ativar. Rollback é ativar uma versão anterior.</small>
    <Feedback text={feedback} error={isError} />
  </section>;
}

function BansPanel() {
  const [bans, setBans] = useState<CardBan[]>([]);
  const [cardID, setCardID] = useState("");
  const [reason, setReason] = useState("");
  const [feedback, notify, isError] = useFeedback();
  const reload = useCallback(() => { void api<CardBan[]>("/v1/admin/bans").then((list) => setBans(list ?? [])).catch(() => undefined); }, []);
  useEffect(reload, [reload]);

  const ban = async () => {
    try {
      await api<CardBan>("/v1/admin/bans", { method: "POST", body: JSON.stringify({ card_id: cardID.trim().toUpperCase(), reason: reason.trim() }) });
      notify("ok", `${cardID.toUpperCase()} banida do competitivo.`);
      setCardID(""); setReason(""); reload();
    } catch (caught) { notify("erro", errorText(caught)); }
  };
  const lift = async (id: string) => {
    try { await api<CardBan>(`/v1/admin/bans/${id}/lift`, { method: "POST" }); notify("ok", `${id} liberada.`); reload(); }
    catch (caught) { notify("erro", errorText(caught)); }
  };

  const active = bans.filter((entry) => !entry.lifted_at);
  return <section className="panel admin-panel" aria-label="Bans">
    <header><p className="eyebrow">EMERGÊNCIA</p><h2>Bans competitivos</h2></header>
    {active.length ? <ul className="admin-list">{active.map((entry) => <li key={entry.id}>
      <div><code>{entry.card_id}</code><small>{entry.reason} · {entry.created_by}</small></div>
      <button type="button" onClick={() => lift(entry.card_id)}>Derrubar</button>
    </li>)}</ul> : <p className="admin-empty">Nenhuma carta banida.</p>}
    <div className="admin-form">
      <input placeholder="VR-000" value={cardID} onChange={(event) => setCardID(event.target.value)} />
      <input placeholder="Motivo (obrigatório)" value={reason} onChange={(event) => setReason(event.target.value)} />
      <button type="button" className="danger" disabled={!cardID.trim() || !reason.trim()} onClick={ban}>Banir</button>
    </div>
    <small>Bans desativam a carta no competitivo sem apagar histórico ou coleções.</small>
    <Feedback text={feedback} error={isError} />
  </section>;
}

function SeasonPanel() {
  const [season, setSeason] = useState<Season | null>(null);
  const [name, setName] = useState("");
  const [feedback, notify, isError] = useFeedback();
  const reload = useCallback(() => { void api<Season>("/v1/seasons/current").then(setSeason).catch(() => setSeason(null)); }, []);
  useEffect(reload, [reload]);

  const open = async () => {
    if (!window.confirm(`Abrir a temporada "${name}"? A temporada atual será encerrada.`)) return;
    try {
      await api<Season>("/v1/admin/seasons", { method: "POST",
        body: JSON.stringify({ name: name.trim(), starts_at: new Date().toISOString() }) });
      notify("ok", `Temporada "${name}" aberta.`); setName(""); reload();
    } catch (caught) { notify("erro", errorText(caught)); }
  };

  return <section className="panel admin-panel" aria-label="Temporada">
    <header><p className="eyebrow">CICLO COMPETITIVO</p><h2>Temporada</h2></header>
    {season ? <p className="admin-season"><strong>{season.name}</strong><small>ruleset {season.ruleset_version} · desde {new Date(season.starts_at).toLocaleDateString("pt-BR")}</small></p>
      : <p className="admin-empty">Sem temporada aberta.</p>}
    <div className="admin-form">
      <input placeholder="Nome da nova temporada" value={name} onChange={(event) => setName(event.target.value)} />
      <button type="button" disabled={!name.trim()} onClick={open}>Abrir temporada</button>
    </div>
    <small>Abrir uma temporada encerra a anterior; ratings ranqueados começam do zero na nova.</small>
    <Feedback text={feedback} error={isError} />
  </section>;
}

function TelemetryPanel() {
  const [data, setData] = useState<Telemetry | null>(null);
  useEffect(() => { void api<Telemetry>("/v1/admin/telemetry").then(setData).catch(() => undefined); }, []);
  return <section className="panel admin-panel" aria-label="Telemetria">
    <header><p className="eyebrow">O QUE A MESA CONTA</p><h2>Telemetria</h2></header>
    {data ? <>
      <p className="admin-season"><strong>{data.finished_matches}</strong><small>partidas encerradas de {data.total_matches} criadas</small></p>
      {data.rhythm.sample_matches > 0 ? <>
        <div className="admin-grid">
          <p className="admin-season"><strong>{formatDuration(data.rhythm.p50_duration_seconds)}</strong><small>duração mediana · p95 {formatDuration(data.rhythm.p95_duration_seconds)}</small></p>
          <p className="admin-season"><strong>{data.rhythm.p50_rounds.toFixed(0)}</strong><small>rodadas medianas · p95 {data.rhythm.p95_rounds.toFixed(0)}</small></p>
        </div>
        <p className="admin-empty">Amostra humana: {data.rhythm.sample_matches} PvP · média {formatDuration(data.rhythm.average_duration_seconds)} / {data.rhythm.average_rounds.toFixed(1)} rodadas · {data.rhythm.over_thirty_minutes} acima de 30 min.</p>
      </> : <p className="admin-empty">O ritmo humano aparecerá após a primeira PvP concluída.</p>}
      {data.by_champion?.length ? <table className="admin-table"><thead><tr><th>Avatar</th><th>Partidas</th><th>Vitórias</th><th>WR</th></tr></thead>
        <tbody>{data.by_champion.map((stat) => <tr key={stat.champion_id}>
          <td><code>{stat.champion_id}</code></td><td>{stat.games}</td><td>{stat.wins}</td>
          <td>{(stat.win_rate * 100).toFixed(1)}%</td>
        </tr>)}</tbody></table> : <p className="admin-empty">Sem partidas suficientes para agregados.</p>}
    </> : <p className="admin-empty">Carregando…</p>}
  </section>;
}

function formatDuration(seconds: number) {
  const minutes = Math.max(0, Math.round(seconds / 60));
  if (minutes < 60) return `${minutes} min`;
  return `${Math.floor(minutes / 60)}h ${String(minutes % 60).padStart(2, "0")}min`;
}

function AuditPanel() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const reload = useCallback(() => { void api<AuditEntry[]>("/v1/admin/audit?limit=20").then((list) => setEntries(list ?? [])).catch(() => undefined); }, []);
  useEffect(reload, [reload]);
  return <section className="panel admin-panel" aria-label="Auditoria">
    <header><p className="eyebrow">TRILHA</p><h2>Auditoria</h2><button type="button" className="admin-refresh" onClick={reload}>Atualizar</button></header>
    {entries.length ? <ul className="admin-audit">{entries.map((entry) => <li key={entry.id}>
      <code>{entry.action}</code><span>{entry.subject}</span>
      <small>{entry.actor.slice(0, 8)}… · {new Date(entry.created_at).toLocaleString("pt-BR")}</small>
    </li>)}</ul> : <p className="admin-empty">Nenhuma ação administrativa registrada.</p>}
  </section>;
}

const DRAFT_CARD_TEMPLATE = `{
  "id": "VR-901",
  "name": "Nova Carta",
  "faction": "Errantes",
  "type": "Rito",
  "rarity": "Comum",
  "cost": 2,
  "eclipse_shift": 0,
  "sigil": "Coroa",
  "rules_text": "Compre 1.",
  "flavor": "…",
  "design_role": "exemplo"
}`;
const DRAFT_FX_TEMPLATE = `{ "rite": { "steps": [ { "op": "draw", "n": 1 } ] } }`;

function DraftsPanel() {
  const [drafts, setDrafts] = useState<Draft[]>([]);
  const [cardJSON, setCardJSON] = useState(DRAFT_CARD_TEMPLATE);
  const [fxJSON, setFxJSON] = useState(DRAFT_FX_TEMPLATE);
  const [note, setNote] = useState("");
  const [games, setGames] = useState("500");
  const [version, setVersion] = useState("");
  const [selected, setSelected] = useState("");
  const [output, setOutput] = useState("");
  const [feedback, notify, isError] = useFeedback();
  const reload = useCallback(() => { void api<Draft[]>("/v1/admin/drafts").then((list) => setDrafts(list ?? [])).catch(() => undefined); }, []);
  useEffect(reload, [reload]);

  const parse = (raw: string, label: string): unknown => {
    try { return JSON.parse(raw); } catch { throw new Error(`${label}: JSON inválido.`); }
  };

  const create = async () => {
    try {
      const card = parse(cardJSON, "Carta") as { id?: string };
      const effects = parse(fxJSON, "Efeitos");
      const draft = await api<Draft>("/v1/admin/drafts", { method: "POST",
        body: JSON.stringify({ card_id: card.id ?? "", note: note.trim(), card, effects }) });
      notify("ok", `Draft ${draft.id.slice(0, 8)}… criado para ${draft.card_id}.`);
      setSelected(draft.id); reload();
    } catch (caught) { notify("erro", errorText(caught)); }
  };

  const step = async (action: "validate" | "simulate" | "publish") => {
    if (!selected) { notify("erro", "Selecione um draft."); return; }
    setOutput("Executando…");
    try {
      let result: unknown;
      if (action === "validate") result = await api(`/v1/admin/drafts/${selected}/validate`, { method: "POST" });
      if (action === "simulate") result = await api(`/v1/admin/drafts/${selected}/simulate`, { method: "POST", body: JSON.stringify({ games: Number(games) || 200 }) });
      if (action === "publish") result = await api(`/v1/admin/drafts/${selected}/publish`, { method: "POST", body: JSON.stringify({ version: version.trim() }) });
      setOutput(JSON.stringify(result, null, 2));
      notify("ok", `${action} concluído.`); reload();
    } catch (caught) { setOutput(""); notify("erro", errorText(caught)); }
  };

  return <section className="panel admin-panel" aria-label="Forja de cartas">
    <header><p className="eyebrow">FORJA DE CARTAS</p><h2>Drafts</h2></header>
    <div className="admin-drafts">
      <div className="admin-drafts__editor">
        <label>Carta (CardDef JSON)<textarea rows={9} value={cardJSON} onChange={(event) => setCardJSON(event.target.value)} spellCheck={false} /></label>
        <label>Efeitos (CardFx JSON)<textarea rows={4} value={fxJSON} onChange={(event) => setFxJSON(event.target.value)} spellCheck={false} /></label>
        <div className="admin-form">
          <input placeholder="Nota do draft" value={note} onChange={(event) => setNote(event.target.value)} />
          <button type="button" onClick={create}>Criar draft</button>
        </div>
      </div>
      <div className="admin-drafts__list">
        {drafts.length ? <ul className="admin-list">{drafts.map((draft) => <li key={draft.id} className={draft.id === selected ? "is-selected" : ""}>
          <button type="button" className="admin-draft-pick" onClick={() => setSelected(draft.id)}>
            <code>{draft.card_id}</code><small>{draft.status}{draft.published_version ? ` → ${draft.published_version}` : ""} · {draft.note || "sem nota"}</small>
          </button>
        </li>)}</ul> : <p className="admin-empty">Nenhum draft ainda — a forja está fria.</p>}
        <div className="admin-form admin-pipeline">
          <button type="button" disabled={!selected} onClick={() => step("validate")}>Validar</button>
          <input className="admin-narrow" value={games} onChange={(event) => setGames(event.target.value)} aria-label="Partidas de simulação" />
          <button type="button" disabled={!selected} onClick={() => step("simulate")}>Simular</button>
          <input placeholder={CURRENT_RULESET} value={version} onChange={(event) => setVersion(event.target.value)} />
          <button type="button" disabled={!selected || !version.trim()} onClick={() => step("publish")}>Publicar</button>
        </div>
        {output && <pre className="admin-output">{output}</pre>}
      </div>
    </div>
    <small>Pipeline: rascunhar → validar (schema + DSL + compilação) → simular (réplicas headless) → publicar versão imutável.</small>
    <Feedback text={feedback} error={isError} />
  </section>;
}
