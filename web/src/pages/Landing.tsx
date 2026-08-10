import { FormEvent, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useSessionStore } from "../store";
import type { AuthEnvelope } from "../types";

export function Landing() {
  const tokens = useSessionStore((state) => state.tokens);
  const setAuth = useSessionStore((state) => state.setAuth);
  const navigate = useNavigate();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  if (tokens) return <Navigate to="/app" replace />;

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const body = mode === "register" ? { email, password, display_name: displayName } : { email, password };
      const response = await api<AuthEnvelope>(`/v1/auth/${mode}`, { method: "POST", body: JSON.stringify(body) });
      setAuth(response.user, response.tokens);
      navigate("/app");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Não foi possível entrar.");
    } finally { setBusy(false); }
  };

  return (
    <main className="landing">
      <div className="landing-art" aria-hidden="true" />
      <header className="landing-header"><a href="#inicio" className="brand"><span className="brand-mark">◐</span><span>VÉU<br />RUBRO</span></a><span className="alpha-tag">ALPHA · RULESET 0.3</span></header>
      <section id="inicio" className="landing-copy">
        <p className="eyebrow">UM DUELO SOB DOIS CÉUS</p>
        <h1>Domine o Eclipse.<br /><em>Reescreva o destino.</em></h1>
        <p>Um jogo de cartas competitivo, justo e inteiramente original. Conduza sua Casa, encadeie Sigilos e decida qual luz sobreviverá à rodada.</p>
        <div className="feature-row" aria-label="Características"><span><b>80</b> cartas</span><span><b>10</b> campeões</span><span><b>1v1</b> em tempo real</span></div>
      </section>
      <section className="auth-panel" aria-labelledby="auth-title">
        <div className="auth-tabs" role="tablist">
          <button type="button" role="tab" aria-selected={mode === "login"} onClick={() => setMode("login")}>Entrar</button>
          <button type="button" role="tab" aria-selected={mode === "register"} onClick={() => setMode("register")}>Criar conta</button>
        </div>
        <form onSubmit={submit}>
          <div><p className="eyebrow">ABRA O VÉU</p><h2 id="auth-title">{mode === "login" ? "Retorne ao círculo" : "Escolha seu nome"}</h2></div>
          {mode === "register" && <label>Nome de exibição<input required minLength={2} maxLength={40} autoComplete="nickname" value={displayName} onChange={(event) => setDisplayName(event.target.value)} /></label>}
          <label>E-mail<input required type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label>
          <label>Senha<input required type="password" minLength={12} autoComplete={mode === "login" ? "current-password" : "new-password"} value={password} onChange={(event) => setPassword(event.target.value)} /><small>Mínimo de 12 caracteres.</small></label>
          {error && <p className="form-error" role="alert">{error}</p>}
          <button className="primary-button" disabled={busy} type="submit">{busy ? "Atravessando…" : mode === "login" ? "Entrar no Véu" : "Criar conta gratuita"}</button>
          <p className="auth-note">O Alpha competitivo libera todas as cartas. Vitória vem das decisões, não da compra.</p>
        </form>
      </section>
      <footer className="landing-footer">VÉU RUBRO · IP ORIGINAL · ALPHA FECHADO</footer>
    </main>
  );
}
