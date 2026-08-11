import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useSessionStore } from "../store";
import { NytharaBrand } from "./NytharaBrand";
import { Onboarding } from "./Onboarding";
import { UiIcon, type UiIconName } from "./UiIcon";

const navItems: Array<{ to: string; icon: UiIconName; label: string }> = [
  { to: "/app", icon: "home", label: "Início" },
  { to: "/collection", icon: "collection", label: "Coleção" },
  { to: "/champions", icon: "champion", label: "Avatares" },
  { to: "/decks", icon: "deck", label: "Baralho" },
  { to: "/queue", icon: "duel", label: "Jogar" },
  { to: "/arena", icon: "rank", label: "Arena" },
];

export function Shell() {
  const user = useSessionStore((state) => state.user);
  const principal = useSessionStore((state) => state.principal);
  const tokens = useSessionStore((state) => state.tokens);
  const clear = useSessionStore((state) => state.clear);
  const navigate = useNavigate();

  const logout = async () => {
    try {
      if (tokens) await api<void>("/v1/auth/logout", { method: "POST", body: JSON.stringify({ refresh_token: tokens.refresh_token }) });
    } finally {
      clear();
      navigate("/");
    }
  };

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">Pular para o conteúdo</a>
      <aside className="side-nav" aria-label="Navegação principal">
        <NavLink className="brand" to="/app" aria-label="Nythara — início"><NytharaBrand /></NavLink>
        <nav>
          {navItems.map(({ to, icon, label }) => <NavLink key={to} to={to} end={to === "/app"}><UiIcon name={icon} /><span>{label}</span></NavLink>)}
        </nav>
        <div className="side-nav__foot">
          {principal?.role === "admin" && <NavLink to="/salao"><UiIcon name="balance" /><span>LiveOps</span></NavLink>}
          <NavLink to="/tutorial"><UiIcon name="guide" /><span>Tutorial</span></NavLink>
          <NavLink to="/settings"><UiIcon name="settings" /><span>Ajustes</span></NavLink>
          <button type="button" onClick={logout}><UiIcon name="logout" /><span>Sair</span></button>
        </div>
      </aside>
      <div className="shell-body">
        <header className="top-bar">
          <NavLink className="mobile-brand" to="/app" aria-label="Nythara — início"><NytharaBrand /></NavLink>
          <NavLink className="profile-chip" to="/profile"><span className="avatar">{user?.display_name?.slice(0, 1).toUpperCase() ?? "V"}</span><span>{user?.display_name ?? "Viajante"}<small>Perfil</small></span></NavLink>
        </header>
        <main id="main-content" tabIndex={-1}><Outlet /></main>
      </div>
      <nav className="bottom-nav" aria-label="Navegação móvel">
        {navItems.map(({ to, icon, label }) => <NavLink key={to} to={to} end={to === "/app"}><UiIcon name={icon} /><small>{label}</small></NavLink>)}
      </nav>
      <Onboarding />
    </div>
  );
}
