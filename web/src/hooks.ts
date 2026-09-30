import { useCallback, useEffect, useState } from "react";
import { api, errorMessage } from "./api";

export type Resource<T> = {
  data: T | undefined;
  error: string | null;
  loading: boolean;
  reload: () => void;
};

// useResource fetches an API path and refetches whenever it changes. Pass
// null to skip fetching (e.g. a "new" form with nothing to load).
export function useResource<T>(path: string | null): Resource<T> {
  const [data, setData] = useState<T>();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(path !== null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    if (path === null) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    api<T>(path)
      .then((d) => {
        if (!cancelled) {
          setData(d);
          setError(null);
        }
      })
      .catch((err) => !cancelled && setError(errorMessage(err)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [path, tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, error, loading, reload };
}

// useDebounced returns value once it has stopped changing for delay ms.
export function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}

// useCountdown ticks down from a number of seconds given by the server. It
// anchors on a local deadline so a phone waking from sleep shows the right
// time. Returns undefined when there is no limit.
export function useCountdown(initialSec: number | undefined): number | undefined {
  const [deadline] = useState(() => (initialSec === undefined ? undefined : Date.now() + initialSec * 1000));
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (deadline === undefined) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [deadline]);
  if (deadline === undefined) return undefined;
  return Math.max(0, Math.round((deadline - now) / 1000));
}
