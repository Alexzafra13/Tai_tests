import { useParams } from "react-router";
import { useResource } from "../../hooks";
import type { Test } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";
import { Runner } from "./Runner";
import { Results } from "./Results";
import "./test.css";

// TestPage shows a test in progress, or its results once it is over.
export function TestPage() {
  const { id } = useParams();
  const { data, error, loading, reload } = useResource<Test>(`/tests/${id}`);

  if (loading && !data) return <Loading />;
  if (error || !data) return <ErrorBox message={error ?? "Test no encontrado"} />;
  // Keyed by id so moving to another test (e.g. "retry wrong") resets state.
  if (data.status === "in_progress") return <Runner key={data.id} initial={data} onFinished={reload} />;
  return <Results key={data.id} test={data} />;
}
