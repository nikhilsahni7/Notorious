import { API_CONFIG } from "@/config/api";
import { apiRequest } from "@/lib/api-client";

export interface User {
  id: string;
  email: string;
  name: string;
  phone: string;
  role: string;
  region: string; // "pan-india" or "delhi-ncr"
  daily_search_limit: number;
  device_limit: number;
  searches_used_today: number;
  total_searches: number; // Total searches done overall
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserMetadata {
  id: string;
  user_id: string;
  ip_address?: string;
  country?: string;
  country_code?: string;
  city?: string;
  latitude?: number;
  longitude?: number;
  timezone?: string;
  device_type?: string;
  browser?: string;
  browser_version?: string;
  os?: string;
  os_version?: string;
  user_agent?: string;
  created_at: string;
}

export interface UserSession {
  id: string;
  user_id: string;
  device_name: string;
  device_os: string;
  device_type: string;
  ip_address: string;
  location: string;
  last_active: string;
  created_at: string;
}

export interface UserWithMetadata {
  user: User;
  metadata: UserMetadata | null;
  sessions: UserSession[] | null;
}

export interface UserRequest {
  id: string;
  email: string;
  name: string;
  phone: string;
  requested_searches_per_day: number;
  status: string;
  created_at: string;
  admin_notes?: string; // Deprecated - keeping for backward compatibility
  admin_note?: string; // New field
  reviewed_by?: string; // UUID of admin who reviewed
  reviewed_at?: string; // Timestamp of review
  ip_address?: string;
  country?: string;
  city?: string;
  device_type?: string;
  browser?: string;
  os?: string;
  user_agent?: string;
}

export interface SearchHistoryItem {
  id: string;
  user_id: string;
  user_email: string;
  user_name: string;
  query: string;
  total_results: number;
  searched_at: string;
}

export interface PasswordChangeRequestWithUser {
  id: string;
  user_id: string;
  user_email: string;
  user_name: string;
  reason: string;
  status: string;
  admin_notes?: string;
  created_at: string;
  updated_at: string;
}

export interface AdminSession {
  id: string;
  admin_id: string;
  admin_email: string;
  admin_name: string;
  ip_address?: string;
  country?: string;
  country_code?: string;
  city?: string;
  latitude?: number;
  longitude?: number;
  timezone?: string;
  device_type?: string;
  browser?: string;
  browser_version?: string;
  os?: string;
  os_version?: string;
  user_agent?: string;
  is_active: boolean;
  created_at: string;
  last_used_at: string;
  expires_at: string;
}

export interface DashboardStats {
  pending_user_requests: number;
  pending_password_requests: number;
  total_users: number;
  active_users: number;
  total_searches: number;
}

// ── Per-user stats ──────────────────────────────────────────────────────────

export interface TermFreq {
  query: string;
  count: number;
}

export interface DayVolume {
  date: string;
  count: number;
}

export interface UserStats {
  identity: {
    id: string;
    email: string;
    name: string;
    created_at: string;
    is_active: boolean;
    device_limit: number;
    daily_search_limit: number;
    searches_used_today: number;
    region: string;
  };
  sessions: {
    devices_registered: number;
    last_login: string | null;
  };
  search_behavior: {
    total_searches: number;
    avg_searches_per_day: string;
    first_search_at: string | null;
    last_search_at: string | null;
    top_terms: TermFreq[];
    daily_volume: DayVolume[];
    peak_hour: number;
    peak_hour_formatted: string;
    zero_result_searches: number;
    zero_result_pct: string;
  };
  security: {
    total_password_reset_requests: number;
    last_password_reset_at: string | null;
    devices_registered: number;
    device_limit: number;
  };
  engagement: {
    first_search_at: string | null;
    last_search_at: string | null;
    longest_gap_days: number;
  };
}

// ── System-wide stats ───────────────────────────────────────────────────────

export interface HourBucket {
  hour: number;
  label: string;
  count: number;
}

export interface DowBucket {
  dow: number;
  day_name: string;
  count: number;
}

export interface MonthBucket {
  month: string;
  count: number;
}

export interface ActiveUser {
  id: string;
  name: string;
  email: string;
  search_count: number;
}

export interface DeviceBucket {
  device_count: number;
  user_count: number;
}

export interface DeviceExceeded {
  id: string;
  name: string;
  email: string;
  device_limit: number;
  session_count: number;
}

export interface SystemStats {
  search_volume: {
    total_all_time: number;
    total_last_30_days: number;
    avg_daily: string;
    avg_per_user_per_day: string;
    daily_trend: DayVolume[];
    peak_hour: number;
    peak_hour_formatted: string;
  };
  user_patterns: {
    total_users: number;
    active_users_last_30d: number;
    avg_searches_per_user: string;
    most_active_users: ActiveUser[];
    device_distribution: DeviceBucket[];
  };
  search_patterns: {
    top_terms: TermFreq[];
    zero_result_count: number;
    zero_result_pct: string;
  };
  security: {
    total_password_resets: number;
    password_resets_last_30_days: number;
    users_exceeding_device_limit: DeviceExceeded[];
  };
  time_distributions: {
    by_hour: HourBucket[];
    by_day_of_week: DowBucket[];
    by_month: MonthBucket[];
  };
}

export const adminService = {
  // Users
  listUsers: async (
    token: string,
    limit = 50,
    offset = 0,
    search = "",
    region = "all"
  ): Promise<{
    users: User[];
    total: number;
    counts: { all: number; "pan-india": number; "delhi-ncr": number };
  }> => {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    });
    if (search.trim()) {
      params.set("search", search.trim());
    }
    if (region && region !== "all") {
      params.set("region", region);
    }
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}?${params.toString()}`, {
      method: "GET",
      token,
    });
  },

  getUser: async (userId: string, token: string): Promise<User> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}`, {
      method: "GET",
      token,
    });
  },

  getUserDetails: async (
    userId: string,
    token: string
  ): Promise<UserWithMetadata> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}/details`, {
      method: "GET",
      token,
    });
  },

  createUser: async (
    userData: Partial<User> & { password: string },
    token: string
  ): Promise<User> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.USERS, {
      method: "POST",
      body: JSON.stringify(userData),
      token,
    });
  },

  updateUser: async (
    userId: string,
    userData: Partial<User>,
    token: string
  ): Promise<User> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}`, {
      method: "PUT",
      body: JSON.stringify(userData),
      token,
    });
  },

  deleteUser: async (userId: string, token: string): Promise<void> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}`, {
      method: "DELETE",
      token,
    });
  },

  changeUserPassword: async (
    userId: string,
    newPassword: string,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}/change-password`,
      {
        method: "POST",
        body: JSON.stringify({ new_password: newPassword }),
        token,
      }
    );
  },

  // Bulk update users status (activate/deactivate multiple users at once)
  bulkUpdateUsers: async (
    userIds: string[],
    isActive: boolean,
    token: string
  ): Promise<{ message: string; updated: number }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.BULK_UPDATE_USERS, {
      method: "POST",
      body: JSON.stringify({ user_ids: userIds, is_active: isActive }),
      token,
    });
  },

  // Toggle single user status (convenience method)
  toggleUserStatus: async (
    userId: string,
    isActive: boolean,
    token: string
  ): Promise<{ message: string; updated: number }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.BULK_UPDATE_USERS, {
      method: "POST",
      body: JSON.stringify({ user_ids: [userId], is_active: isActive }),
      token,
    });
  },

  // Get online users (for presence indicators)
  getOnlineUsers: async (
    token: string
  ): Promise<{ online_user_ids: string[] }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.ONLINE_USERS, {
      method: "GET",
      token,
    });
  },

  // User Requests
  listUserRequests: async (
    token: string,
    status = "pending",
    limit = 100
  ): Promise<UserRequest[]> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USER_REQUESTS}?status=${status}&limit=${limit}`,
      {
        method: "GET",
        token,
      }
    );
  },

  approveUserRequest: async (
    requestId: string,
    token: string,
    adminNote?: string
  ): Promise<{ message: string; request: UserRequest }> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USER_REQUESTS}/${requestId}/approve`,
      {
        method: "POST",
        body: JSON.stringify({
          admin_note: adminNote,
        }),
        token,
      }
    );
  },

  rejectUserRequest: async (
    requestId: string,
    reason: string,
    token: string
  ): Promise<void> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USER_REQUESTS}/${requestId}/reject`,
      {
        method: "POST",
        body: JSON.stringify({ reason }),
        token,
      }
    );
  },

  // Search History
  getSearchHistory: async (
    token: string,
    limit = 50,
    offset = 0,
    search = ""
  ): Promise<{ history: SearchHistoryItem[]; total: number }> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.SEARCH_HISTORY}?limit=${limit}&offset=${offset}&search=${encodeURIComponent(search)}`,
      {
        method: "GET",
        token,
      }
    );
  },

  getUserSearchHistory: async (
    userId: string,
    token: string,
    limit = 50,
    offset = 0
  ): Promise<SearchHistoryItem[]> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}/search-history?limit=${limit}&offset=${offset}`,
      {
        method: "GET",
        token,
      }
    );
  },

  // Password Change Requests
  listPasswordChangeRequests: async (
    token: string,
    status = "pending"
  ): Promise<PasswordChangeRequestWithUser[]> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.PASSWORD_CHANGE_REQUESTS}?status=${status}&limit=100`,
      {
        method: "GET",
        token,
      }
    );
  },

  approvePasswordChangeRequest: async (
    requestId: string,
    newPassword: string,
    adminNotes: string | undefined,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.PASSWORD_CHANGE_REQUESTS}/${requestId}/approve`,
      {
        method: "POST",
        body: JSON.stringify({
          new_password: newPassword,
          admin_notes: adminNotes,
        }),
        token,
      }
    );
  },

  rejectPasswordChangeRequest: async (
    requestId: string,
    adminNotes: string,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.PASSWORD_CHANGE_REQUESTS}/${requestId}/reject`,
      {
        method: "POST",
        body: JSON.stringify({ admin_notes: adminNotes }),
        token,
      }
    );
  },

  // Sessions
  getAdminSessions: async (
    token: string,
    limit = 100
  ): Promise<AdminSession[]> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.SESSIONS}?limit=${limit}`, {
      method: "GET",
      token,
    });
  },

  invalidateSession: async (
    sessionId: string,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.SESSIONS}/${sessionId}`, {
      method: "DELETE",
      token,
    });
  },

  // Dashboard Stats
  getDashboardStats: async (token: string): Promise<DashboardStats> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.DASHBOARD_STATS, {
      method: "GET",
      token,
    });
  },

  // Generate EOD Report for User
  generateUserEOD: async (userId: string, token: string): Promise<Blob> => {
    const response = await fetch(
      `${API_CONFIG.BASE_URL}/api/admin/users/${userId}/eod-report`,
      {
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }
    );

    if (!response.ok) {
      throw new Error("Failed to generate EOD report");
    }

    return response.blob();
  },

  // Revoke User Session
  revokeUserSession: async (
    sessionId: string,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USER_SESSIONS}/${sessionId}`, {
      method: "DELETE",
      token,
    });
  },

  // Per-user detailed stats
  getUserStats: async (userId: string, token: string): Promise<UserStats> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USER_STATS}/${userId}`, {
      method: "GET",
      token,
    });
  },

  // System-wide stats
  getSystemStats: async (token: string): Promise<SystemStats> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.SYSTEM_STATS, {
      method: "GET",
      token,
    });
  },
};

