import { FormEvent, useEffect, useRef, useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { api } from "../api";
import { NytharaBrand } from "../components/NytharaBrand";
import { LanguageSelector } from "../components/LanguageSelector";
import { needsFirstLoginTutorial, usePreferencesStore, useSessionStore } from "../store";
import type { AuthEnvelope } from "../types";
import { useCards } from "../queries";
import { UiIcon } from "../components/UiIcon";
import { releaseTextEntryFocus } from "../mobileViewport";

export function Landing() {
  const tokens = useSessionStore((state) => state.tokens);
  const sessionUser = useSessionStore((state) => state.user);
  const setAuth = useSessionStore((state) => state.setAuth);
  const navigate = useNavigate();
	const location = useLocation();
	const { data: catalog } = useCards();
	const oauthHandled = useRef(false);
	const enteringTutorial = useRef(false);
	const adminInvite = new URLSearchParams(location.search).get("admin_invite")?.trim() ?? "";
  const [mode, setMode] = useState<"login" | "register">(() => adminInvite ? "register" : "login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
	const [providers, setProviders] = useState<{ google: boolean }>({ google: false });
	const [passwordChanged] = useState(() => {
		if ((location.state as { passwordChanged?: boolean } | null)?.passwordChanged) return true;
		return sessionStorage.getItem("nythara-password-changed") === "1";
	});
	const [accountDeactivated] = useState(() => Boolean((location.state as { accountDeactivated?: boolean } | null)?.accountDeactivated) ||
		sessionStorage.getItem("nythara-account-deactivated") === "1");
	useEffect(() => { if (accountDeactivated) sessionStorage.removeItem("nythara-account-deactivated"); }, [accountDeactivated]);
	useEffect(() => { if (passwordChanged) sessionStorage.removeItem("nythara-password-changed"); }, [passwordChanged]);

	const enterApp = (response: AuthEnvelope, forceTutorial = false) => {
		releaseTextEntryFocus();
		if (response.user.role !== "player") {
			enteringTutorial.current = false;
			setAuth(response.user, response.tokens);
			navigate("/salao", { replace: true });
			return;
		}
		const firstLogin = forceTutorial || needsFirstLoginTutorial(response.user.id);
		enteringTutorial.current = firstLogin;
		if (firstLogin) usePreferencesStore.getState().beginTutorial(response.user.id);
		setAuth(response.user, response.tokens);
		navigate(firstLogin ? "/tutorial" : "/app", { replace: true });
	};

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
			.then((response) => enterApp(response))
			.catch((caught) => setError(caught instanceof Error ? caught.message : "Não foi possível concluir a entrada com Google."))
			.finally(() => setBusy(false));
	}, [navigate, setAuth]);

  if (tokens) return <Navigate to={sessionUser?.role !== "player" ? "/salao"
		: enteringTutorial.current || (sessionUser && needsFirstLoginTutorial(sessionUser.id)) ? "/tutorial" : "/app"} replace />;

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const body = mode === "register" ? { email, password, username, ...(adminInvite ? { admin_invite: adminInvite } : {}) } : { email, password };
      const response = await api<AuthEnvelope>(`/v1/auth/${mode}`, { method: "POST", body: JSON.stringify(body) });
      enterApp(response, mode === "register");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Não foi possível entrar.");
    } finally { setBusy(false); }
  };

  return (
    <main className="landing landing--public">
      <div className="landing-hero">
        <div className="landing-art" aria-hidden="true" />
        <header className="landing-header"><a href="#inicio" className="brand" aria-label="Nythara — início"><NytharaBrand /></a><div className="landing-header__tools"><LanguageSelector compact /><span className="alpha-tag">ALPHA · MODO CONFRONTO</span></div></header>
        <section id="inicio" className="landing-copy">
        <img className="landing-hero-logo" src="/assets/nythara-apocalypse-logo.webp" width="817" height="413" alt="Nythara — jogo de cartas online PvP" />
        <p className="eyebrow">UM CONFRONTO. UMA DECISÃO POR VEZ.</p>
		<h1><span>Nythara: jogo de cartas online PvP.</span> Jogue sua carta.<br /><em>Veja o duelo acontecer.</em></h1>
		<p>Monte um baralho de 30 cartas e dispute duelos estratégicos 1v1 em tempo real. Assalte, defenda e use Ritos em uma mesa direta: as cartas se encontram no centro, a perdedora se estilhaça e toda escolha pesa na sua Vitalidade.</p>
		<div className="feature-row" aria-label="Características"><span><b>30</b> cartas no baralho</span><span><b>3</b> ações claras</span><span><b>1v1</b> em tempo real</span></div>
		  <a className="landing-glimpse" href="#conheca-o-jogo"><span><img src="/card-art/vr-001.webp" width="640" height="960" alt="" /><img src="/card-art/vr-014.webp" width="640" height="960" alt="" /><img src="/card-art/vr-076.webp" width="640" height="960" alt="" /></span><strong>Ver a mesa, as cartas e o duelo antes de criar conta <UiIcon name="arrow-right" /></strong></a>
        </section>
        <section className="auth-panel" aria-labelledby="auth-title">
		{adminInvite && <div className="admin-invite-banner"><strong>Convite administrativo</strong><span>Crie a conta com o mesmo e-mail para aceitar este convite de uso único.</span></div>}
        <div className="auth-tabs" role="tablist">
          <button type="button" role="tab" aria-selected={mode === "login"} onClick={() => setMode("login")}>Entrar</button>
          <button type="button" role="tab" aria-selected={mode === "register"} onClick={() => setMode("register")}>Criar conta</button>
        </div>
		<form onSubmit={submit}>
          <div><p className="eyebrow">ABRA O VÉU</p><h2 id="auth-title">{mode === "login" ? "Retorne ao círculo" : "Escolha seu nome"}</h2></div>
          {providers.google && !adminInvite && <><a className="oauth-button" href="/v1/auth/google/start"><span aria-hidden="true">G</span> Continuar com Google</a><div className="auth-divider"><span>ou use e-mail</span></div></>}
          {mode === "register" && <label>Nome de usuário<input required minLength={2} maxLength={32} pattern={"[A-Za-z0-9_\\-]+"} title="Use apenas letras, números, hífen (-) e sublinhado (_), sem espaços." autoComplete="username" autoCapitalize="none" spellCheck={false} value={username} onChange={(event) => setUsername(event.target.value)} /><small>Apenas letras, números, hífen (-) e sublinhado (_), sem espaços.</small></label>}
          <label>E-mail<input required type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label>
          <label>Senha<input required type="password" minLength={12} autoComplete={mode === "login" ? "current-password" : "new-password"} value={password} onChange={(event) => setPassword(event.target.value)} /><small>Mínimo de 12 caracteres.</small></label>
		  {error && <p className="form-error" role="alert">{error}</p>}
		  {passwordChanged && <p className="form-success" role="status">Senha atualizada. Entre novamente com a nova credencial.</p>}
		  {accountDeactivated && <p className="form-success" role="status">Conta desativada. Quando quiser voltar, entre novamente com seus dados.</p>}
          <button className="primary-button" disabled={busy} type="submit">{busy ? "Atravessando…" : mode === "login" ? "Entrar no Véu" : "Criar conta gratuita"}</button>
          {mode === "login" && <Link className="auth-link" to="/forgot-password">Esqueci minha senha</Link>}
          <p className="auth-note">Sem venda de poder. Monte um único baralho, treine contra o adversário virtual e entre no confronto quando estiver pronto.</p>
		  </form>
        </section>
      </div>
	  <section className="landing-showcase" id="conheca-o-jogo" aria-labelledby="showcase-title">
		<header><p className="eyebrow">VEJA ANTES DE ENTRAR</p><h2 id="showcase-title">Como funciona o duelo de cartas.</h2><p>As cartas são reais do catálogo. Na partida, elas viajam até o centro, comparam Assalto e Guarda e resolvem o impacto diante de você.</p></header>
		<div className="landing-duel-preview" aria-label="Prévia visual da arena de Nythara">
		  <div className="preview-player preview-player--rival"><span>RIVAL</span><b>26</b><small>VITALIDADE</small></div>
		  <div className="preview-stage">
			<div className="preview-card preview-card--assault"><img src="/card-art/vr-001.webp" width="640" height="960" loading="lazy" alt="Carta de Assalto Corte Rubro" /><span><b>{catalog?.cards.find((card) => card.id === "VR-001")?.name ?? "Corte Rubro"}</b><small>ASSALTO · PODER 6</small></span></div>
			<div className="preview-impact"><UiIcon name="versus" /><strong>CONFRONTO</strong><small>6 ATAQUE × 3 DEFESA</small></div>
			<div className="preview-card preview-card--guard"><img src="/card-art/vr-014.webp" width="640" height="960" loading="lazy" alt="Carta de Guarda do Limiar" /><span><b>{catalog?.cards.find((card) => card.id === "VR-014")?.name ?? "Guarda do Limiar"}</b><small>GUARDA · PREVENÇÃO 3</small></span></div>
		  </div>
		  <div className="preview-player"><span>VOCÊ</span><b>30</b><small>VITALIDADE</small></div>
		</div>
		<div className="landing-flow"><article><b>1</b><span><strong>Escolha uma carta</strong><small>Sua mão mostra somente jogadas válidas.</small></span></article><article><b>2</b><span><strong>Veja a comparação</strong><small>Assalto e Guarda se encontram no centro.</small></span></article><article><b>3</b><span><strong>Sinta o resultado</strong><small>A carta derrotada se rompe e o dano é aplicado.</small></span></article></div>
		<a className="secondary-button landing-showcase__cta" href="#inicio">Gostei — quero montar meu baralho</a>
	  </section>
	  <section className="landing-discovery" aria-labelledby="discovery-title">
		<header><p className="eyebrow">JOGUE NO NAVEGADOR</p><h2 id="discovery-title">Um jogo de cartas competitivo, gratuito e original.</h2><p>Nythara reúne construção de baralho, treino contra adversário virtual e PvP em tempo real em uma experiência direta para computador e celular.</p></header>
		<div><article><h3>Monte seu baralho</h3><p>Escolha 30 cartas entre Assaltos, Guardas e Ritos. Cada tipo ocupa um papel claro durante o confronto.</p></article><article><h3>Treine sem espera</h3><p>Teste suas decisões contra o adversário virtual antes de buscar um rival na Arena competitiva.</p></article><article><h3>Dispute duelos 1v1</h3><p>Enfrente jogadores reais, acompanhe sua progressão e reveja o histórico das partidas.</p></article></div>
	  </section>
      <footer className="landing-footer">NYTHARA · IP ORIGINAL · ALPHA FECHADO</footer>
    </main>
  );
}
