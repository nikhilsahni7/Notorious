"use client";

import { useAuth } from "@/contexts/AuthContext";
import { ChatProvider } from "@/contexts/ChatContext";

export function ChatProviderWrapper({
  children,
}: {
  children: React.ReactNode;
}) {
  const { user } = useAuth();

  // Only mount ChatProvider when user is authenticated
  if (!user) {
    return <>{children}</>;
  }

  return <ChatProvider>{children}</ChatProvider>;
}
