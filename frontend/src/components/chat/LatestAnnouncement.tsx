"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useChat } from "@/contexts/ChatContext";
import { Broadcast, chatService } from "@/services/chat.service";
import { Megaphone, ArrowRight } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

/** A preview does not mark an announcement read; opening it does. */
export function LatestAnnouncement() {
  const { token, user } = useAuth();
  const { onBroadcast, onBroadcastRead } = useChat();
  const [unread, setUnread] = useState<Broadcast[]>([]);
  const readIds = useRef(new Set<string>());

  useEffect(() => {
    if (!token || user?.role === "admin") return;
    let cancelled = false;
    chatService.getBroadcasts(token, 100).then((items) => {
      if (cancelled) return;
      setUnread((current) => {
        const combined = new Map<string, Broadcast>();
        for (const item of [...items, ...current]) {
          if (!item.is_read && !readIds.current.has(item.id)) combined.set(item.id, item);
        }
        return [...combined.values()].sort((a, b) =>
          new Date(b.sent_at).getTime() - new Date(a.sent_at).getTime());
      });
    }).catch(console.error);
    return () => { cancelled = true; };
  }, [token, user?.role]);

  useEffect(() => onBroadcast((item) => {
    if (item.is_read || readIds.current.has(item.id)) return;
    setUnread((current) => [item, ...current.filter((entry) => entry.id !== item.id)]);
  }), [onBroadcast]);
  useEffect(() => onBroadcastRead((id) => {
    readIds.current.add(id);
    setUnread((current) => current.filter((item) => item.id !== id));
  }), [onBroadcastRead]);

  const latest = unread[0];
  if (!latest || user?.role === "admin") return null;
  return (
    <Link href={`/chat?tab=announcements&announcement=${encodeURIComponent(latest.id)}`}
      className="mb-3 flex items-center gap-3 rounded-lg border border-sky-400/30 bg-sky-950/40 px-4 py-3 text-white transition-colors hover:bg-sky-900/50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-300">
      <Megaphone className="h-5 w-5 shrink-0 text-sky-300" aria-hidden="true" />
      <span className="min-w-0 flex-1">
        <span className="block text-xs text-sky-200">Latest announcement · New</span>
        <span className="block truncate text-sm">{latest.content}</span>
      </span>
      <span className="hidden text-xs text-sky-200 sm:inline">Read announcement</span>
      <ArrowRight className="h-4 w-4 shrink-0 text-sky-200" aria-hidden="true" />
    </Link>
  );
}
