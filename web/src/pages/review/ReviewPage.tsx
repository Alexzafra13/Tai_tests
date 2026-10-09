import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { api, errorMessage } from "../../api";
import { useResource } from "../../hooks";
import type { Block, Page, ReviewCounts, ReviewItem, ReviewKind, ReviewState } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";
import { Toast, type ToastMessage } from "../../components/Toast";
import { NoteReports } from "./NoteReports";
import { ReviewCard } from "./ReviewCard";
import "./review.css";

const tabs: { kind: ReviewKind; label: string; count: (c: ReviewCounts) => number }[] = [
  { kind: "", label: "Todas", count: (c) => c.total },
  { kind: "reported", label: "Dudosas", count: (c) => c.reported },
  { kind: "drafts", label: "Borradores", count: (c) => c.drafts },
];

// ReviewPage walks the queue one question at a time. Decided questions
// leave the queue, so the next one is always at the current offset;
// "Saltar" moves the offset past questions left for later.
export function ReviewPage() {
  const [kind, setKind] = useState<ReviewKind>("");
  const [skipped, setSkipped] = useState(0);
  const [topicIds, setTopicIds] = useState<number[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<ToastMessage | null>(null);
  const [busy, setBusy] = useState(false);

  const counts = useResource<ReviewCounts>("/review/counts");
  const queue = useResource<Page<ReviewItem>>(`/review?kind=${kind}&offset=${skipped}&limit=1`);
  const { data: blocks } = useResource<Block[]>("/syllabus");
  const item = queue.data?.items[0];

  useEffect(() => {
    setTopicIds(item?.topic_ids ?? []);
    setError(null);
  }, [item]);

  const { reload: reloadCounts } = counts;
  const { reload: reloadQueue } = queue;
  const refresh = useCallback(() => {
    reloadCounts();
    reloadQueue();
  }, [reloadCounts, reloadQueue]);
  const closeToast = useCallback(() => setToast(null), []);

  function selectKind(k: ReviewKind) {
    setKind(k);
    setSkipped(0);
  }

  async function decide(action: "accept" | "discard") {
    if (!item || busy) return;
    setBusy(true);
    setError(null);
    try {
      const body = action === "accept" ? { topic_ids: topicIds } : undefined;
      const { previous } = await api<{ previous: ReviewState }>(`/review/${item.id}/${action}`, { method: "POST", body });
      const id = item.id;
      setToast({
        text: action === "accept" ? "Aceptada: ya sale en los tests" : "Descartada",
        onAction: async () => {
          await api(`/review/${id}/restore`, { method: "POST", body: previous }).catch((e) => setError(errorMessage(e)));
          refresh();
        },
      });
      refresh();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  const total = queue.data?.total ?? 0;

  return (
    <div className="review">
      <div className="page-head">
        <h2>Revisión</h2>
        <Link to="/review/batch" className="link">
          Aceptar en bloque
        </Link>
      </div>

      <NoteReports />

      <div className="chips">
        {tabs.map((t) => (
          <button key={t.kind} type="button" className={kind === t.kind ? "chip active" : "chip"} onClick={() => selectKind(t.kind)}>
            {t.label} ({counts.data ? t.count(counts.data) : "…"})
          </button>
        ))}
      </div>

      <ErrorBox message={error ?? queue.error} />
      {queue.loading && !queue.data && <Loading />}

      {queue.data && !item && (
        <div className="card empty">
          {skipped > 0 ? (
            <>
              <p>Has saltado {skipped}. No quedan más por delante.</p>
              <button type="button" onClick={() => setSkipped(0)}>
                Volver a las saltadas
              </button>
            </>
          ) : (
            <p>Nada pendiente de revisar.</p>
          )}
        </div>
      )}

      {item && blocks && (
        <>
          <p className="muted small">
            {skipped + 1} de {total}
          </p>
          <ReviewCard item={item} blocks={blocks} topicIds={topicIds} onTopicsChange={setTopicIds} />
          <div className="actions sticky review-actions">
            <button type="button" className="danger" disabled={busy} onClick={() => decide("discard")}>
              Descartar
            </button>
            <button type="button" disabled={busy} onClick={() => setSkipped((s) => s + 1)}>
              Saltar
            </button>
            <button type="button" className="primary" disabled={busy || topicIds.length === 0} onClick={() => decide("accept")}>
              Aceptar
            </button>
          </div>
        </>
      )}

      <Toast toast={toast} onClose={closeToast} />
    </div>
  );
}
