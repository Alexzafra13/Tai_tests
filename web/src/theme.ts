// The colour theme is a per-device preference (dark by default), applied as
// <html data-theme>, which tokens.css reads. public/theme.js applies it
// before the first paint and must use the same key and values.

import { loadPref, savePref } from "./storage";

export type Theme = "dark" | "light" | "auto";
const KEY = "tai.theme";

export const themeLabel: Record<Theme, string> = { dark: "Oscuro", light: "Claro", auto: "Según el sistema" };

export function savedTheme(): Theme {
  const v = loadPref(KEY);
  return v === "light" || v === "auto" ? v : "dark";
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme;
  savePref(KEY, theme);
}
