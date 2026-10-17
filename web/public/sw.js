// SPDX-License-Identifier: AGPL-3.0
// Minimal service worker: cache the app shell so the UI opens instantly and
// survives transient network failures. API and git traffic always go to the
// network so data is never served stale.
const CACHE = 'gitli-shell-v1'
const SHELL = ['/', '/manifest.webmanifest']

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL)).catch(() => undefined))
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))),
  )
  self.clients.claim()
})

self.addEventListener('fetch', (event) => {
  const req = event.request
  if (req.method !== 'GET') return
  const url = new URL(req.url)
  if (url.origin !== self.location.origin) return

  if (
    url.pathname.startsWith('/api/') ||
    url.pathname.startsWith('/v2/') ||
    url.pathname.endsWith('/info/refs') ||
    url.pathname.includes('git-upload-pack') ||
    url.pathname.includes('git-receive-pack')
  ) {
    return
  }

  if (url.pathname.startsWith('/assets/')) {
    event.respondWith(
      caches.match(req).then(
        (hit) =>
          hit ||
          fetch(req).then((res) => {
            const copy = res.clone()
            caches.open(CACHE).then((c) => c.put(req, copy)).catch(() => undefined)
            return res
          }),
      ),
    )
    return
  }

  if (req.mode === 'navigate') {
    event.respondWith(fetch(req).catch(() => caches.match('/').then((hit) => hit || Response.error())))
  }
})
