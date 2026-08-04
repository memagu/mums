const CACHE_NAME = "mums-static-v1";

self.addEventListener("install", (event) => {
	event.waitUntil(
		caches.open(CACHE_NAME).then((cache) =>
			cache.addAll(["/static/offline.html", "/static/icons/mums.svg"]),
		),
	);
	self.skipWaiting();
});

self.addEventListener("activate", (event) => {
	event.waitUntil(
		caches.keys().then((keys) =>
			Promise.all(
				keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)),
			),
		),
	);
	self.clients.claim();
});

self.addEventListener("fetch", (event) => {
	const url = new URL(event.request.url);

	if (url.origin !== self.location.origin) {
		return;
	}

	if (url.pathname.startsWith("/static/")) {
		event.respondWith(
			caches.match(event.request).then((cached) => {
				if (cached) {
					return cached;
				}
				return fetch(event.request).then((response) => {
					const copy = response.clone();
					caches.open(CACHE_NAME).then((cache) => cache.put(event.request, copy));
					return response;
				});
			}),
		);
		return;
	}

	if (event.request.mode === "navigate") {
		event.respondWith(
			fetch(event.request).catch(() =>
				caches
					.match("/static/offline.html")
					.then((cached) => cached || new Response("Offline", { status: 503 })),
			),
		);
	}
});
