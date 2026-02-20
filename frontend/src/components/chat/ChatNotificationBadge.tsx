"use client";

import { useChat } from "@/contexts/ChatContext";
import { AnimatePresence, motion } from "framer-motion";
import { MessageCircle } from "lucide-react";
import Link from "next/link";

interface ChatNotificationBadgeProps {
  href?: string;
  className?: string;
  showIcon?: boolean;
  size?: "sm" | "md" | "lg";
}

export function ChatNotificationBadge({
  href = "/chat",
  className = "",
  showIcon = true,
  size = "md",
}: ChatNotificationBadgeProps) {
  const { unreadCount, broadcastUnread, isConnected } = useChat();
  const totalUnread = unreadCount + broadcastUnread;

  const sizeStyles = {
    sm: { icon: 16, badge: "min-w-[16px] h-4 text-[10px] px-1" },
    md: { icon: 20, badge: "min-w-[20px] h-5 text-xs px-1.5" },
    lg: { icon: 24, badge: "min-w-[24px] h-6 text-sm px-2" },
  };

  const s = sizeStyles[size];

  return (
    <Link href={href} className={`relative inline-flex items-center ${className}`}>
      {showIcon && (
        <div className="relative">
          <MessageCircle
            size={s.icon}
            className={`transition-colors ${
              isConnected ? "text-emerald-400" : "text-gray-400"
            }`}
          />
          {/* Connection indicator dot */}
          <div
            className={`absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full border border-gray-900 ${
              isConnected ? "bg-emerald-400" : "bg-gray-500"
            }`}
          />
        </div>
      )}
      <AnimatePresence>
        {totalUnread > 0 && (
          <motion.span
            initial={{ scale: 0, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0, opacity: 0 }}
            className={`absolute -top-1.5 -right-2 ${s.badge} flex items-center justify-center rounded-full bg-gradient-to-r from-red-500 to-pink-500 text-white font-bold shadow-lg shadow-red-500/30`}
          >
            {totalUnread > 99 ? "99+" : totalUnread}
          </motion.span>
        )}
      </AnimatePresence>
    </Link>
  );
}

// Inline badge variant (no link, just the number)
export function InlineUnreadBadge({
  count,
  className = "",
}: {
  count: number;
  className?: string;
}) {
  if (count <= 0) return null;

  return (
    <motion.span
      initial={{ scale: 0 }}
      animate={{ scale: 1 }}
      className={`inline-flex items-center justify-center min-w-[20px] h-5 px-1.5 rounded-full bg-gradient-to-r from-red-500 to-pink-500 text-white text-xs font-bold shadow-lg shadow-red-500/30 ${className}`}
    >
      {count > 99 ? "99+" : count}
    </motion.span>
  );
}
