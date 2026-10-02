import { useEffect, useState } from "react";
import { api } from "../api";
import { useDebounced } from "../hooks";
import type { Source } from "../types";

// Same minimum as content.MinQuoteLength on the server.
export const MIN_QUOTE = 20;

export type QuoteState = "idle" | "checking" | "found" | "missing" | "no-text";

// useQuoteCheck asks the backend whether the quote appears literally in the
// source text, debounced while typing. The backend repeats this check on
// save; this is only early feedback.
export function useQuoteCheck(source: Source | undefined, quote: string): QuoteState {
  const debounced = useDebounced(quote.trim(), 400);
  const [state, setState] = useState<QuoteState>("idle");

  useEffect(() => {
    if (!source || debounced.length < MIN_QUOTE) {
      setState("idle");
      return;
    }
    if (!source.has_text) {
      setState("no-text");
      return;
    }
    let cancelled = false;
    setState("checking");
    api<{ found: boolean }>(`/sources/${source.id}/check-quote`, { method: "POST", body: { quote: debounced } })
      .then((r) => !cancelled && setState(r.found ? "found" : "missing"))
      .catch(() => !cancelled && setState("idle"));
    return () => {
      cancelled = true;
    };
  }, [source, debounced]);

  return state;
}

export function QuoteHint({ state, length, required }: { state: QuoteState; length: number; required: boolean }) {
  switch (state) {
    case "checking":
      return <span className="muted">Comprobando…</span>;
    case "found":
      return <span className="ok">✓ Aparece literalmente en la fuente</span>;
    case "missing":
      return <span className="error">✗ No aparece literalmente en el texto de la fuente</span>;
    case "no-text":
      return <span className="error">La fuente no tiene texto completo; no se puede verificar</span>;
    default:
      if (length > 0 && length < MIN_QUOTE) return <span className="muted">Mínimo {MIN_QUOTE} caracteres</span>;
      return required ? <span className="muted">Obligatoria: se comprueba contra el texto de la fuente</span> : null;
  }
}
