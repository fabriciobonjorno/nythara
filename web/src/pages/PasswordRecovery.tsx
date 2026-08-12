import { FormEvent, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api } from "../api";
import { LanguageSelector } from "../components/LanguageSelector";
import { NytharaBrand } from "../components/NytharaBrand";
import { usePreferencesStore } from "../store";

function RecoveryShell({ children }: { children: React.ReactNode }) {
  return <main className="landing auth-standalone">
    <div className="landing-art" aria-hidden="true" />
    <header className="landing-header"><Link to="/" className="brand" aria-label="Nythara — início"><NytharaBrand /></Link><div className="landing-header__tools"><LanguageSelector compact /><span className="alpha-tag">ALPHA · MODO CONFRONTO</span></div></header>
    {children}
    <footer className="landing-footer">NYTHARA · IP ORIGINAL · ALPHA FECHADO</footer>
  </main>;
}

export function ForgotPasswordPage() {
  const locale = usePreferencesStore((state) => state.locale);
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");

  const submit = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setError("");
    try {
      await api<void>("/v1/auth/forgot-password", { method: "POST", body: JSON.stringify({ email, locale }) });
      setSent(true);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Não foi possível enviar o link agora.");
    } finally { setBusy(false); }
  };

  return <RecoveryShell><section className="auth-panel recovery-panel" aria-labelledby="recovery-title">
    <form onSubmit={submit}>
      <div><p className="eyebrow">RECUPERE O ACESSO</p><h1 id="recovery-title">Esqueceu sua senha?</h1></div>
      {sent ? <>
        <p className="form-success" role="status">Se existir uma conta com esse e-mail, você receberá um link válido por 30 minutos.</p>
        <Link className="primary-button" to="/">Voltar para entrar</Link>
      </> : <>
        <p className="recovery-copy">Informe seu e-mail. Por segurança, a resposta será a mesma mesmo que não exista uma conta cadastrada.</p>
        <label>E-mail<input required type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button className="primary-button" disabled={busy} type="submit">{busy ? "Enviando…" : "Enviar link de recuperação"}</button>
        <Link className="auth-link" to="/">Voltar para entrar</Link>
      </>}
    </form>
  </section></RecoveryShell>;
}

export function ResetPasswordPage() {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState(token ? "" : "Este link é inválido ou expirou.");

  const submit = async (event: FormEvent) => {
    event.preventDefault(); setError("");
    if (password !== confirmation) { setError("As senhas não coincidem."); return; }
    setBusy(true);
    try {
      await api<void>("/v1/auth/reset-password", { method: "POST", body: JSON.stringify({ token, password }) });
      setDone(true);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Não foi possível redefinir a senha.");
    } finally { setBusy(false); }
  };

  return <RecoveryShell><section className="auth-panel recovery-panel" aria-labelledby="reset-title">
    <form onSubmit={submit}>
      <div><p className="eyebrow">NOVO PACTO</p><h1 id="reset-title">Crie uma nova senha</h1></div>
      {done ? <>
        <p className="form-success" role="status">Senha redefinida. Todas as sessões anteriores foram encerradas.</p>
        <Link className="primary-button" to="/">Entrar com a nova senha</Link>
      </> : <>
        <label>Nova senha<input required disabled={!token} type="password" minLength={12} maxLength={256} autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} /><small>Mínimo de 12 caracteres.</small></label>
        <label>Confirme a nova senha<input required disabled={!token} type="password" minLength={12} maxLength={256} autoComplete="new-password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} /></label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button className="primary-button" disabled={busy || !token} type="submit">{busy ? "Redefinindo…" : "Redefinir senha"}</button>
        <Link className="auth-link" to="/forgot-password">Solicitar outro link</Link>
      </>}
    </form>
  </section></RecoveryShell>;
}
