import { Link, NavLink, Outlet } from "react-router";
import { useAuth } from "./auth";

type NavItem = { to: string; label: string; end?: boolean };

// Five destinations fit the thumb bar; the rest is reached from Inicio.
const userNav: NavItem[] = [
  { to: "/", label: "Inicio", end: true },
  { to: "/tests", label: "Tests" },
  { to: "/syllabus", label: "Temario" },
  { to: "/stats", label: "Progreso" },
  { to: "/search", label: "Buscar" },
];

const adminNav: NavItem[] = [
  { to: "/", label: "Inicio", end: true },
  { to: "/tests", label: "Tests" },
  { to: "/syllabus", label: "Temario" },
  { to: "/review", label: "Revisión" },
  { to: "/questions", label: "Preguntas" },
];

export function Layout() {
  const { user, isAdmin } = useAuth();
  const name = user?.display_name || user?.username || "";

  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">
          TAI<span>Estudio</span>
        </Link>
        <Link to="/settings" className="avatar" title={`${name} · Ajustes`} aria-label="Ajustes">
          {name.slice(0, 1)}
        </Link>
      </header>
      <main className="content">
        <Outlet />
      </main>
      <nav className="bottomnav">
        {(isAdmin ? adminNav : userNav).map((n) => (
          <NavLink key={n.to} to={n.to} end={n.end}>
            {n.label}
          </NavLink>
        ))}
      </nav>
    </div>
  );
}
