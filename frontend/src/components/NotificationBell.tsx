"use client";

import { useAuth } from "@/contexts/AuthContext";
import { Bell } from "lucide-react";
import { useRouter } from "next/navigation";

/**
 * Header bell: shows the heartbeat-driven unread badge and navigates to the
 * full /notifications page (the in-page drawer was replaced by that page).
 */
export function NotificationBell() {
  const { unreadNotifications } = useAuth();
  const router = useRouter();

  return (
    <button
      onClick={() => router.push("/notifications")}
      aria-label="Notifications"
      className="relative p-2 rounded-lg bg-transparent border border-white/10 text-white hover:bg-white/10 hover:border-white/20 transition-all duration-300"
    >
      <Bell className="h-4 w-4" />
      {unreadNotifications > 0 && (
        <span
          className={`absolute -top-1.5 -right-1.5 min-w-[18px] h-[18px] px-1 flex items-center justify-center rounded-full text-[10px] font-bold text-white bg-pink-500 shadow-[0_0_8px_rgba(236,72,153,0.6)] ${
            unreadNotifications > 99 ? "text-[8px]" : ""
          }`}
        >
          {unreadNotifications > 99 ? "99+" : unreadNotifications}
        </span>
      )}
    </button>
  );
}
