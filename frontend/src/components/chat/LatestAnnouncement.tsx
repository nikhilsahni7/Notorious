"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useChat } from "@/contexts/ChatContext";
import { Broadcast, chatService } from "@/services/chat.service";
import { Megaphone, ArrowRight } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

/** A preview does not mark an announcement read; opening it does. */
export function LatestAnnouncement() {
  const { token, user } = useAuth();
  const { onBroadcast, onBroadcastRead } = useChat();
  const [latest, setLatest] = useState<Broadcast | null>(null);

  useEffect(() => {
    if (!token || user?.role === "admin") return;
    let cancelled = false;
    chatService.getBroadcasts(token, 1).then((items) => {
      if (!cancelled && items[0]) setLatest((current) =>
        current && new Date(current.sent_at) > new Date(items[0].sent_at) ? current : items[0]);
    }).catch(console.error);
    return () => { cancelled = true; };
  }, [token, user?.role]);

  useEffect(() => onBroadcast((item) => setLatest(item)), [onBroadcast]);
  useEffect(() => onBroadcastRead((id) => setLatest((item) =>
    item?.id === id ? { ...item, is_read: true } : item)), [onBroadcastRead]);

  if (!latest || user?.role === "admin") return null;
  return (
    <Link href={`/chat?tab=announcements&announcement=${encodeURIComponent(latest.id)}`}
      className="mb-3 flex items-center gap-3 rounded-lg border border-purple-400/30 bg-purple-950/40 px-4 py-3 text-white transition-colors hover:bg-purple-900/50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-purple-300">
      <Megaphone className="h-5 w-5 shrink-0 text-purple-300" aria-hidden="true" />
      <span className="min-w-0 flex-1">
        <span className="block text-xs text-purple-200">Latest announcement {!latest.is_read && "· New"}</span>
        <span className="block truncate text-sm">{latest.content}</span>
      </span>
      <span className="hidden text-xs text-purple-200 sm:inline">Read announcement</span>
      <ArrowRight className="h-4 w-4 shrink-0 text-purple-200" aria-hidden="true" />
    </Link>
  );
}
