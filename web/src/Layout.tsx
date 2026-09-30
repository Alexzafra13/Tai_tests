import { Link, NavLink, Outlet } from "react-router";
import { useAuth } from "./auth";

const nav = [
  { to: "/", label: "Inicio", end: true },
  { to: "/tests", label: "Tests" },
  { to: "/questions", label: "Preguntas" },
  { to: "/syllabus", label: "Temario" },
];

export function Layout() {
  const { logout } = useAuth();

  return (
    <div className="app">
      <header className="topbar">
        <span className="brand">TAI · Estudio</span>
        <span className="topbar-links">
          <Link to="/settings" className="link">
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
        {nav.map((n) => (
          <NavLink key={n.to} to={n.to} end={n.end}>
            {n.label}
          </NavLink>
        ))}
      </nav>
    </div>
  );
}
