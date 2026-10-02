// Service worker: the app opens without a connection and loads instantly.
// Built by the Vite config, which fills in VERSION and PRECACHE.
//
// - App files (HTML, JS, CSS, icons, fonts) come from the cache.
// - /api is never cached: data is per user and must always be current.
// - A new version waits until the app asks it to take over ("Actualizar"),
//   so a test in progress is never reloaded under the user's feet.

const VERSION = "__VERSION__";
const PRECACHE = __PRECACHE__;
const CACHE = `tai-${VERSION}`;

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(PRECACHE)));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k.startsWith("tai-") && k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("message", (event) => {
  if (event.data === "activate-update") self.skipWaiting();
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  const url = new URL(req.url);
  if (req.method !== "GET" || url.origin !== self.location.origin || url.pathname.startsWith("/api/")) return;

  // Pages: always try the network for the newest app, fall back to the
  // cached shell offline. Every route is the same single-page app.
  if (req.mode === "navigate") {
    event.respondWith(fetch(req).catch(() => caches.match("/", { cacheName: CACHE })));
    return;
  }

  // Files: fingerprinted or versioned with the cache, so cache first.
  event.respondWith(
    caches.match(req, { cacheName: CACHE }).then(
      (hit) =>
        hit ||
        fetch(req).then((res) => {
          if (res.ok && url.pathname.startsWith("/assets/")) {
            const copy = res.clone();
            caches.open(CACHE).then((cache) => cache.put(req, copy));
          }
          return res;
        }),
    ),
  );
});
