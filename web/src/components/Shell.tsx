import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useSessionStore } from "../store";

const navItems = [
  ["/app", "⌂", "Início"],
  ["/collection", "▧", "Coleção"],
  ["/champions", "♙", "Campeões"],
  ["/decks", "▤", "Decks"],
  ["/queue", "⚔", "Jogar"],
];

export function Shell() {
  const user = useSessionStore((state) => state.user);
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
        <NavLink className="brand" to="/app" aria-label="Véu Rubro — início"><span className="brand-mark">◐</span><span>VÉU<br />RUBRO</span></NavLink>
        <nav>
          {navItems.map(([to, icon, label]) => <NavLink key={to} to={to} end={to === "/app"}><span aria-hidden="true">{icon}</span><span>{label}</span></NavLink>)}
        </nav>
        <div className="side-nav__foot">
          <NavLink to="/tutorial"><span aria-hidden="true">?</span><span>Tutorial</span></NavLink>
          <NavLink to="/settings"><span aria-hidden="true">⚙</span><span>Ajustes</span></NavLink>
          <button type="button" onClick={logout}><span aria-hidden="true">↪</span><span>Sair</span></button>
        </div>
      </aside>
      <div className="shell-body">
        <header className="top-bar">
          <NavLink className="mobile-brand" to="/app">◐ VÉU RUBRO</NavLink>
          <NavLink className="profile-chip" to="/profile"><span className="avatar">{user?.display_name?.slice(0, 1).toUpperCase() ?? "V"}</span><span>{user?.display_name ?? "Viajante"}<small>Perfil</small></span></NavLink>
        </header>
        <main id="main-content" tabIndex={-1}><Outlet /></main>
      </div>
      <nav className="bottom-nav" aria-label="Navegação móvel">
        {navItems.map(([to, icon, label]) => <NavLink key={to} to={to} end={to === "/app"}><span aria-hidden="true">{icon}</span><small>{label}</small></NavLink>)}
      </nav>
    </div>
  );
}
