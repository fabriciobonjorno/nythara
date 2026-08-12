import { FormEvent, useEffect, useRef, useState } from "react";
import { Link, Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../api";
import { NytharaBrand } from "../components/NytharaBrand";
import { LanguageSelector } from "../components/LanguageSelector";
import { useSessionStore } from "../store";
import type { AuthEnvelope } from "../types";
import { useCards } from "../queries";
import { UiIcon } from "../components/UiIcon";

export function Landing() {
  const tokens = useSessionStore((state) => state.tokens);
  const setAuth = useSessionStore((state) => state.setAuth);
  const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const { data: catalog } = useCards();
	const oauthHandled = useRef(false);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
	const [providers, setProviders] = useState<{ google: boolean }>({ google: false });

	useEffect(() => {
		api<{ google: boolean }>("/v1/auth/providers").then(setProviders).catch(() => undefined);
	}, []);

	useEffect(() => {
		if (oauthHandled.current) return;
		const fragment = new URLSearchParams(window.location.hash.slice(1));
		const ticket = fragment.get("oauth_ticket");
		const oauthError = fragment.get("oauth_error");
		if (!ticket && !oauthError) return;
		oauthHandled.current = true;
		window.history.replaceState(null, "", window.location.pathname + window.location.search);
		if (oauthError) {
			setError("Não foi possível concluir a entrada com Google. Tente novamente.");
			return;
		}
		setBusy(true);
		api<AuthEnvelope>("/v1/auth/oauth/exchange", { method: "POST", body: JSON.stringify({ ticket }) })
			.then((response) => { setAuth(response.user, response.tokens); navigate("/app", { replace: true }); })
			.catch((caught) => setError(caught instanceof Error ? caught.message : "Não foi possível concluir a entrada com Google."))
			.finally(() => setBusy(false));
	}, [navigate, setAuth]);

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
      <header className="landing-header"><a href="#inicio" className="brand" aria-label="Nythara — início"><NytharaBrand /></a><div className="landing-header__tools"><LanguageSelector compact /><span className="alpha-tag">ALPHA · MODO CONFRONTO</span></div></header>
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
		  {searchParams.get("password") === "changed" && <p className="form-success" role="status">Senha atualizada. Entre novamente com a nova credencial.</p>}
          <button className="primary-button" disabled={busy} type="submit">{busy ? "Atravessando…" : mode === "login" ? "Entrar no Véu" : "Criar conta gratuita"}</button>
          {mode === "login" && <Link className="auth-link" to="/forgot-password">Esqueci minha senha</Link>}
          <p className="auth-note">Sem venda de poder. Monte um único baralho, treine contra o bot e entre no confronto quando estiver pronto.</p>
        </form>
      </section>
	  <section className="landing-showcase" id="conheca-o-jogo" aria-labelledby="showcase-title">
		<header><p className="eyebrow">VEJA ANTES DE ENTRAR</p><h2 id="showcase-title">É assim que uma decisão vira confronto.</h2><p>As cartas são reais do catálogo. Na partida, elas viajam até o centro, comparam Assalto e Guarda e resolvem o impacto diante de você.</p></header>
		<div className="landing-duel-preview" aria-label="Prévia visual da arena de Nythara">
		  <div className="preview-player preview-player--rival"><span>RIVAL</span><b>26</b><small>VITALIDADE</small></div>
		  <div className="preview-stage">
			<div className="preview-card preview-card--assault"><img src="/card-art/vr-001.webp" alt="Arte de carta de Assalto" /><span><b>{catalog?.cards.find((card) => card.id === "VR-001")?.name ?? "Corte Rubro"}</b><small>ASSALTO · PODER 6</small></span></div>
			<div className="preview-impact"><UiIcon name="versus" /><strong>CONFRONTO</strong><small>6 ATAQUE × 3 DEFESA</small></div>
			<div className="preview-card preview-card--guard"><img src="/card-art/vr-014.webp" alt="Arte de carta de Guarda" /><span><b>{catalog?.cards.find((card) => card.id === "VR-014")?.name ?? "Guarda do Limiar"}</b><small>GUARDA · PREVENÇÃO 3</small></span></div>
		  </div>
		  <div className="preview-player"><span>VOCÊ</span><b>30</b><small>VITALIDADE</small></div>
		</div>
		<div className="landing-flow"><article><b>1</b><span><strong>Escolha uma carta</strong><small>Sua mão mostra somente jogadas válidas.</small></span></article><article><b>2</b><span><strong>Veja a comparação</strong><small>Assalto e Guarda se encontram no centro.</small></span></article><article><b>3</b><span><strong>Sinta o resultado</strong><small>A carta derrotada se rompe e o dano é aplicado.</small></span></article></div>
		<a className="secondary-button landing-showcase__cta" href="#inicio">Gostei — quero montar meu baralho</a>
	  </section>
      <footer className="landing-footer">NYTHARA · IP ORIGINAL · ALPHA FECHADO</footer>
    </main>
  );
}
		  {providers.google && <><a className="oauth-button" href="/v1/auth/google/start"><span aria-hidden="true">G</span> Continuar com Google</a><div className="auth-divider"><span>ou use e-mail</span></div></>}
