"use client";

import { useEffect, useState } from "react";

// IST is UTC+05:30 all year (no DST in India), so the offset is a constant.
const IST_OFFSET_MS = (5 * 60 + 30) * 60 * 1000;
const DAY_MS = 24 * 60 * 60 * 1000;

interface ClockState {
  /** Current IST wall clock label, e.g. "23:42" (append IST in the UI). */
  nowLabel: string;
  /** Time until the next IST midnight reset, e.g. "5h 17m". */
  resetsIn: string;
  /** Same as resetsIn: the quota-reset push is delivered at midnight itself. */
  notifiesIn: string;
}

/** Wall clock in IST for a UTC timestamp. */
function istWallclock(nowMs: number): { h: number; m: number; s: number } {
  const ist = new Date(nowMs + IST_OFFSET_MS);
  return { h: ist.getUTCHours(), m: ist.getUTCMinutes(), s: ist.getUTCSeconds() };
}

/** Milliseconds until the next occurrence of hh:mm IST. */
function msUntilIst(hh: number, mm: number): number {
  const { h, m, s } = istWallclock(Date.now());
  const currentMs = h * 3600000 + m * 60000 + s * 1000;
  const targetMs = hh * 3600000 + mm * 60000;
  return (targetMs - currentMs + DAY_MS) % DAY_MS;
}

function formatCountdown(ms: number): string {
  const mins = Math.max(0, Math.floor(ms / 60000));
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function compute(): ClockState {
  const { h, m } = istWallclock(Date.now());
  const p2 = (n: number) => String(n).padStart(2, "0");
  const resetsIn = formatCountdown(msUntilIst(0, 0));
  return {
    nowLabel: `${p2(h)}:${p2(m)}`,
    resetsIn,
    notifiesIn: resetsIn,
  };
}

/**
 * Live IST clock + countdown to the midnight IST reset (limits reset AND the
 * opted-in notification are delivered at midnight). Null until mounted so
 * SSR markup and hydration never disagree; refreshed every 30 seconds.
 */
export function useIstCountdown(): ClockState | null {
  const [clock, setClock] = useState<ClockState | null>(null);

  useEffect(() => {
    const tick = () => setClock(compute());
    tick();
    const interval = setInterval(tick, 30000);
    return () => clearInterval(interval);
  }, []);

  return clock;
}
