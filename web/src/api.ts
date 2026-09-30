export type FieldErrors = Record<string, string>;

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public fields: FieldErrors = {},
  ) {
    super(message);
  }
}

// api performs a JSON request against the Go backend. The session cookie is
// sent automatically because the app is served from the same origin.
export async function api<T>(path: string, init: { method?: string; body?: unknown } = {}): Promise<T> {
  const res = await fetch(`/api${path}`, {
    method: init.method ?? "GET",
    headers: init.body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: init.body !== undefined ? JSON.stringify(init.body) : undefined,
    credentials: "same-origin",
  });
  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new ApiError(res.status, data?.error ?? `Error ${res.status}`, data?.fields ?? {});
  }
  return data as T;
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : "Error inesperado";
}
