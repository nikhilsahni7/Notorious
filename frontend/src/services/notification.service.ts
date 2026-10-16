import { API_CONFIG } from "@/config/api";
import { apiRequest } from "@/lib/api-client";

export type NotificationType =
  | "quota_reset"
  | "watchlist_match"
  | "export_ready"
  | "quota_warning";

export interface AppNotification {
  id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  body: string;
  payload?: {
    url?: string;
    label?: string;
    [key: string]: unknown;
  };
  created_at: string;
  read_at?: string | null;
}

export interface NotificationListResponse {
  notifications: AppNotification[];
  total: number;
  limit: number;
  offset: number;
}

export interface NotificationPrefs {
  [key: string]: boolean;
}

export const notificationService = {
  list: async (
    token: string,
    params: { unread?: boolean; limit?: number; offset?: number } = {}
  ): Promise<NotificationListResponse> => {
    const search = new URLSearchParams();
    if (params.unread) search.set("unread", "true");
    search.set("limit", String(params.limit ?? 50));
    search.set("offset", String(params.offset ?? 0));
    return apiRequest(`${API_CONFIG.ENDPOINTS.NOTIFICATIONS.BASE}?${search.toString()}`, {
      method: "GET",
      token,
    });
  },

  getUnreadCount: async (token: string): Promise<{ unread: number }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.NOTIFICATIONS.UNREAD_COUNT, {
      method: "GET",
      token,
    });
  },

  markRead: async (id: string, token: string): Promise<void> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.NOTIFICATIONS.BASE}/${id}/read`, {
      method: "POST",
      token,
    });
  },

  markAllRead: async (token: string): Promise<void> => {
    return apiRequest(API_CONFIG.ENDPOINTS.NOTIFICATIONS.READ_ALL, {
      method: "POST",
      token,
    });
  },

  getPrefs: async (token: string): Promise<{ prefs: NotificationPrefs }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.NOTIFICATIONS.PREFS, {
      method: "GET",
      token,
    });
  },

  updatePrefs: async (prefs: NotificationPrefs, token: string): Promise<{ prefs: NotificationPrefs }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.NOTIFICATIONS.PREFS, {
      method: "PUT",
      body: JSON.stringify({ prefs }),
      token,
    });
  },

  getPushPublicKey: async (token: string): Promise<{ public_key: string }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.PUSH.PUBLIC_KEY, {
      method: "GET",
      token,
    });
  },

  subscribePush: async (subscription: PushSubscriptionPayload, token: string): Promise<void> => {
    return apiRequest(API_CONFIG.ENDPOINTS.PUSH.SUBSCRIBE, {
      method: "POST",
      body: JSON.stringify(subscription),
      token,
    });
  },

  unsubscribePush: async (endpoint: string, token: string): Promise<void> => {
    return apiRequest(API_CONFIG.ENDPOINTS.PUSH.UNSUBSCRIBE, {
      method: "POST",
      body: JSON.stringify({ endpoint }),
      token,
    });
  },
};

export interface PushSubscriptionPayload {
  endpoint: string;
  keys: { p256dh: string; auth: string };
  device_type?: string;
}
