import { Link } from "react-router";
import { useResource } from "../../hooks";
import type { Page, Question, ReviewCounts } from "../../types";

// AdminTiles is only rendered for administrators, so these admin-only
// endpoints are never requested by other users.
export function AdminTiles() {
  const review = useResource<ReviewCounts>("/review/counts");
  const published = useResource<Page<Question>>("/questions?status=published&limit=1");

  let reviewText = "Borradores y dudas";
  if (review.data) {
    reviewText =
      review.data.total === 0 ? "Nada pendiente" : `${review.data.total} pendientes · ${review.data.reported} dudosas`;
  }

  return (
    <section className="home-section">
      <h3>Administración</h3>
      <ul className="tiles">
        <li>
          <Link to="/review" className="card tile">
            <strong>Revisión</strong>
            <span className="muted small">{reviewText}</span>
          </Link>
        </li>
        <li>
          <Link to="/questions" className="card tile">
            <strong>Preguntas</strong>
            <span className="muted small">
              {published.data ? `${published.data.total} publicadas` : "Banco de preguntas"}
            </span>
          </Link>
        </li>
        <li>
          <Link to="/stats" className="card tile">
            <strong>Progreso</strong>
            <span className="muted small">Tus estadísticas</span>
          </Link>
        </li>
        <li>
          <Link to="/search" className="card tile">
            <strong>Buscar</strong>
            <span className="muted small">En todas las preguntas</span>
          </Link>
        </li>
        <li>
          <Link to="/users" className="card tile">
            <strong>Usuarios</strong>
            <span className="muted small">Cuentas y permisos</span>
          </Link>
        </li>
      </ul>
    </section>
  );
}
