import { useState } from "react";
import { useNavigate } from "react-router";
import { api, errorMessage } from "../../api";
import { ErrorBox, Loading } from "../../components/Form";
import { useResource } from "../../hooks";
import type { CreateTestInput, Exam, ExamPart, ScoringSettings, TestMode } from "../../types";
import { emptyFilters, examMinutes } from "../test/start";
import "./exams.css";

// ExamsPage lists the official INAP exams so they can be taken as they were
// set: the first part plus one practical case, in the original order.
export function ExamsPage() {
  const { data: exams, error, loading } = useResource<Exam[]>("/exams");
  const scoring = useResource<ScoringSettings>("/settings/scoring");
  const penalty = scoring.data?.default_penalty ?? 1 / 3;

  return (
    <>
      <div className="page-head">
        <h2>Exámenes oficiales</h2>
      </div>
      <p className="muted">
        Cuestionarios del INAP con la plantilla definitiva. Como en el examen, haces la primera parte y uno de los
        supuestos, y las preguntas de reserva sustituyen a las anuladas.
      </p>
      <ErrorBox message={error} />
      {loading && <Loading />}
      {exams?.length === 0 && <p className="muted">Todavía no hay exámenes oficiales publicados.</p>}
      {exams?.map((e) => <ExamCard key={e.id} exam={e} penalty={penalty} />)}
    </>
  );
}

function ExamCard({ exam, penalty }: { exam: Exam; penalty: number }) {
  const navigate = useNavigate();
  const first = exam.parts.filter((p) => !p.case);
  const cases = exam.parts.filter((p) => p.case);
  const [caseName, setCaseName] = useState(cases[0]?.name ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const parts = [...first, ...cases.filter((p) => p.name === caseName)];
  const ids = parts.flatMap((p) => p.question_ids);
  const replaced = sum(parts, (p) => p.replaced);
  const missing = sum(parts, (p) => p.missing);

  async function start(mode: TestMode) {
    setBusy(true);
    setError(null);
    const body: CreateTestInput = {
      mode,
      count: ids.length,
      penalty,
      time_limit_min: mode === "exam" ? examMinutes(ids.length) : 0,
      filters: { ...emptyFilters, question_ids: ids, ordered: true },
    };
    try {
      const { id } = await api<{ id: number }>("/tests", { method: "POST", body });
      navigate(`/tests/${id}`);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  }

  return (
    <section className="card exam">
      <h3>{exam.title}</h3>
      <ErrorBox message={error} />
      <ul className="exam-parts">
        {first.map((p) => (
          <li key={p.name}>
            <span>{p.name}</span>
            <span className="count">{p.question_ids.length}</span>
          </li>
        ))}
      </ul>
      {cases.length > 0 && (
        <div className="segmented" role="group" aria-label="Supuesto práctico">
          {cases.map((p) => (
            <button
              key={p.name}
              type="button"
              className={p.name === caseName ? "active" : ""}
              onClick={() => setCaseName(p.name)}
            >
              {p.name} · {p.question_ids.length}
            </button>
          ))}
        </div>
      )}
      <p className="hint">
        {ids.length} preguntas · {examMinutes(ids.length)} min
        {replaced > 0 && ` · ${replaced} ${replaced === 1 ? "sustituida" : "sustituidas"} por reservas`}
        {missing > 0 && ` · ${missing} sin poder responder`}
      </p>
      <div className="actions">
        <button className="primary" disabled={busy} onClick={() => start("exam")}>
          Simulacro
        </button>
        <button disabled={busy} onClick={() => start("practice")}>
          Practicar
        </button>
      </div>
      {exam.url && (
        <a className="small" href={exam.url} target="_blank" rel="noreferrer">
          Convocatoria en el INAP
        </a>
      )}
    </section>
  );
}

const sum = (parts: ExamPart[], f: (p: ExamPart) => number) => parts.reduce((n, p) => n + f(p), 0);
