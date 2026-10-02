import { createHash } from "node:crypto";
import { readdirSync, readFileSync } from "node:fs";
import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";

// serviceWorker emits /sw.js from the sw.js template with the list of files
// to precache. Fonts are limited to the Latin subsets the app uses; the rest
// are cached the first time the browser needs them.
function serviceWorker(): Plugin {
  return {
    name: "tai-service-worker",
    apply: "build",
    generateBundle(_, bundle) {
      const built = Object.keys(bundle).filter(
        (f) => /\.(js|css)$/.test(f) || (f.endsWith(".woff2") && f.includes("-latin-")),
      );
      const pub = readdirSync("public").filter((f) => f !== "screenshots");
      const precache = ["/", ...[...built, ...pub].sort().map((f) => `/${f}`)];
      const version = createHash("sha256").update(precache.join("\n")).digest("hex").slice(0, 12);
      this.emitFile({
        type: "asset",
        fileName: "sw.js",
        source: readFileSync("sw.js", "utf8")
          .replace("__VERSION__", version)
          .replace("__PRECACHE__", JSON.stringify(precache)),
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), serviceWorker()],
  server: {
    // In development the Go server runs on :8080 and Vite proxies the API.
    proxy: { "/api": "http://localhost:8080" },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
