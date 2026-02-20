"use client";

import ChatPage from "@/components/chat/ChatPage";
import { Suspense } from "react";

export default function ChatPageRoute() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center h-screen bg-gray-950 text-white">
          <div className="animate-pulse text-gray-400">Loading chat...</div>
        </div>
      }
    >
      <ChatPage />
    </Suspense>
  );
}
