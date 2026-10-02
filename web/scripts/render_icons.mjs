// Renders the PNG icons (Android, maskable, Apple) from public/icon.svg.
// Run from web/:  npm i --no-save playwright-core && node scripts/render_icons.mjs
// (set CHROMIUM to the browser binary if it is not the default).
import { chromium } from "playwright-core";
import fs from "node:fs";
const dir = "public";
const svg = fs.readFileSync(`${dir}/icon.svg`, "utf8");
// Full-bleed variant: platforms (Android maskable, iOS) apply their own mask.
const bleed = svg.replace('rx="112"', 'rx="0"');
const b = await chromium.launch({ executablePath: process.env.CHROMIUM });
const p = await b.newPage();
for (const [name, size, src] of [["icon-192.png", 192, svg], ["icon-512.png", 512, svg], ["icon-maskable-512.png", 512, bleed], ["apple-touch-icon.png", 180, bleed]]) {
  await p.setViewportSize({ width: size, height: size });
  await p.setContent(`<style>html,body{margin:0;background:transparent}svg{display:block;width:${size}px;height:${size}px}</style>${src}`);
  await p.screenshot({ path: `${dir}/${name}`, omitBackground: true });
}
await b.close();
