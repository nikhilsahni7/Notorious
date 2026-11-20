"use client";

import { ChatMessage } from "@/services/chat.service";
import { motion } from "framer-motion";
import { Check, CheckCheck, Clock } from "lucide-react";

interface ChatBubbleProps {
  message: ChatMessage;
  isOwn: boolean;
  showStatus?: boolean;
}

export function ChatBubble({ message, isOwn, showStatus = true }: ChatBubbleProps) {
  const time = new Date(message.sent_at).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });

  return (
    <motion.div
      initial={{ opacity: 0, y: 10, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.2 }}
      className={`flex ${isOwn ? "justify-end" : "justify-start"} mb-2`}
    >
      <div
        className={`min-w-[80px] max-w-[75%] rounded-2xl px-4 py-2.5 shadow-md ${
          isOwn
            ? "bg-gradient-to-br from-indigo-500 to-purple-600 text-white rounded-br-sm"
            : "bg-gray-800/80 text-gray-100 rounded-bl-sm border border-gray-700/50"
        }`}
      >
        <p className="text-[15px] leading-relaxed whitespace-pre-wrap break-words">
          {message.content}
        </p>
        <div
          className={`flex items-center gap-1 mt-1 ${
            isOwn ? "justify-end" : "justify-start"
          }`}
        >
          <span className={`text-[11px] ${isOwn ? "text-white/70" : "text-gray-400"}`}>{time}</span>
          {isOwn && showStatus && <MessageStatus status={message.status} />}
        </div>
      </div>
    </motion.div>
  );
}

function MessageStatus({ status }: { status: string }) {
  switch (status) {
    case "sent":
      return <Check size={12} className="opacity-60" />;
    case "delivered":
      return <CheckCheck size={12} className="opacity-60" />;
    case "read":
      return <CheckCheck size={12} className="text-blue-300" />;
    default:
      return <Clock size={12} className="opacity-40" />;
  }
}

// Typing indicator dots animation
export function TypingIndicator() {
  return (
    <div className="flex items-center gap-1 px-4 py-3 bg-gray-800/60 rounded-2xl rounded-bl-sm border border-gray-700/50 w-fit">
      {[0, 1, 2].map((i) => (
        <motion.div
          key={i}
          className="w-2 h-2 rounded-full bg-gray-400"
          animate={{ opacity: [0.3, 1, 0.3], y: [0, -4, 0] }}
          transition={{
            duration: 0.8,
            repeat: Infinity,
            delay: i * 0.15,
          }}
        />
      ))}
    </div>
  );
}
