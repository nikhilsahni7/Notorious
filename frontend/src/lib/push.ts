import { notificationService, PushSubscriptionPayload } from "@/services/notification.service";

/**
 * Web push opt-in helpers. IMPORTANT UX RULE:
 * Notification permission is NEVER requested on app load or login. These
 * functions must only be called from an explicit user action (button tap).
 * All failures degrade silently to in-app notifications (return false).
 */

const SW_PATH = "/sw.js";

function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = window.atob(base64);
  const output = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; ++i) {
    output[i] = raw.charCodeAt(i);
  }
  return output;
}

function pushSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window
  );
}

// detect coarse pointer devices as a cheap device_type signal
function guessDeviceType(): string {
  if (typeof window === "undefined") return "desktop";
  return window.matchMedia("(max-width: 767px)").matches ? "mobile" : "desktop";
}

/**
 * Registers the service worker (idempotent, permission-safe). Called on app
 * mount — this alone does NOT trigger a permission prompt.
 */
export async function registerServiceWorker(): Promise<void> {
  if (!pushSupported()) return;
  try {
    await navigator.serviceWorker.register(SW_PATH);
  } catch {
    // Service worker registration isn't required for anything else to work.
    console.debug("Service worker registration failed (continuing without push)");
  }
}

/**
 * Requests browser notification permission AND subscribes the browser for
 * push. Call strictly from a user gesture (e.g. "Notify me on reset" tap).
 * Returns true only if permission was granted and the subscription was saved.
 */
export async function optInToPush(token: string): Promise<boolean> {
  if (!pushSupported()) return false;

  try {
    const permission = await Notification.requestPermission();
    if (permission !== "granted") return false;

    const registration = await navigator.serviceWorker.register(SW_PATH);
    await navigator.serviceWorker.ready;

    const { public_key } = await notificationService.getPushPublicKey(token);

    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(public_key),
    });

    const json = subscription.toJSON();
    const payload: PushSubscriptionPayload = {
      endpoint: subscription.endpoint,
      keys: {
        p256dh: (json.keys as Record<string, string>)?.p256dh ?? "",
        auth: (json.keys as Record<string, string>)?.auth ?? "",
      },
      device_type: guessDeviceType(),
    };
    if (!payload.keys.p256dh || !payload.keys.auth) return false;

    await notificationService.subscribePush(payload, token);
    return true;
  } catch (error) {
    // Degrade silently to in-app only — no nagging on permission denial.
    console.debug("Push opt-in failed:", error);
    return false;
  }
}

/** Removes the browser's push subscription (server + browser side). */
export async function optOutFromPush(token: string): Promise<void> {
  if (!pushSupported()) return;
  try {
    const registration = await navigator.serviceWorker.ready;
    const subscription = await registration.pushManager.getSubscription();
    if (subscription) {
      await notificationService.unsubscribePush(subscription.endpoint, token);
      await subscription.unsubscribe();
    }
  } catch (error) {
    console.debug("Push opt-out failed:", error);
  }
}

/** Reports the user's current notification permission state, read-only. */
export function pushPermissionState(): string {
  if (typeof window === "undefined") return "unsupported";
  if (!("Notification" in window)) return "unsupported";
  return Notification.permission;
}

/**
 * Full push status: permission + whether the browser actually holds a push
 * subscription. Lets the profile page distinguish "not enrolled" from
 * "permission granted but subscription missing" — no silent downgrade.
 */
export async function readPushStatus(): Promise<{
  permission: NotificationPermission | "unsupported";
  subscribed: boolean;
}> {
  if (typeof window === "undefined" || !("Notification" in window)) {
    return { permission: "unsupported", subscribed: false };
  }
  const permission = Notification.permission;
  try {
    if (!pushSupported()) return { permission, subscribed: false };
    const registration = await navigator.serviceWorker.getRegistration(SW_PATH);
    if (!registration) return { permission, subscribed: false };
    const subscription = await registration.pushManager.getSubscription();
    return { permission, subscribed: !!subscription };
  } catch {
    return { permission, subscribed: false };
  }
}
