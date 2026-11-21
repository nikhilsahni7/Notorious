"use client";

import { ChatMessage } from "@/services/chat.service";
import { motion } from "framer-motion";
import { Check, CheckCheck, Clock, Forward } from "lucide-react";

interface ChatBubbleProps {
  message: ChatMessage;
  isOwn: boolean;
  showStatus?: boolean;
  onForward?: (message: ChatMessage) => void;
}

export function ChatBubble({ message, isOwn, showStatus = true, onForward }: ChatBubbleProps) {
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
      {isOwn && onForward && (
        <button type="button" onClick={() => onForward(message)} aria-label="Forward message" title="Forward message" className="mr-2 self-center rounded-lg p-2 text-slate-400 hover:bg-sky-900/50 hover:text-sky-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-400"><Forward size={16} /></button>
      )}
      <div
        className={`min-w-[80px] max-w-[75%] rounded-2xl px-4 py-2.5 ${
          isOwn
            ? "bg-[#14558b] text-white rounded-br-sm"
            : "bg-[#102947] text-slate-100 rounded-bl-sm border border-sky-900/50"
        }`}
      >
        <p className="text-sm leading-relaxed whitespace-pre-wrap break-words">
          {message.content}
        </p>
        <div
          className={`flex items-center gap-1 mt-1 ${
            isOwn ? "justify-end" : "justify-start"
          }`}
        >
          <span className={`text-xs ${isOwn ? "text-white/70" : "text-gray-400"}`}>{time}</span>
          {isOwn && showStatus && <MessageStatus status={message.status} />}
        </div>
      </div>
      {!isOwn && onForward && (
        <button type="button" onClick={() => onForward(message)} aria-label="Forward message" title="Forward message" className="ml-2 self-center rounded-lg p-2 text-slate-400 hover:bg-sky-900/50 hover:text-sky-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-400"><Forward size={16} /></button>
      )}
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
