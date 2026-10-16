"use client";

import { useEffect } from "react";
import { registerServiceWorker } from "@/lib/push";

/**
 * Registers /sw.js once on app mount so the browser can receive pushes.
 * Registration alone never requests notification permission — permission is
 * only requested from explicit user taps (see lib/push.ts).
 */
export function PWARegistrar() {
  useEffect(() => {
    registerServiceWorker();
  }, []);

  return null;
}
