/*
 * Knotorious web-push service worker.
 *
 * Responsibilities (only):
 *  1. push            -> show an OS notification with the deep link from the payload
 *  2. notificationclick -> open the deep link (payload url), never a generic page
 *
 * No caching/fetch handlers here on purpose: this worker is for push only and
 * must not interfere with Next.js asset delivery.
 */

self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("push", (event) => {
  let data = {};
  try {
    if (event.data) {
      data = event.data.json();
    }
  } catch {
    // Non-JSON push payload: fall back to a generic-but-safe notification
    data = { title: "Knotorious", body: event.data ? event.data.text() : "" };
  }

  if (!data.title && !data.body) return; // nothing visible to show

  const title = data.title || "Knotorious";
  const options = {
    body: data.body || "",
    // Deep link consumed by notificationclick below
    data: { url: data.url || "/", type: data.type || "" },
    tag: data.type || undefined, // same-type notifications replace each other
    renotify: false,
    requireInteraction: false,
    icon: "/third_eye_emblem.png",
    badge: "/third_eye_emblem.png",
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();

  const rawUrl = (event.notification.data && event.notification.data.url) || "/";
  const target = new URL(rawUrl, self.location.origin).href;

  event.waitUntil(
    (async () => {
      const windowClients = await self.clients.matchAll({
        type: "window",
        includeUncontrolled: true,
      });

      // If a window of ours is already open, focus it and navigate to the deep link
      for (const client of windowClients) {
        if (client.url.startsWith(self.location.origin) && "navigate" in client) {
          try {
            await client.focus();
          } catch {
            // focus can be rejected on some platforms; still navigate below
          }
          if ("navigate" in client) {
            await client.navigate(target);
          }
          return;
        }
      }

      // No window open: open a new one straight at the deep link
      await self.clients.openWindow(target);
    })()
  );
});
