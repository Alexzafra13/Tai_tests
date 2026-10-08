const dateFmt = new Intl.DateTimeFormat("es-ES", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });

export function formatDate(iso: string | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "" : dateFmt.format(d);
}

// formatClock renders seconds as h:mm:ss or m:ss.
export function formatClock(totalSec: number): string {
  const s = Math.max(0, Math.floor(totalSec));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = String(s % 60).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${ss}` : `${m}:${ss}`;
}

export function formatScore(score: number, decimals = 2): string {
  return score.toLocaleString("es-ES", { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
}

const dayFmt = new Intl.DateTimeFormat("es-ES", { day: "numeric", month: "long", year: "numeric", timeZone: "UTC" });

// formatDay renders a calendar date (YYYY-MM-DD) such as "23 de octubre de 2026".
export function formatDay(day: string | undefined): string {
  if (!day) return "";
  const d = new Date(`${day}T00:00:00Z`);
  return Number.isNaN(d.getTime()) ? day : dayFmt.format(d);
}

// lawShortName names a law by what it approves when its official title is
// a resolution or decree "por la que se aprueba…" ("Norma Técnica de
// Interoperabilidad de Documento Electrónico"), and an EU regulation by its
// number and popular name; otherwise it returns "".
export function lawShortName(title: string): string {
  const eu = /^(Reglamento \(UE\) (?:n\.º )?\d+\/\d+) del Parlamento.*?(\([^()]+\))?$/.exec(title);
  if (eu) return eu[2] ? `${eu[1]} ${eu[2]}` : eu[1];
  const m = /, por (?:la|el) que se (?:aprueba|regula) (?:la |el )?(.+)$/.exec(title);
  if (!m || !/^Resolución/.test(title)) return "";
  return m[1].charAt(0).toUpperCase() + m[1].slice(1);
}

// lawOrigin names where a study text comes from: EU regulations (CELEX
// numbers) are EUR-Lex's consolidation, the rest the BOE's.
export function lawOrigin(reference: string): { text: string; site: string } {
  return /^3\d{4}R\d{4}$/.test(reference)
    ? { text: "Texto consolidado de EUR-Lex", site: "eur-lex.europa.eu" }
    : { text: "Texto consolidado del BOE", site: "boe.es" };
}
