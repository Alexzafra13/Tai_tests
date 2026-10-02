// The colour theme is a per-device preference: dark by default (the app's
// "study lamp" look), light, or follow the system. It lives on <html> as
// data-theme, which tokens.css reads.

export type Theme = "dark" | "light" | "auto";
const KEY = "tai.theme";

export const themeLabel: Record<Theme, string> = { dark: "Oscuro", light: "Claro", auto: "Según el sistema" };

export function savedTheme(): Theme {
  try {
    const v = localStorage.getItem(KEY);
    if (v === "dark" || v === "light" || v === "auto") return v;
  } catch {
    // Storage unavailable: use the default.
  }
  return "dark";
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme;
  try {
    localStorage.setItem(KEY, theme);
  } catch {
    // Not critical: the choice lasts until the page is reloaded.
  }
}
