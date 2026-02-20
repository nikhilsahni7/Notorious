"use client";

import { ChatBubble, TypingIndicator } from "@/components/chat/ChatBubble";
import { InlineUnreadBadge } from "@/components/chat/ChatNotificationBadge";
import { useAuth } from "@/contexts/AuthContext";
import { useChat } from "@/contexts/ChatContext";
import { adminService } from "@/services/admin.service";
import {
  Broadcast,
  ChatMessage,
  chatService,
  Conversation,
} from "@/services/chat.service";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowLeft,
  Bell,
  ChevronLeft,
  Megaphone,
  MessageCircle,
  Plus,
  Search,
  Send,
  Users,
  Wifi,
  WifiOff,
  X,
} from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";

interface PendingMessage {
  tempId: string;
  content: string;
  sentAt: string;
  status: "pending" | "sent";
}

interface SimpleUser {
  id: string;
  name: string;
  email: string;
  role: string;
}

export default function ChatPage() {
  const { user, token } = useAuth();
  const {
    conversations,
    onlineUsers,
    typingUsers,
    isConnected,
    broadcastUnread,
    sendMessage,
    sendBroadcast,
    sendReadReceipt,
    sendTypingStart,
    sendTypingStop,
    refreshConversations,
    markBroadcastRead,
    onNewMessage,
    onMessageAck,
    onReadUpdate,
    onDeliveryUpdate,
    onBroadcast,
  } = useChat();

  const router = useRouter();
  const searchParams = useSearchParams();
  const targetUserId = searchParams.get("user");

  const [selectedConv, setSelectedConv] = useState<Conversation | null>(null);
  // For new conversations (no existing conversation yet)
  const [directChatUser, setDirectChatUser] = useState<SimpleUser | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [pendingMessages, setPendingMessages] = useState<PendingMessage[]>([]);
  const pendingMessagesRef = useRef<PendingMessage[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [isLoadingMessages, setIsLoadingMessages] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [showSidebar, setShowSidebar] = useState(true);

  // New Chat modal
  const [showNewChat, setShowNewChat] = useState(false);
  const [allUsers, setAllUsers] = useState<SimpleUser[]>([]);
  const [userSearchQuery, setUserSearchQuery] = useState("");
  const [loadingUsers, setLoadingUsers] = useState(false);

  // Broadcast modal
  const [showBroadcast, setShowBroadcast] = useState(false);
  const [broadcastContent, setBroadcastContent] = useState("");

  // Broadcasts view
  const [sidebarTab, setSidebarTab] = useState<"chats" | "broadcasts">("chats");
  const [broadcasts, setBroadcasts] = useState<Broadcast[]>([]);
  const [selectedBroadcast, setSelectedBroadcast] = useState<Broadcast | null>(null);
  const [loadingBroadcasts, setLoadingBroadcasts] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const hasHandledTargetUser = useRef(false);

  const isAdmin = user?.role === "admin";

  // Load conversations on mount
  useEffect(() => {
    refreshConversations();
  }, [refreshConversations]);

  // Load broadcasts when switching to broadcasts tab
  useEffect(() => {
    if (sidebarTab !== "broadcasts" || !token) return;
    setLoadingBroadcasts(true);
    chatService
      .getBroadcasts(token, 100)
      .then((data) => setBroadcasts(data || []))
      .catch(console.error)
      .finally(() => setLoadingBroadcasts(false));
  }, [sidebarTab, token]);

  // Listen for new broadcasts in real-time
  useEffect(() => {
    const unsub = onBroadcast((broadcast: Broadcast) => {
      setBroadcasts((prev) => [broadcast, ...prev]);
    });
    return unsub;
  }, [onBroadcast]);

  // Handle ?user=ID from URL — auto-create a direct chat target
  useEffect(() => {
    if (!targetUserId || !token || hasHandledTargetUser.current) return;

    // Check if conversation already exists
    const existingConv = conversations.find(
      (c) => c.other_user_id === targetUserId
    );
    if (existingConv) {
      setSelectedConv(existingConv);
      setDirectChatUser(null);
      setShowSidebar(false);
      hasHandledTargetUser.current = true;
      return;
    }

    // No existing conversation — fetch user info and set up direct chat
    if (isAdmin && conversations.length >= 0) {
      // Fetch user list to find the user's name
      adminService
        .listUsers(token, 200)
        .then((users) => {
          const targetUser = users.find(
            (u: SimpleUser) => u.id === targetUserId
          );
          if (targetUser) {
            setDirectChatUser({
              id: targetUser.id,
              name: targetUser.name,
              email: targetUser.email,
              role: targetUser.role,
            });
            setSelectedConv(null);
            setMessages([]);
            setShowSidebar(false);
            hasHandledTargetUser.current = true;
          }
        })
        .catch(console.error);
    }
  }, [targetUserId, conversations, token, isAdmin]);

  // The active chat target (either from existing conversation or new direct chat)
  const activeChatUserId = selectedConv?.other_user_id || directChatUser?.id;
  const activeChatUserName =
    selectedConv?.other_user_name || directChatUser?.name;
  const activeChatOnline =
    (selectedConv?.is_online ||
      (activeChatUserId && onlineUsers.has(activeChatUserId))) ??
    false;

  // Load messages when conversation is selected
  useEffect(() => {
    if (!activeChatUserId || !token) return;

    setIsLoadingMessages(true);
    chatService
      .getMessages(activeChatUserId, token)
      .then((res) => {
        setMessages(res.messages || []);
        setHasMore(res.has_more);
        setPendingMessages([]);
      })
      .catch(console.error)
      .finally(() => setIsLoadingMessages(false));

    // Send read receipt
    sendReadReceipt(activeChatUserId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeChatUserId, token]);

  // Scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, pendingMessages]);

  // Listen for new messages
  useEffect(() => {
    const unsubMessage = onNewMessage((msg) => {
      if (
        activeChatUserId &&
        (msg.sender_id === activeChatUserId ||
          msg.receiver_id === activeChatUserId)
      ) {
        setMessages((prev) => [...prev, msg]);
        if (msg.sender_id === activeChatUserId) {
          sendReadReceipt(activeChatUserId);
        }
      }
      // If this is a new conversation from directChatUser, refresh to get it
      if (directChatUser && msg.sender_id === directChatUser.id) {
        refreshConversations();
      }
    });

    const unsubAck = onMessageAck((ack) => {
      // Use ref to avoid stale closure — pendingMessages state would be stale here
      const pendingContent =
        pendingMessagesRef.current.find((p) => p.tempId === ack.temp_id)?.content || "";

      setPendingMessages((prev) => {
        const updated = prev.filter((p) => p.tempId !== ack.temp_id);
        pendingMessagesRef.current = updated;
        return updated;
      });

      setMessages((prev) => {
        if (prev.some((m) => m.id === ack.message_id)) return prev;
        const confirmedMsg: ChatMessage = {
          id: ack.message_id,
          conversation_id: "",
          sender_id: user?.id || "",
          receiver_id: activeChatUserId || "",
          content: pendingContent,
          sequence_num: ack.sequence_num,
          status: "sent",
          sent_at: ack.sent_at,
        };
        return [...prev, confirmedMsg];
      });

      // After first message sent to a new user, refresh conversations
      if (directChatUser) {
        refreshConversations().then(() => {
          // Switch from directChatUser to the real conversation
          setDirectChatUser(null);
        });
      }
    });

    const unsubRead = onReadUpdate((update) => {
      if (activeChatUserId && update.reader_id === activeChatUserId) {
        setMessages((prev) =>
          prev.map((m) =>
            m.sender_id === user?.id && m.status !== "read"
              ? { ...m, status: "read" as const, read_at: update.read_at }
              : m
          )
        );
      }
    });

    const unsubDelivery = onDeliveryUpdate((update) => {
      if (activeChatUserId && update.receiver_id === activeChatUserId) {
        setMessages((prev) =>
          prev.map((m) =>
            m.sender_id === user?.id && m.status === "sent"
              ? { ...m, status: "delivered" as const }
              : m
          )
        );
      }
    });

    return () => {
      unsubMessage();
      unsubAck();
      unsubRead();
      unsubDelivery();
    };
    // Note: removed pendingMessages from deps — we use pendingMessagesRef instead
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    activeChatUserId,
    directChatUser,
    user,
    onNewMessage,
    onMessageAck,
    onReadUpdate,
    onDeliveryUpdate,
    sendReadReceipt,
    refreshConversations,
  ]);

  // Handle send
  const handleSend = useCallback(() => {
    if (!inputValue.trim() || !activeChatUserId) return;

    const content = inputValue.trim();
    const tempId = sendMessage(activeChatUserId, content);

    const newPending: PendingMessage = {
      tempId,
      content,
      sentAt: new Date().toISOString(),
      status: "pending",
    };
    setPendingMessages((prev) => {
      const updated = [...prev, newPending];
      pendingMessagesRef.current = updated;
      return updated;
    });

    setInputValue("");
    inputRef.current?.focus();

    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
      sendTypingStop(activeChatUserId);
    }
  }, [inputValue, activeChatUserId, sendMessage, sendTypingStop]);

  // Handle typing indicator
  const handleInputChange = useCallback(
    (value: string) => {
      setInputValue(value);
      if (!activeChatUserId) return;

      if (value.trim()) {
        sendTypingStart(activeChatUserId);
        if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
        typingTimeoutRef.current = setTimeout(() => {
          sendTypingStop(activeChatUserId);
        }, 2000);
      } else {
        sendTypingStop(activeChatUserId);
      }
    },
    [activeChatUserId, sendTypingStart, sendTypingStop]
  );

  // Load more messages
  const loadMore = useCallback(async () => {
    if (!activeChatUserId || !token || !hasMore || isLoadingMessages) return;
    const firstMsg = messages[0];
    if (!firstMsg) return;

    setIsLoadingMessages(true);
    try {
      const res = await chatService.getMessages(
        activeChatUserId,
        token,
        firstMsg.sequence_num
      );
      setMessages((prev) => [...(res.messages || []), ...prev]);
      setHasMore(res.has_more);
    } catch (err) {
      console.error("Failed to load more:", err);
    } finally {
      setIsLoadingMessages(false);
    }
  }, [activeChatUserId, token, hasMore, isLoadingMessages, messages]);

  // Load all users for new chat picker (admin only)
  const handleNewChat = useCallback(async () => {
    if (!token) return;
    setShowNewChat(true);
    setLoadingUsers(true);
    try {
      if (isAdmin) {
        const users = await adminService.listUsers(token, 200);
        setAllUsers(
          users
            .filter((u: SimpleUser) => u.id !== user?.id)
            .map((u: SimpleUser) => ({
              id: u.id,
              name: u.name,
              email: u.email,
              role: u.role,
            }))
        );
      }
    } catch (err) {
      console.error("Failed to load users:", err);
    } finally {
      setLoadingUsers(false);
    }
  }, [token, isAdmin, user]);

  // Start conversation with selected user
  const startChatWith = useCallback(
    (targetUser: SimpleUser) => {
      // Check if conversation already exists
      const existing = conversations.find(
        (c) => c.other_user_id === targetUser.id
      );
      if (existing) {
        setSelectedConv(existing);
        setDirectChatUser(null);
      } else {
        setDirectChatUser(targetUser);
        setSelectedConv(null);
        setMessages([]);
      }
      setShowNewChat(false);
      setShowSidebar(false);
      setUserSearchQuery("");
    },
    [conversations]
  );

  // Handle broadcast send
  const handleBroadcastSend = useCallback(() => {
    if (!broadcastContent.trim()) return;
    sendBroadcast(broadcastContent.trim());
    setBroadcastContent("");
    setShowBroadcast(false);
  }, [broadcastContent, sendBroadcast]);

  // Filter conversations
  const filteredConversations = conversations.filter(
    (c) =>
      c.other_user_name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      c.other_user_email?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Filter users for new chat picker
  const filteredUsers = allUsers.filter(
    (u) =>
      u.name.toLowerCase().includes(userSearchQuery.toLowerCase()) ||
      u.email.toLowerCase().includes(userSearchQuery.toLowerCase())
  );

  const isOtherTyping =
    activeChatUserId && typingUsers.get(activeChatUserId);

  const hasActiveChat = selectedConv || directChatUser;
  const hasActiveView = hasActiveChat || selectedBroadcast;

  return (
    <div className="flex h-screen bg-gray-950 text-white">
      {/* Sidebar */}
      <AnimatePresence>
        {showSidebar && (
          <motion.div
            initial={{ x: -300, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: -300, opacity: 0 }}
            className="w-full md:w-80 lg:w-96 border-r border-gray-800 flex flex-col bg-gray-950 md:relative absolute z-20 h-full"
          >
            {/* Header */}
            <div className="p-4 border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => router.push(isAdmin ? '/admin' : '/')}
                    className="p-1.5 rounded-lg hover:bg-gray-800 transition-colors"
                  >
                    <ArrowLeft size={18} />
                  </button>
                  <h1 className="text-lg font-bold bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent">
                    Messages
                  </h1>
                </div>
                <div className="flex items-center gap-2">
                  {isConnected ? (
                    <Wifi size={14} className="text-emerald-400" />
                  ) : (
                    <WifiOff size={14} className="text-red-400" />
                  )}
                  <span className="text-xs text-gray-500">
                    {isConnected ? "Connected" : "Reconnecting..."}
                  </span>
                </div>
              </div>

              {/* Action buttons */}
              <div className="flex gap-2 mb-3">
                <button
                  onClick={handleNewChat}
                  className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl bg-indigo-500/20 text-indigo-400 text-xs font-medium hover:bg-indigo-500/30 transition-colors"
                >
                  <Plus size={14} />
                  New Chat
                </button>
                {isAdmin && (
                  <button
                    onClick={() => setShowBroadcast(true)}
                    className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl bg-purple-500/20 text-purple-400 text-xs font-medium hover:bg-purple-500/30 transition-colors"
                  >
                    <Megaphone size={14} />
                    Broadcast
                  </button>
                )}
              </div>

              {/* Search */}
              <div className="relative">
                <Search
                  size={14}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
                />
                <input
                  type="text"
                  placeholder="Search conversations..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-9 pr-3 py-2 bg-gray-800/50 border border-gray-700/50 rounded-xl text-sm placeholder-gray-500 focus:outline-none focus:border-indigo-500/50 transition-colors"
                />
              </div>
            </div>

            {/* Sidebar Tab Toggle */}
            <div className="flex border-b border-gray-800">
              <button
                onClick={() => { setSidebarTab("chats"); setSelectedBroadcast(null); }}
                className={`flex-1 py-2.5 text-xs font-medium transition-colors relative ${
                  sidebarTab === "chats"
                    ? "text-indigo-400"
                    : "text-gray-500 hover:text-gray-300"
                }`}
              >
                Chats
                {sidebarTab === "chats" && (
                  <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />
                )}
              </button>
              <button
                onClick={() => { setSidebarTab("broadcasts"); setSelectedBroadcast(null); }}
                className={`flex-1 py-2.5 text-xs font-medium transition-colors relative flex items-center justify-center gap-1.5 ${
                  sidebarTab === "broadcasts"
                    ? "text-purple-400"
                    : "text-gray-500 hover:text-gray-300"
                }`}
              >
                <Bell size={12} />
                Announcements
                {broadcastUnread > 0 && (
                  <span className="inline-flex items-center justify-center min-w-[16px] h-4 px-1 rounded-full bg-purple-500 text-white text-[10px] font-bold">
                    {broadcastUnread}
                  </span>
                )}
                {sidebarTab === "broadcasts" && (
                  <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-purple-500" />
                )}
              </button>
            </div>

            <div className="flex-1 overflow-y-auto">
              {sidebarTab === "chats" ? (
                /* Conversations List */
                filteredConversations.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-48 text-gray-500">
                    <MessageCircle size={32} className="mb-2 opacity-50" />
                    <p className="text-sm">No conversations yet</p>
                    <p className="text-xs mt-1 text-gray-600">
                      Click &quot;New Chat&quot; to start one
                    </p>
                  </div>
                ) : (
                  filteredConversations.map((conv) => (
                    <motion.button
                      key={conv.id}
                      whileHover={{ backgroundColor: "rgba(99,102,241,0.08)" }}
                      onClick={() => {
                        setSelectedConv(conv);
                        setDirectChatUser(null);
                        setSelectedBroadcast(null);
                        setShowSidebar(false);
                      }}
                      className={`w-full p-4 flex items-center gap-3 border-b border-gray-800/50 transition-colors ${
                        selectedConv?.id === conv.id
                          ? "bg-indigo-500/10 border-l-2 border-l-indigo-500"
                          : ""
                      }`}
                    >
                      {/* Avatar */}
                      <div className="relative flex-shrink-0">
                        <div className="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-sm font-bold">
                          {conv.other_user_name?.charAt(0)?.toUpperCase() || "?"}
                        </div>
                        <div
                          className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-gray-950 ${
                            conv.is_online || onlineUsers.has(conv.other_user_id)
                              ? "bg-emerald-400"
                              : "bg-gray-600"
                          }`}
                        />
                      </div>

                      {/* Info */}
                      <div className="flex-1 min-w-0 text-left">
                        <div className="flex items-center justify-between">
                          <span className="font-medium text-sm truncate">
                            {conv.other_user_name}
                          </span>
                          <span className="text-[10px] text-gray-500 flex-shrink-0">
                            {formatTime(conv.last_message_at)}
                          </span>
                        </div>
                        <div className="flex items-center justify-between mt-0.5">
                          <p className="text-xs text-gray-400 truncate max-w-[180px]">
                            {typingUsers.get(conv.other_user_id)
                              ? "typing..."
                              : conv.last_message || "Start a conversation"}
                          </p>
                          <InlineUnreadBadge count={conv.unread_count} />
                        </div>
                      </div>
                    </motion.button>
                  ))
                )
              ) : (
                /* Broadcasts List */
                loadingBroadcasts ? (
                  <div className="text-center py-12 text-gray-500 text-sm">Loading announcements...</div>
                ) : broadcasts.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-48 text-gray-500">
                    <Bell size={32} className="mb-2 opacity-50" />
                    <p className="text-sm">No announcements yet</p>
                  </div>
                ) : (
                  broadcasts.map((b) => (
                    <button
                      key={b.id}
                      onClick={() => {
                        setSelectedBroadcast(b);
                        setSelectedConv(null);
                        setDirectChatUser(null);
                        setShowSidebar(false);
                        if (!b.is_read) {
                          markBroadcastRead(b.id);
                          setBroadcasts((prev) =>
                            prev.map((br) =>
                              br.id === b.id ? { ...br, is_read: true } : br
                            )
                          );
                        }
                      }}
                      className={`w-full p-4 flex items-start gap-3 border-b border-gray-800/50 transition-colors text-left hover:bg-purple-500/5 ${
                        selectedBroadcast?.id === b.id
                          ? "bg-purple-500/10 border-l-2 border-l-purple-500"
                          : ""
                      } ${!b.is_read ? "bg-purple-500/5" : ""}`}
                    >
                      <div className="w-10 h-10 rounded-full bg-gradient-to-br from-purple-500 to-pink-600 flex items-center justify-center text-sm flex-shrink-0">
                        <Megaphone size={18} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <span className="font-medium text-sm flex items-center gap-1.5">
                            {isAdmin && b.sender_id === user?.id ? (
                              <span className="text-purple-400">You (Sent)</span>
                            ) : (
                              <span>Admin Announcement</span>
                            )}
                            {!b.is_read && (
                              <span className="w-2 h-2 rounded-full bg-purple-400 inline-block" />
                            )}
                          </span>
                          <span className="text-[10px] text-gray-500 flex-shrink-0">
                            {formatTime(b.sent_at)}
                          </span>
                        </div>
                        <p className="text-xs text-gray-400 truncate mt-0.5">
                          {b.content}
                        </p>
                      </div>
                    </button>
                  ))
                )
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Chat Area */}
      <div className="flex-1 flex flex-col min-w-0">
        {selectedBroadcast ? (
          /* Broadcast Detail View */
          <>
            <div className="px-4 py-3 border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm flex items-center gap-3">
              <button
                onClick={() => { setSelectedBroadcast(null); setShowSidebar(true); }}
                className="p-1.5 rounded-lg hover:bg-gray-800 transition-colors"
              >
                <ChevronLeft size={20} />
              </button>
              <div className="w-9 h-9 rounded-full bg-gradient-to-br from-purple-500 to-pink-600 flex items-center justify-center">
                <Megaphone size={16} />
              </div>
              <div>
                <h2 className="font-semibold text-sm">Announcement</h2>
                <p className="text-xs text-gray-400">
                  {new Date(selectedBroadcast.sent_at).toLocaleDateString([], {
                    weekday: "short",
                    month: "short",
                    day: "numeric",
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </p>
              </div>
            </div>
            <div className="flex-1 overflow-y-auto p-6 flex items-start justify-center">
              <div className="max-w-lg w-full">
                <div className="bg-gradient-to-br from-purple-500/10 to-pink-500/10 border border-purple-500/20 rounded-2xl p-6">
                  <div className="flex items-center gap-2 mb-4">
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-purple-500 to-pink-600 flex items-center justify-center">
                      <Megaphone size={14} />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-purple-300">
                        {isAdmin && selectedBroadcast.sender_id === user?.id
                          ? "You sent this broadcast"
                          : "Admin Announcement"}
                      </p>
                      <p className="text-[10px] text-gray-500">
                        {new Date(selectedBroadcast.sent_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                  <p className="text-[15px] leading-relaxed whitespace-pre-wrap text-gray-200">
                    {selectedBroadcast.content}
                  </p>
                </div>
              </div>
            </div>
          </>
        ) : hasActiveChat ? (
          <>
            {/* Chat Header */}
            <div className="px-4 py-3 border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm flex items-center gap-3">
              <button
                onClick={() => {
                  setSelectedConv(null);
                  setDirectChatUser(null);
                  setShowSidebar(true);
                }}
                className="p-1.5 rounded-lg hover:bg-gray-800 transition-colors"
              >
                <ChevronLeft size={20} />
              </button>
              <div className="relative">
                <div className="w-9 h-9 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-sm font-bold">
                  {activeChatUserName?.charAt(0)?.toUpperCase() || "?"}
                </div>
                <div
                  className={`absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-gray-900 ${
                    activeChatOnline ? "bg-emerald-400" : "bg-gray-600"
                  }`}
                />
              </div>
              <div>
                <h2 className="font-semibold text-sm">
                  {activeChatUserName}
                </h2>
                <p className="text-xs text-gray-400">
                  {isOtherTyping
                    ? "typing..."
                    : activeChatOnline
                    ? "Online"
                    : "Offline"}
                </p>
              </div>
            </div>

            {/* Messages */}
            <div
              ref={messagesContainerRef}
              className="flex-1 overflow-y-auto p-4 space-y-1"
              onScroll={(e) => {
                const el = e.currentTarget;
                if (el.scrollTop < 100 && hasMore && !isLoadingMessages) {
                  loadMore();
                }
              }}
            >
              {hasMore && (
                <button
                  onClick={loadMore}
                  disabled={isLoadingMessages}
                  className="w-full py-2 text-xs text-indigo-400 hover:text-indigo-300 disabled:opacity-50"
                >
                  {isLoadingMessages ? "Loading..." : "Load earlier messages"}
                </button>
              )}

              {messages.length === 0 &&
                pendingMessages.length === 0 &&
                !isLoadingMessages && (
                  <div className="flex flex-col items-center justify-center py-20 text-gray-500">
                    <MessageCircle size={40} className="mb-3 opacity-30" />
                    <p className="text-sm">No messages yet</p>
                    <p className="text-xs mt-1">
                      Send a message to start the conversation
                    </p>
                  </div>
                )}

              {messages.map((msg, idx) => {
                const prevMsg = messages[idx - 1];
                const showDateSeparator = shouldShowDateSeparator(
                  prevMsg?.sent_at,
                  msg.sent_at
                );
                return (
                  <div key={msg.id}>
                    {showDateSeparator && (
                      <DateSeparator dateStr={msg.sent_at} />
                    )}
                    <ChatBubble
                      message={msg}
                      isOwn={msg.sender_id === user?.id}
                    />
                  </div>
                );
              })}

              {pendingMessages.map((pm) => (
                <motion.div
                  key={pm.tempId}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 0.7, y: 0 }}
                  className="flex justify-end mb-2"
                >
                  <div className="max-w-[75%] rounded-2xl rounded-br-sm px-4 py-2.5 bg-gradient-to-br from-indigo-500/70 to-purple-600/70 text-white">
                    <p className="text-sm">{pm.content}</p>
                    <div className="flex items-center justify-end gap-1 mt-1">
                      <span className="text-[10px] opacity-50">
                        Sending...
                      </span>
                    </div>
                  </div>
                </motion.div>
              ))}

              {isOtherTyping && (
                <div className="flex justify-start mb-2">
                  <TypingIndicator />
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>

            {/* Input */}
            <div className="p-4 border-t border-gray-800 bg-gray-900/50">
              <div className="flex items-end gap-2">
                <textarea
                  ref={inputRef}
                  value={inputValue}
                  onChange={(e) => handleInputChange(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      handleSend();
                    }
                  }}
                  placeholder="Type a message..."
                  rows={1}
                  className="flex-1 bg-gray-800/50 border border-gray-700/50 rounded-xl px-4 py-3 text-sm resize-none placeholder-gray-500 focus:outline-none focus:border-indigo-500/50 transition-colors max-h-32"
                  style={{
                    height: "auto",
                    minHeight: "44px",
                  }}
                  onInput={(e) => {
                    const target = e.currentTarget;
                    target.style.height = "auto";
                    target.style.height =
                      Math.min(target.scrollHeight, 128) + "px";
                  }}
                />
                <button
                  onClick={handleSend}
                  disabled={!inputValue.trim()}
                  className="p-3 rounded-xl bg-gradient-to-r from-indigo-500 to-purple-600 hover:from-indigo-600 hover:to-purple-700 disabled:opacity-30 disabled:cursor-not-allowed transition-all shadow-lg shadow-indigo-500/20 hover:shadow-indigo-500/40"
                >
                  <Send size={18} />
                </button>
              </div>
            </div>
          </>
        ) : (
          /* Empty state */
          <div className="flex-1 flex flex-col items-center justify-center text-gray-500">
            <div className="w-20 h-20 rounded-2xl bg-gray-800/50 flex items-center justify-center mb-4">
              <MessageCircle size={36} className="opacity-50" />
            </div>
            <h3 className="text-lg font-semibold text-gray-400 mb-1">
              {conversations.length === 0
                ? "Start a conversation"
                : "Select a conversation"}
            </h3>
            <p className="text-sm text-center max-w-xs">
              {conversations.length === 0
                ? 'Click "New Chat" in the sidebar to start messaging'
                : "Choose from your existing conversations or start a new one"}
            </p>
            {!showSidebar && (
              <button
                onClick={() => setShowSidebar(true)}
                className="mt-4 px-4 py-2 rounded-lg bg-indigo-500/20 text-indigo-400 text-sm hover:bg-indigo-500/30 transition-colors"
              >
                Show conversations
              </button>
            )}
          </div>
        )}
      </div>

      {/* New Chat Modal */}
      <AnimatePresence>
        {showNewChat && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4"
            onClick={() => setShowNewChat(false)}
          >
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
              className="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl"
            >
              <div className="p-4 border-b border-gray-700 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Users size={18} className="text-indigo-400" />
                  <h3 className="font-semibold">New Chat</h3>
                </div>
                <button
                  onClick={() => setShowNewChat(false)}
                  className="p-1 rounded-lg hover:bg-gray-800"
                >
                  <X size={18} />
                </button>
              </div>

              <div className="p-4">
                <div className="relative mb-3">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
                  />
                  <input
                    type="text"
                    placeholder="Search users..."
                    value={userSearchQuery}
                    onChange={(e) => setUserSearchQuery(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 bg-gray-800/50 border border-gray-700/50 rounded-xl text-sm placeholder-gray-500 focus:outline-none focus:border-indigo-500/50"
                    autoFocus
                  />
                </div>

                <div className="max-h-72 overflow-y-auto space-y-1">
                  {loadingUsers ? (
                    <div className="text-center py-8 text-gray-500 text-sm">
                      Loading users...
                    </div>
                  ) : filteredUsers.length === 0 ? (
                    <div className="text-center py-8 text-gray-500 text-sm">
                      No users found
                    </div>
                  ) : (
                    filteredUsers.map((u) => (
                      <button
                        key={u.id}
                        onClick={() => startChatWith(u)}
                        className="w-full flex items-center gap-3 p-3 rounded-xl hover:bg-gray-800/50 transition-colors text-left"
                      >
                        <div className="relative">
                          <div className="w-9 h-9 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-xs font-bold">
                            {u.name.charAt(0).toUpperCase()}
                          </div>
                          <div
                            className={`absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-gray-900 ${
                              onlineUsers.has(u.id)
                                ? "bg-emerald-400"
                                : "bg-gray-600"
                            }`}
                          />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate">
                            {u.name}
                          </p>
                          <p className="text-xs text-gray-500 truncate">
                            {u.email}
                          </p>
                        </div>
                        <span className="text-[10px] px-2 py-0.5 rounded-full bg-gray-800 text-gray-400">
                          {u.role}
                        </span>
                      </button>
                    ))
                  )}
                </div>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Broadcast Modal */}
      <AnimatePresence>
        {showBroadcast && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4"
            onClick={() => setShowBroadcast(false)}
          >
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
              className="bg-gray-900 border border-gray-700 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl"
            >
              <div className="p-4 border-b border-gray-700 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Megaphone size={18} className="text-purple-400" />
                  <h3 className="font-semibold">Broadcast to All Users</h3>
                </div>
                <button
                  onClick={() => setShowBroadcast(false)}
                  className="p-1 rounded-lg hover:bg-gray-800"
                >
                  <X size={18} />
                </button>
              </div>

              <div className="p-4">
                <p className="text-xs text-gray-400 mb-3">
                  This message will be sent to all active users as an
                  announcement.
                </p>
                <textarea
                  value={broadcastContent}
                  onChange={(e) => setBroadcastContent(e.target.value)}
                  placeholder="Type your announcement..."
                  rows={4}
                  className="w-full bg-gray-800/50 border border-gray-700/50 rounded-xl px-4 py-3 text-sm resize-none placeholder-gray-500 focus:outline-none focus:border-purple-500/50 mb-3"
                  autoFocus
                />
                <button
                  onClick={handleBroadcastSend}
                  disabled={!broadcastContent.trim()}
                  className="w-full py-2.5 rounded-xl bg-gradient-to-r from-purple-500 to-pink-600 hover:from-purple-600 hover:to-pink-700 disabled:opacity-30 disabled:cursor-not-allowed text-sm font-medium transition-all flex items-center justify-center gap-2"
                >
                  <Megaphone size={16} />
                  Send Broadcast
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function formatTime(dateStr: string): string {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  const now = new Date();

  const isToday =
    date.getDate() === now.getDate() &&
    date.getMonth() === now.getMonth() &&
    date.getFullYear() === now.getFullYear();

  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  const isYesterday =
    date.getDate() === yesterday.getDate() &&
    date.getMonth() === yesterday.getMonth() &&
    date.getFullYear() === yesterday.getFullYear();

  if (isToday) {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  } else if (isYesterday) {
    return "Yesterday";
  } else if (now.getTime() - date.getTime() < 7 * 86400000) {
    return date.toLocaleDateString([], { weekday: "short" });
  } else {
    return date.toLocaleDateString([], { month: "short", day: "numeric" });
  }
}

function shouldShowDateSeparator(
  prevDateStr: string | undefined,
  currentDateStr: string
): boolean {
  if (!prevDateStr) return true; // First message always shows date
  const prev = new Date(prevDateStr);
  const curr = new Date(currentDateStr);
  return (
    prev.getDate() !== curr.getDate() ||
    prev.getMonth() !== curr.getMonth() ||
    prev.getFullYear() !== curr.getFullYear()
  );
}

function formatDateSeparator(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();

  const isToday =
    date.getDate() === now.getDate() &&
    date.getMonth() === now.getMonth() &&
    date.getFullYear() === now.getFullYear();

  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  const isYesterday =
    date.getDate() === yesterday.getDate() &&
    date.getMonth() === yesterday.getMonth() &&
    date.getFullYear() === yesterday.getFullYear();

  if (isToday) return "Today";
  if (isYesterday) return "Yesterday";

  const sameYear = date.getFullYear() === now.getFullYear();
  if (sameYear) {
    return date.toLocaleDateString([], {
      weekday: "long",
      month: "long",
      day: "numeric",
    });
  }
  return date.toLocaleDateString([], {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
  });
}

function DateSeparator({ dateStr }: { dateStr: string }) {
  return (
    <div className="flex items-center gap-3 my-4">
      <div className="flex-1 h-px bg-gray-800" />
      <span className="text-xs text-gray-500 font-medium px-2">
        {formatDateSeparator(dateStr)}
      </span>
      <div className="flex-1 h-px bg-gray-800" />
    </div>
  );
}
