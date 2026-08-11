import { FormEvent, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { api } from "../api";
import { NytharaBrand } from "../components/NytharaBrand";
import { useSessionStore } from "../store";
import type { AuthEnvelope } from "../types";

export function Landing() {
  const tokens = useSessionStore((state) => state.tokens);
  const setAuth = useSessionStore((state) => state.setAuth);
  const navigate = useNavigate();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  if (tokens) return <Navigate to="/app" replace />;

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const body = mode === "register" ? { email, password, username } : { email, password };
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
      <header className="landing-header"><a href="#inicio" className="brand" aria-label="Nythara — início"><NytharaBrand /></a><span className="alpha-tag">ALPHA · SELOS TÁTICOS 0.10</span></header>
      <section id="inicio" className="landing-copy">
        <img className="landing-hero-logo" src="/assets/nythara-apocalypse-logo.webp" alt="Nythara" />
        <p className="eyebrow">UM CONFRONTO. UMA DECISÃO POR VEZ.</p>
        <h1>Jogue sua carta.<br /><em>Veja o duelo acontecer.</em></h1>
        <p>Assalte, defenda e use Ritos em uma mesa direta. As cartas se encontram no centro, a perdedora se estilhaça e toda escolha pesa na sua Vitalidade.</p>
        <div className="feature-row" aria-label="Características"><span><b>30</b> cartas no baralho</span><span><b>3</b> ações claras</span><span><b>1v1</b> em tempo real</span></div>
      </section>
      <section className="auth-panel" aria-labelledby="auth-title">
        <div className="auth-tabs" role="tablist">
          <button type="button" role="tab" aria-selected={mode === "login"} onClick={() => setMode("login")}>Entrar</button>
          <button type="button" role="tab" aria-selected={mode === "register"} onClick={() => setMode("register")}>Criar conta</button>
        </div>
        <form onSubmit={submit}>
          <div><p className="eyebrow">ABRA O VÉU</p><h2 id="auth-title">{mode === "login" ? "Retorne ao círculo" : "Escolha seu nome"}</h2></div>
          {mode === "register" && <label>Nome de usuário<input required minLength={2} maxLength={32} pattern={"[A-Za-z0-9_\\-]+"} title="Use apenas letras, números, hífen (-) e sublinhado (_), sem espaços." autoComplete="username" autoCapitalize="none" spellCheck={false} value={username} onChange={(event) => setUsername(event.target.value)} /><small>Apenas letras, números, hífen (-) e sublinhado (_), sem espaços.</small></label>}
          <label>E-mail<input required type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label>
          <label>Senha<input required type="password" minLength={12} autoComplete={mode === "login" ? "current-password" : "new-password"} value={password} onChange={(event) => setPassword(event.target.value)} /><small>Mínimo de 12 caracteres.</small></label>
          {error && <p className="form-error" role="alert">{error}</p>}
          <button className="primary-button" disabled={busy} type="submit">{busy ? "Atravessando…" : mode === "login" ? "Entrar no Véu" : "Criar conta gratuita"}</button>
          <p className="auth-note">Sem venda de poder. Monte um único baralho, treine contra o bot e entre no confronto quando estiver pronto.</p>
        </form>
      </section>
      <footer className="landing-footer">NYTHARA · IP ORIGINAL · ALPHA FECHADO</footer>
    </main>
  );
}
