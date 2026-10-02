// Per-device preferences in localStorage. Storage can be unavailable
// (private mode, blocked site data), so reads fall back to null and writes
// are best effort: losing a preference must never break the app.

export function loadPref(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function savePref(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // The preference lasts until the page is reloaded.
  }
}
