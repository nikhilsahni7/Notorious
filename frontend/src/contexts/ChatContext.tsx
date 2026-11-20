"use client";

import { getWsUrl } from "@/config/api";
import { useAuth } from "@/contexts/AuthContext";
import {
  Broadcast,
  ChatMessage,
  chatService,
  Conversation,
} from "@/services/chat.service";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

// WebSocket event types (must match backend)
const WS_TYPES = {
  CHAT_MESSAGE: "chat_message",
  MESSAGE_ACK: "message_ack",
  NEW_MESSAGE: "new_message",
  TYPING_START: "typing_start",
  TYPING_STOP: "typing_stop",
  READ_RECEIPT: "read_receipt",
  READ_UPDATE: "read_update",
  ONLINE_STATUS: "online_status",
  DELIVERY_UPDATE: "delivery_update",
  BROADCAST: "broadcast",
  BROADCAST_READ: "broadcast_read",
  ERROR: "error",
} as const;

interface WSEnvelope {
  type: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  payload: any;
}

interface PendingMessage {
  tempId: string;
  receiverId: string;
  content: string;
  sentAt: string;
}

interface ChatContextType {
  // State
  conversations: Conversation[];
  unreadCount: number;
  broadcastUnread: number;
  onlineUsers: Set<string>;
  typingUsers: Map<string, boolean>;
  isConnected: boolean;

  // Actions
  sendMessage: (receiverId: string, content: string) => string; // returns tempId
  sendReadReceipt: (senderId: string) => void;
  sendTypingStart: (userId: string) => void;
  sendTypingStop: (userId: string) => void;
  sendBroadcast: (content: string) => void;
  markBroadcastRead: (broadcastId: string) => void;
  refreshConversations: () => Promise<void>;
  refreshUnreadCount: () => Promise<void>;

  // Listeners
  onNewMessage: (callback: (msg: ChatMessage) => void) => () => void;
  onMessageAck: (
    callback: (ack: {
      temp_id: string;
      message_id: string;
      sequence_num: number;
      sent_at: string;
    }) => void
  ) => () => void;
  onReadUpdate: (
    callback: (update: { reader_id: string; read_at: string }) => void
  ) => () => void;
  onDeliveryUpdate: (
    callback: (update: { receiver_id: string }) => void
  ) => () => void;
  onBroadcast: (callback: (broadcast: Broadcast) => void) => () => void;
  onTyping: (
    callback: (data: { userId: string; isTyping: boolean }) => void
  ) => () => void;
}

const ChatContext = createContext<ChatContextType | undefined>(undefined);

const RECONNECT_BASE_DELAY = 1000; // 1s
const RECONNECT_MAX_DELAY = 30000; // 30s

export function ChatProvider({ children }: { children: React.ReactNode }) {
  const { token, user } = useAuth();
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [broadcastUnread, setBroadcastUnread] = useState(0);
  const [onlineUsers, setOnlineUsers] = useState<Set<string>>(new Set());
  const [typingUsers, setTypingUsers] = useState<Map<string, boolean>>(
    new Map()
  );
  const [isConnected, setIsConnected] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptRef = useRef(0);
  const isCleaningUpRef = useRef(false);

  // Event listener maps
  const messageListenersRef = useRef<Set<(msg: ChatMessage) => void>>(
    new Set()
  );
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const ackListenersRef = useRef<Set<(ack: any) => void>>(new Set());
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const readUpdateListenersRef = useRef<Set<(update: any) => void>>(new Set());
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const deliveryListenersRef = useRef<Set<(update: any) => void>>(new Set());
  const broadcastListenersRef = useRef<Set<(broadcast: Broadcast) => void>>(
    new Set()
  );
  const typingListenersRef = useRef<
    Set<(data: { userId: string; isTyping: boolean }) => void>
  >(new Set());
  const typingTimeoutsRef = useRef<Map<string, NodeJS.Timeout>>(new Map());

  // Connect WebSocket
  const connect = useCallback(() => {
    if (!token || isCleaningUpRef.current) return;

    // Don't reconnect if already connected or connecting
    if (
      wsRef.current &&
      (wsRef.current.readyState === WebSocket.OPEN ||
        wsRef.current.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    // Close existing connection if in closing state
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    const wsUrl = getWsUrl(token);
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log("[Chat] WebSocket connected");
      setIsConnected(true);
      reconnectAttemptRef.current = 0; // Reset backoff on success

      // Fetch latest unread count on connect
      if (token) {
        chatService.getUnreadCount(token).then((counts) => {
          setUnreadCount(counts.dm_unread);
          setBroadcastUnread(counts.broadcast_unread);
        }).catch(console.error);
      }
    };

    ws.onmessage = (event) => {
      try {
        // Handle multiple messages in one frame (server batches)
        const messages = event.data.split("\n");
        for (const msgStr of messages) {
          if (!msgStr.trim()) continue;
          const envelope: WSEnvelope = JSON.parse(msgStr);
          handleWSMessage(envelope);
        }
      } catch (err) {
        console.error("[Chat] Failed to parse WS message:", err);
      }
    };

    ws.onclose = (event) => {
      console.log("[Chat] WebSocket closed:", event.code, event.reason);

      // Only handle close if this is still the active connection
      // (Prevents stale connections from React Strict Mode from
      //  wiping the ref to the current live connection)
      if (wsRef.current !== ws) {
        console.log("[Chat] Ignoring close from stale connection");
        return;
      }

      setIsConnected(false);
      wsRef.current = null;

      // Reconnect with exponential backoff (unless cleanup)
      if (!isCleaningUpRef.current && token) {
        const delay = Math.min(
          RECONNECT_BASE_DELAY * Math.pow(2, reconnectAttemptRef.current),
          RECONNECT_MAX_DELAY
        );
        console.log(`[Chat] Reconnecting in ${delay}ms...`);
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectAttemptRef.current++;
          connect();
        }, delay);
      }
    };

    ws.onerror = (error) => {
      console.error("[Chat] WebSocket error:", error);
    };

    wsRef.current = ws;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  // Handle incoming WebSocket messages
  const handleWSMessage = useCallback((envelope: WSEnvelope) => {
    switch (envelope.type) {
      case WS_TYPES.NEW_MESSAGE: {
        const msg: ChatMessage = envelope.payload.message;
        messageListenersRef.current.forEach((cb) => cb(msg));
        // Update unread count
        setUnreadCount((prev) => prev + 1);
        // Update conversations list
        setConversations((prev) => {
          const existing = prev.find(
            (c) => c.other_user_id === msg.sender_id
          );
          if (existing) {
            return prev
              .map((c) =>
                c.other_user_id === msg.sender_id
                  ? {
                      ...c,
                      last_message: msg.content,
                      last_message_at: msg.sent_at,
                      last_sender_id: msg.sender_id,
                      unread_count: c.unread_count + 1,
                    }
                  : c
              )
              .sort(
                (a, b) =>
                  new Date(b.last_message_at).getTime() -
                  new Date(a.last_message_at).getTime()
              );
          }
          // New conversation — will be populated on refresh
          return prev;
        });
        break;
      }

      case WS_TYPES.MESSAGE_ACK:
        ackListenersRef.current.forEach((cb) => cb(envelope.payload));
        break;

      case WS_TYPES.READ_UPDATE:
        readUpdateListenersRef.current.forEach((cb) => cb(envelope.payload));
        break;

      case WS_TYPES.DELIVERY_UPDATE:
        deliveryListenersRef.current.forEach((cb) => cb(envelope.payload));
        break;

      case WS_TYPES.ONLINE_STATUS: {
        const { user_id, is_online } = envelope.payload;
        setOnlineUsers((prev) => {
          const next = new Set(prev);
          if (is_online) {
            next.add(user_id);
          } else {
            next.delete(user_id);
          }
          return next;
        });
        // Update conversations
        setConversations((prev) =>
          prev.map((c) =>
            c.other_user_id === user_id ? { ...c, is_online } : c
          )
        );
        break;
      }

      case WS_TYPES.TYPING_START: {
        const userId = envelope.payload.user_id;
        setTypingUsers((prev) => new Map(prev).set(userId, true));
        typingListenersRef.current.forEach((cb) =>
          cb({ userId, isTyping: true })
        );
        // Auto-clear typing after 3s
        const existingTimeout = typingTimeoutsRef.current.get(userId);
        if (existingTimeout) clearTimeout(existingTimeout);
        typingTimeoutsRef.current.set(
          userId,
          setTimeout(() => {
            setTypingUsers((prev) => {
              const next = new Map(prev);
              next.delete(userId);
              return next;
            });
            typingListenersRef.current.forEach((cb) =>
              cb({ userId, isTyping: false })
            );
          }, 3000)
        );
        break;
      }

      case WS_TYPES.TYPING_STOP: {
        const userId = envelope.payload.user_id;
        setTypingUsers((prev) => {
          const next = new Map(prev);
          next.delete(userId);
          return next;
        });
        typingListenersRef.current.forEach((cb) =>
          cb({ userId, isTyping: false })
        );
        break;
      }

      case WS_TYPES.BROADCAST: {
        const broadcast: Broadcast = envelope.payload.broadcast;
        broadcastListenersRef.current.forEach((cb) => cb(broadcast));
        setBroadcastUnread((prev) => prev + 1);
        break;
      }

      case WS_TYPES.ERROR:
        console.error("[Chat] Server error:", envelope.payload);
        break;
    }
  }, []);

  // Connect on auth
  useEffect(() => {
    if (token && user) {
      isCleaningUpRef.current = false;
      connect();
    }

    return () => {
      isCleaningUpRef.current = true;
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close(1000, "cleanup");
        wsRef.current = null;
      }
      // Clear all typing timeouts to prevent memory leaks
      typingTimeoutsRef.current.forEach((t) => clearTimeout(t));
      typingTimeoutsRef.current.clear();
    };
  }, [token, user, connect]);

  // Visibility change handler
  useEffect(() => {
    const handleVisibility = () => {
      if (document.visibilityState === "visible") {
        // Only reconnect if truly disconnected (not connecting)
        if (
          !wsRef.current ||
          wsRef.current.readyState === WebSocket.CLOSED
        ) {
          connect();
        }
        // Refresh unread count
        if (token) {
          chatService
            .getUnreadCount(token)
            .then((counts) => {
              setUnreadCount(counts.dm_unread);
              setBroadcastUnread(counts.broadcast_unread);
            })
            .catch(console.error);
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibility);
    return () =>
      document.removeEventListener("visibilitychange", handleVisibility);
  }, [token, connect]);

  const sendWS = useCallback(
    (type: string, payload: unknown) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type, payload }));
      } else {
        console.warn("[Chat] Cannot send, WebSocket not connected. Type:", type);
      }
    },
    []
  );

  // Actions
  const sendMessage = useCallback(
    (receiverId: string, content: string): string => {
      const tempId = `temp_${Date.now()}_${Math.random().toString(36).slice(2)}`;
      sendWS(WS_TYPES.CHAT_MESSAGE, {
        receiver_id: receiverId,
        content,
        temp_id: tempId,
      });
      return tempId;
    },
    [sendWS]
  );

  const sendReadReceipt = useCallback(
    (senderId: string) => {
      sendWS(WS_TYPES.READ_RECEIPT, { sender_id: senderId });
      // Optimistically update unread
      setConversations((prev) =>
        prev.map((c) =>
          c.other_user_id === senderId ? { ...c, unread_count: 0 } : c
        )
      );
    },
    [sendWS]
  );

  const sendTypingStart = useCallback(
    (userId: string) => {
      sendWS(WS_TYPES.TYPING_START, { user_id: userId });
    },
    [sendWS]
  );

  const sendTypingStop = useCallback(
    (userId: string) => {
      sendWS(WS_TYPES.TYPING_STOP, { user_id: userId });
    },
    [sendWS]
  );

  const sendBroadcast = useCallback(
    (content: string) => {
      sendWS(WS_TYPES.BROADCAST, { content });
    },
    [sendWS]
  );

  const markBroadcastReadAction = useCallback(
    (broadcastId: string) => {
      sendWS(WS_TYPES.BROADCAST_READ, { broadcast_id: broadcastId });
      setBroadcastUnread((prev) => Math.max(0, prev - 1));
    },
    [sendWS]
  );

  const refreshConversations = useCallback(async () => {
    if (!token) return;
    try {
      const convos = await chatService.getConversations(token);
      setConversations(convos || []);
    } catch (err) {
      console.error("[Chat] Failed to refresh conversations:", err);
    }
  }, [token]);

  const refreshUnreadCount = useCallback(async () => {
    if (!token) return;
    try {
      const counts = await chatService.getUnreadCount(token);
      setUnreadCount(counts.dm_unread);
      setBroadcastUnread(counts.broadcast_unread);
    } catch (err) {
      console.error("[Chat] Failed to refresh unread count:", err);
    }
  }, [token]);

  // Listener registration helpers
  const onNewMessage = useCallback(
    (callback: (msg: ChatMessage) => void) => {
      messageListenersRef.current.add(callback);
      return () => {
        messageListenersRef.current.delete(callback);
      };
    },
    []
  );

  const onMessageAck = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (callback: (ack: any) => void) => {
      ackListenersRef.current.add(callback);
      return () => {
        ackListenersRef.current.delete(callback);
      };
    },
    []
  );

  const onReadUpdate = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (callback: (update: any) => void) => {
      readUpdateListenersRef.current.add(callback);
      return () => {
        readUpdateListenersRef.current.delete(callback);
      };
    },
    []
  );

  const onDeliveryUpdate = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (callback: (update: any) => void) => {
      deliveryListenersRef.current.add(callback);
      return () => {
        deliveryListenersRef.current.delete(callback);
      };
    },
    []
  );

  const onBroadcast = useCallback(
    (callback: (broadcast: Broadcast) => void) => {
      broadcastListenersRef.current.add(callback);
      return () => {
        broadcastListenersRef.current.delete(callback);
      };
    },
    []
  );

  const onTyping = useCallback(
    (callback: (data: { userId: string; isTyping: boolean }) => void) => {
      typingListenersRef.current.add(callback);
      return () => {
        typingListenersRef.current.delete(callback);
      };
    },
    []
  );

  return (
    <ChatContext.Provider
      value={{
        conversations,
        unreadCount,
        broadcastUnread,
        onlineUsers,
        typingUsers,
        isConnected,
        sendMessage,
        sendReadReceipt,
        sendTypingStart,
        sendTypingStop,
        sendBroadcast,
        markBroadcastRead: markBroadcastReadAction,
        refreshConversations,
        refreshUnreadCount,
        onNewMessage,
        onMessageAck,
        onReadUpdate,
        onDeliveryUpdate,
        onBroadcast,
        onTyping,
      }}
    >
      {children}
    </ChatContext.Provider>
  );
}

// Safe no-op defaults for when ChatProvider isn't mounted (SSR, pre-auth)
const NOOP = () => {};
const NOOP_UNSUB = () => NOOP;
const EMPTY_CONTEXT: ChatContextType = {
  conversations: [],
  unreadCount: 0,
  broadcastUnread: 0,
  onlineUsers: new Set<string>(),
  typingUsers: new Map<string, boolean>(),
  isConnected: false,
  sendMessage: () => "",
  sendReadReceipt: NOOP,
  sendTypingStart: NOOP,
  sendTypingStop: NOOP,
  sendBroadcast: NOOP,
  markBroadcastRead: NOOP,
  refreshConversations: async () => {},
  refreshUnreadCount: async () => {},
  onNewMessage: NOOP_UNSUB,
  onMessageAck: NOOP_UNSUB,
  onReadUpdate: NOOP_UNSUB,
  onDeliveryUpdate: NOOP_UNSUB,
  onBroadcast: NOOP_UNSUB,
  onTyping: NOOP_UNSUB,
};

export function useChat() {
  const context = useContext(ChatContext);
  // Return safe defaults when outside ChatProvider (SSR or pre-auth)
  return context ?? EMPTY_CONTEXT;
}
