import { NavLink, Outlet } from "react-router";
import { useAuth } from "./auth";

const nav = [
  { to: "/", label: "Inicio", end: true },
  { to: "/tests", label: "Tests" },
  { to: "/questions", label: "Preguntas" },
  { to: "/sources", label: "Fuentes" },
  { to: "/syllabus", label: "Temario" },
];

export function Layout() {
  const { logout } = useAuth();

  return (
    <div className="app">
      <header className="topbar">
        <span className="brand">TAI · Estudio</span>
        <button className="link" onClick={() => logout()}>
          Salir
        </button>
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
