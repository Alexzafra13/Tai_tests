import { Link, NavLink, Outlet } from "react-router";
import { useAuth } from "./auth";

type NavItem = { to: string; label: string; end?: boolean; adminOnly?: boolean };

const nav: NavItem[] = [
  { to: "/", label: "Inicio", end: true },
  { to: "/tests", label: "Tests" },
  { to: "/review", label: "Revisión", adminOnly: true },
  { to: "/questions", label: "Preguntas", adminOnly: true },
  { to: "/syllabus", label: "Temario" },
];

export function Layout() {
  const { user, isAdmin, logout } = useAuth();

  return (
    <div className="app">
      <header className="topbar">
        <span className="brand">TAI · Estudio</span>
        <span className="topbar-links">
          <Link to="/settings" className="link" title={user?.username}>
            Ajustes
          </Link>
          <button className="link" onClick={() => logout()}>
            Salir
          </button>
        </span>
      </header>
      <main className="content">
        <Outlet />
      </main>
      <nav className="bottomnav">
        {nav
          .filter((n) => isAdmin || !n.adminOnly)
          .map((n) => (
            <NavLink key={n.to} to={n.to} end={n.end}>
              {n.label}
            </NavLink>
          ))}
      </nav>
    </div>
  );
}
