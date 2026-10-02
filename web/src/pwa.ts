// Installable app support: the service worker (offline shell and updates),
// the browser's install prompt and the connection state. The browser only
// allows service workers and installing over HTTPS or on localhost.

import { useEffect, useState, useSyncExternalStore } from "react";

type Listener = () => void;

// A tiny observable value, so React components can follow browser events
// that happen before or outside them.
function store<T>(initial: T) {
  let value = initial;
  const listeners = new Set<Listener>();
  return {
    get: () => value,
    set(v: T) {
      value = v;
      listeners.forEach((l) => l());
    },
    subscribe(l: Listener) {
      listeners.add(l);
      return () => listeners.delete(l);
    },
  };
}

// The waiting service worker of a new version, if any.
const update = store<ServiceWorker | null>(null);
// Set when the user taps "Actualizar". The first install also changes the
// controller (clients.claim), and that must not reload a half-typed login.
let updateRequested = false;

// Chrome and Edge fire beforeinstallprompt once, early; keep it for the
// "Instalar" button in Ajustes.
type InstallPrompt = Event & { prompt: () => Promise<void>; userChoice: Promise<{ outcome: string }> };
const installPrompt = store<InstallPrompt | null>(null);

export function startPWA() {
  window.addEventListener("beforeinstallprompt", (e) => {
    e.preventDefault();
    installPrompt.set(e as InstallPrompt);
  });
  window.addEventListener("appinstalled", () => installPrompt.set(null));

  if (!import.meta.env.PROD || !("serviceWorker" in navigator) || !window.isSecureContext) return;

  // When the requested new version takes over, reload once to use it.
  navigator.serviceWorker.addEventListener("controllerchange", () => {
    if (!updateRequested) return;
    updateRequested = false;
    location.reload();
  });

  navigator.serviceWorker.register("/sw.js").then((reg) => {
    const watch = (sw: ServiceWorker | null) => {
      sw?.addEventListener("statechange", () => {
        // A waiting worker with an existing controller is an update; the
        // first install just starts working silently.
        if (sw.state === "installed" && navigator.serviceWorker.controller) update.set(sw);
      });
    };
    if (reg.waiting && navigator.serviceWorker.controller) update.set(reg.waiting);
    watch(reg.installing);
    reg.addEventListener("updatefound", () => watch(reg.installing));
    // Installed apps can stay open for days: look for updates every hour.
    setInterval(() => reg.update().catch(() => {}), 60 * 60 * 1000);
  });
}

export function useUpdate(): (() => void) | null {
  const sw = useSyncExternalStore(update.subscribe, update.get);
  if (!sw) return null;
  return () => {
    updateRequested = true;
    sw.postMessage("activate-update");
  };
}

export type InstallState =
  | { kind: "installed" }
  | { kind: "prompt"; install: () => void }
  | { kind: "ios" }
  | { kind: "needs-https" }
  | { kind: "browser-menu" };

const isStandalone = () =>
  window.matchMedia("(display-mode: standalone)").matches ||
  (navigator as Navigator & { standalone?: boolean }).standalone === true;

// iPadOS reports itself as a Mac; touch support tells them apart.
const isIOS = () =>
  /iPhone|iPad|iPod/.test(navigator.userAgent) || (navigator.userAgent.includes("Mac") && navigator.maxTouchPoints > 1);

// useInstall says how this device can install the app.
export function useInstall(): InstallState {
  const prompt = useSyncExternalStore(installPrompt.subscribe, installPrompt.get);
  if (isStandalone()) return { kind: "installed" };
  if (prompt) {
    return {
      kind: "prompt",
      install: () => {
        prompt.prompt();
        prompt.userChoice.finally(() => installPrompt.set(null));
      },
    };
  }
  // Safari on iOS has no install prompt but can always add to the home screen.
  if (isIOS()) return { kind: "ios" };
  if (!window.isSecureContext) return { kind: "needs-https" };
  return { kind: "browser-menu" };
}

export function useOnline(): boolean {
  const [online, setOnline] = useState(navigator.onLine);
  useEffect(() => {
    const on = () => setOnline(true);
    const off = () => setOnline(false);
    window.addEventListener("online", on);
    window.addEventListener("offline", off);
    return () => {
      window.removeEventListener("online", on);
      window.removeEventListener("offline", off);
    };
  }, []);
  return online;
}
