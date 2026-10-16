"use client";

import { useAuth } from "@/contexts/AuthContext";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import {
  AppNotification,
  NotificationType,
  notificationService,
} from "@/services/notification.service";
import {
  AlertTriangle,
  ArrowLeft,
  Bell,
  BellRing,
  CheckCheck,
  Download,
  Target,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState, type ReactNode } from "react";

const PAGE_SIZE = 50;

const TYPE_META: Record<NotificationType, { icon: ReactNode; accent: string; action: string }> = {
  quota_reset: {
    icon: <BellRing className="h-4 w-4" />,
    accent: "text-yellow-300",
    action: "Go to search",
  },
  watchlist_match: {
    icon: <Target className="h-4 w-4" />,
    accent: "text-pink-400",
    action: "Re-run search",
  },
  export_ready: {
    icon: <Download className="h-4 w-4" />,
    accent: "text-green-400",
    action: "Download",
  },
  quota_warning: {
    icon: <AlertTriangle className="h-4 w-4" />,
    accent: "text-orange-400",
    action: "Open",
  },
};

function actionLabel(n: AppNotification): string {
  return n.payload?.label && typeof n.payload.label === "string"
    ? n.payload.label
    : TYPE_META[n.type]?.action ?? "Open";
}

function isToday(iso: string): boolean {
  return new Date(iso).toDateString() === new Date().toDateString();
}

function formatWhen(iso: string): string {
  const date = new Date(iso);
  if (isToday(iso)) {
    return date.toLocaleTimeString("en-IN", { hour: "2-digit", minute: "2-digit" });
  }
  return date.toLocaleString("en-IN", { dateStyle: "medium", timeStyle: "short" });
}

export default function NotificationsPage() {
  const { user, token, isLoading, unreadNotifications, setUnreadNotifications } = useAuth();
  const router = useRouter();
  const [items, setItems] = useState<AppNotification[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [markingAll, setMarkingAll] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!isLoading && !token) {
      router.push("/login");
    }
  }, [isLoading, token, router]);

  const load = useCallback(
    async (offset: number, append: boolean) => {
      if (!token) return;
      if (append) setLoadingMore(true);
      else setLoading(true);
      setError("");
      try {
        const data = await notificationService.list(token, { limit: PAGE_SIZE, offset });
        const next = data.notifications || [];
        setItems((prev) => (append ? [...prev, ...next] : next));
        setTotal(data.total ?? next.length);
        if (!append) {
          const unread = await notificationService.getUnreadCount(token);
          setUnreadNotifications(unread.unread ?? 0);
        }
      } catch (err) {
        console.error("Failed to load notifications:", err);
        setError("Could not load notifications.");
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [token]
  );

  useEffect(() => {
    if (token) load(0, false);
  }, [token, load]);

  const handleMarkAllRead = async () => {
    if (!token || unreadNotifications === 0) return;
    setMarkingAll(true);
    try {
      await notificationService.markAllRead(token);
      const now = new Date().toISOString();
      setItems((prev) => prev.map((n) => ({ ...n, read_at: n.read_at ?? now })));
      setUnreadNotifications(0);
    } catch (err) {
      console.error("Failed to mark all read:", err);
    } finally {
      setMarkingAll(false);
    }
  };

  const handleOpenItem = async (n: AppNotification) => {
    if (!n.read_at && token) {
      const readAt = new Date().toISOString();
      setItems((prev) => prev.map((x) => (x.id === n.id ? { ...x, read_at: readAt } : x)));
      setUnreadNotifications((prev) => Math.max(0, prev - 1));
      try {
        await notificationService.markRead(n.id, token);
      } catch (err) {
        console.error("Failed to mark notification read:", err);
      }
    }
    const url = (n.payload?.url as string) || "/search";
    router.push(url);
  };

  if (isLoading || !token) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  const today = items.filter((n) => isToday(n.created_at));
  const earlier = items.filter((n) => !isToday(n.created_at));
  const hasMore = items.length < total;

  const renderRow = (n: AppNotification) => {
    const meta = TYPE_META[n.type] ?? TYPE_META.quota_warning;
    const unread = !n.read_at;
    return (
      <div
        key={n.id}
        className={`flex items-start gap-3 p-4 ${unread ? "bg-white/[0.03]" : ""}`}
      >
        <div className={`mt-0.5 flex-shrink-0 ${meta.accent}`}>{meta.icon}</div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className={`text-sm ${unread ? "text-white font-semibold" : "text-gray-300"}`}>
              {n.title}
            </p>
            {unread && <span className="h-1.5 w-1.5 rounded-full bg-pink-500 flex-shrink-0" />}
          </div>
          <p className="text-sm text-gray-400 mt-1 break-words">{n.body}</p>
          <Button
            onClick={() => handleOpenItem(n)}
            size="sm"
            variant="outline"
            className="mt-3 h-8 px-3 text-xs bg-transparent border-white/10 text-white hover:bg-white/10 hover:border-white/20"
          >
            {actionLabel(n)}
          </Button>
        </div>
        <span className="text-xs text-gray-500 flex-shrink-0 mt-0.5">{formatWhen(n.created_at)}</span>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-3 md:p-6">
      <div className="max-w-3xl mx-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-4 bg-white/5 backdrop-blur-xl p-4 rounded-2xl border border-white/10 shadow-xl">
          <div className="flex items-center gap-3 min-w-0">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              size="sm"
              className="h-8 px-3 bg-transparent border-purple-500/50 text-purple-300 hover:bg-purple-500/20 hover:border-purple-400 hover:text-white flex-shrink-0"
            >
              <ArrowLeft className="h-4 w-4 mr-1.5" />
              Back
            </Button>
            <div className="min-w-0">
              <h1 className="text-xl font-bold text-white">Notifications</h1>
              <p className="text-sm text-gray-400">
                <span className="hidden sm:inline">{user?.name} • </span>
                {unreadNotifications > 0 ? `${unreadNotifications} unread` : "All caught up"}
              </p>
            </div>
          </div>
          <Button
            onClick={handleMarkAllRead}
            disabled={markingAll || unreadNotifications === 0}
            size="sm"
            variant="outline"
            className="h-8 px-3 bg-transparent border-white/10 text-gray-300 hover:bg-white/10 hover:border-white/20 hover:text-white disabled:opacity-40"
          >
            <CheckCheck className="h-4 w-4 mr-1.5" />
            Mark all read
          </Button>
        </div>

        <div className="bg-[#1a0f2e] rounded-2xl border border-white/10 overflow-hidden">
          {loading ? (
            <div className="flex items-center justify-center py-16">
              <Spinner size="lg" />
            </div>
          ) : error ? (
            <div className="text-center py-16 px-6">
              <p className="text-sm text-gray-400">{error}</p>
              <Button
                onClick={() => load(0, false)}
                className="mt-4 bg-pink-500 hover:bg-pink-600 text-white"
              >
                Try again
              </Button>
            </div>
          ) : items.length === 0 ? (
            <div className="text-center py-16 px-6 text-gray-400">
              <Bell className="h-10 w-10 mx-auto mb-3 text-white/20" />
              <p className="text-base text-white">No notifications yet</p>
              <p className="text-sm mt-1">Quota resets and alerts will show up here.</p>
            </div>
          ) : (
            <>
              {today.length > 0 && (
                <section>
                  <p className="px-4 pt-4 pb-2 text-xs uppercase tracking-wider text-gray-500">Today</p>
                  <div className="divide-y divide-white/5">{today.map(renderRow)}</div>
                </section>
              )}
              {earlier.length > 0 && (
                <section className={today.length > 0 ? "border-t border-white/10" : ""}>
                  <p className="px-4 pt-4 pb-2 text-xs uppercase tracking-wider text-gray-500">Earlier</p>
                  <div className="divide-y divide-white/5">{earlier.map(renderRow)}</div>
                </section>
              )}
              {hasMore && (
                <div className="p-4 border-t border-white/10 text-center">
                  <Button
                    onClick={() => load(items.length, true)}
                    disabled={loadingMore}
                    variant="outline"
                    size="sm"
                    className="bg-transparent border-white/10 text-white hover:bg-white/10"
                  >
                    {loadingMore ? "Loading…" : "Load more"}
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
