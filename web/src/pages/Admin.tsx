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

export function AdminPage() {
  const principal = useSessionStore((state) => state.principal);
  if (principal?.role !== "admin") {
    return <main className="page admin-page">
      <header className="admin-head">
        <p className="eyebrow">ACESSO RESTRITO</p>
        <h1>Salão de Controle</h1>
        <p className="admin-empty">Esta área é exclusiva para contas com o papel <code>admin</code>.</p>
      </header>
    </main>;
  }
  return <div className="page admin-page">
    <header className="admin-head"><p className="eyebrow">SALÃO DE CONTROLE</p><h1>LiveOps</h1>
      <p className="admin-sub">Toda mudança é idempotente e auditada no servidor — sem auditoria, sem mudança.</p></header>
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
