import { useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import { ErrorBox, Loading } from "../components/Form";
import { Options, SourceLink } from "../components/QuestionView";
import { useResource } from "../hooks";
import { optionLetters, originLabel, type StudyQuestion } from "../types";
import "./SearchPage.css";

// StudyQuestionPage shows one published question, opened from the search.
// The answer stays hidden until the user picks an option or asks for it.
export function StudyQuestionPage() {
  const { id } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const { data: q, error, loading } = useResource<StudyQuestion>(`/search/${id}`);
  const [chosen, setChosen] = useState<number | null>(null);
  const [open, setOpen] = useState(false);
  const answered = open || chosen !== null;

  // Going back keeps the search query, which lives in the previous URL.
  const fromApp = location.key !== "default";

  return (
    <>
      <div className="page-head">
        <h2>Pregunta</h2>
        {fromApp ? (
          <button type="button" className="link" onClick={() => navigate(-1)}>
            Volver
          </button>
        ) : (
          <Link to="/search">Buscar</Link>
        )}
      </div>
      <ErrorBox message={error} />
      {loading && <Loading />}
      {q && (
        <article className="question study-question">
          <p className="stem">{q.stem}</p>
          <Options
            options={q.options}
            chosen={chosen}
            correct={answered ? q.correct : undefined}
            onChoose={answered ? undefined : setChosen}
          />
          {answered ? (
            <div className="solution">
              {chosen === null ? (
                <strong>Correcta: {optionLetters[q.correct]}</strong>
              ) : chosen === q.correct ? (
                <strong className="ok">✓ Correcta</strong>
              ) : (
                <strong className="error">
                  ✗ Incorrecta · marcaste {optionLetters[chosen]}, era {optionLetters[q.correct]}
                </strong>
              )}
              {q.explanation && <p>{q.explanation}</p>}
              {q.source_quote && (
                <div className="source">
                  <span className="muted small">Cita de la fuente</span>
                  <blockquote>{q.source_quote}</blockquote>
                </div>
              )}
            </div>
          ) : (
            <button type="button" className="link" onClick={() => setOpen(true)}>
              Ver respuesta
            </button>
          )}

          <section className="card study-question-source">
            <h3>Fuente</h3>
            <p className="muted small">{originLabel[q.origin]}</p>
            <p>
              {q.source_title}
              <span className="muted"> · {q.source_ref}</span>
            </p>
            <SourceLink kind={q.source_kind} url={q.source_url} />
          </section>

          {q.topics.length > 0 && (
            <section className="card study-question-source">
              <h3>{q.topics.length === 1 ? "Tema" : "Temas"}</h3>
              <ul className="study-question-topics">
                {q.topics.map((t) => (
                  <li key={t.id}>
                    <span>
                      {t.number}. {t.title}
                    </span>
                    <Link to={`/tests/new?topic=${t.id}`} className="small">
                      Practicar
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </article>
      )}
    </>
  );
}
