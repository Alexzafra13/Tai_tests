// Applies the saved colour theme before the first paint, so a light theme
// does not flash dark while the app loads. Kept in sync with src/theme.ts.
try {
  var t = localStorage.getItem("tai.theme");
  if (t === "dark" || t === "light" || t === "auto") document.documentElement.dataset.theme = t;
} catch (e) {}
