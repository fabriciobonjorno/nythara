const CACHE = "nythara-shell-v6";
const SHELL = ["/", "/manifest.webmanifest", "/assets/nythara-apocalypse-arena.webp", "/assets/nythara-apocalypse-logo.webp", "/assets/nythara-apocalypse-wordmark.webp", "/assets/nythara-apocalypse-emblem-v2.webp", "/assets/icon-64.png", "/assets/icon-192.png", "/assets/icon-512.png"];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(SHELL)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(caches.keys().then((keys) => Promise.all(keys.filter((key) => key !== CACHE).map((key) => caches.delete(key)))).then(() => self.clients.claim()));
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  const url = new URL(request.url);
  if (request.method !== "GET" || url.origin !== self.location.origin || url.pathname.startsWith("/v1/") || url.protocol === "ws:" || url.protocol === "wss:") return;
  if (request.mode === "navigate") {
    event.respondWith((async () => {
      try {
        return await fetch(request);
      } catch {
        const shell = await caches.match("/");
        return shell ?? new Response("<!doctype html><html lang=\"pt-BR\"><body><h1>Nythara está temporariamente offline</h1><p>Reconecte-se e tente novamente.</p></body></html>", {
          status: 503,
          headers: { "Content-Type": "text/html; charset=utf-8" },
        });
      }
    })());
    return;
  }
  event.respondWith((async () => {
    const cached = await caches.match(request);
    if (cached) return cached;
    try {
      const response = await fetch(request);
      if (response.ok) {
        const cache = await caches.open(CACHE);
        await cache.put(request, response.clone());
      }
      return response;
    } catch {
      return new Response("Recurso indisponível offline.", { status: 503, headers: { "Content-Type": "text/plain; charset=utf-8" } });
    }
  })());
});
