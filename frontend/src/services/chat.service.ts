import { API_CONFIG } from "@/config/api";
import { apiRequest } from "@/lib/api-client";

// Types
export interface ChatMessage {
  id: string;
  conversation_id: string;
  sender_id: string;
  receiver_id: string;
  content: string;
  sequence_num: number;
  status: "sent" | "delivered" | "read";
  sent_at: string;
  delivered_at?: string;
  read_at?: string;
}

export interface Conversation {
  id: string;
  other_user_id: string;
  other_user_name: string;
  other_user_email: string;
  other_user_role: string;
  last_message?: string;
  last_message_at: string;
  last_sender_id?: string;
  unread_count: number;
  is_online: boolean;
}

export interface Broadcast {
  id: string;
  sender_id: string;
  sender_name: string;
  content: string;
  sent_at: string;
  is_read: boolean;
  read_at?: string;
}

export interface UnreadCount {
  dm_unread: number;
  broadcast_unread: number;
  total_unread: number;
}

export interface MessagesResponse {
  messages: ChatMessage[];
  has_more: boolean;
}

export interface UnreadPerUser {
  unread_per_user: Record<string, number>;
}

// Service
export const chatService = {
  getConversations: async (token: string): Promise<Conversation[]> => {
    return apiRequest(API_CONFIG.ENDPOINTS.CHAT.CONVERSATIONS, {
      method: "GET",
      token,
    });
  },

  getMessages: async (
    userId: string,
    token: string,
    beforeSeq?: number,
    limit?: number
  ): Promise<MessagesResponse> => {
    let url = `${API_CONFIG.ENDPOINTS.CHAT.MESSAGES}/${userId}`;
    const params = new URLSearchParams();
    if (beforeSeq) params.set("before_seq", String(beforeSeq));
    if (limit) params.set("limit", String(limit));
    const qs = params.toString();
    if (qs) url += `?${qs}`;

    return apiRequest(url, {
      method: "GET",
      token,
    });
  },

  getUnreadCount: async (token: string): Promise<UnreadCount> => {
    return apiRequest(API_CONFIG.ENDPOINTS.CHAT.UNREAD, {
      method: "GET",
      token,
    });
  },

  markAsRead: async (
    userId: string,
    token: string
  ): Promise<{ marked_read: number }> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.CHAT.READ}/${userId}`, {
      method: "POST",
      token,
    });
  },

  getBroadcasts: async (
    token: string,
    limit?: number
  ): Promise<Broadcast[]> => {
    let url = API_CONFIG.ENDPOINTS.CHAT.BROADCASTS;
    if (limit) url += `?limit=${limit}`;
    return apiRequest(url, {
      method: "GET",
      token,
    });
  },

  markBroadcastRead: async (
    broadcastId: string,
    token: string
  ): Promise<{ status: string }> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.CHAT.BROADCASTS}/${broadcastId}/read`,
      {
        method: "POST",
        token,
      }
    );
  },

  getOnlineUsers: async (
    token: string
  ): Promise<{ online_user_ids: string[] }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.CHAT.ONLINE, {
      method: "GET",
      token,
    });
  },

  getUnreadPerUser: async (token: string): Promise<UnreadPerUser> => {
    return apiRequest(API_CONFIG.ENDPOINTS.CHAT.UNREAD_PER_USER, {
      method: "GET",
      token,
    });
  },
};
