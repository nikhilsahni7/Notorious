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

export interface UserWithMetadata {
  user: User;
  metadata: UserMetadata | null;
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

export interface RequestCounts {
  pending_user_requests: number;
  pending_password_requests: number;
}

export const adminService = {
  // Users
  listUsers: async (token: string, limit = 100): Promise<User[]> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}?limit=${limit}`, {
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
    limit = 50
  ): Promise<SearchHistoryItem[]> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.SEARCH_HISTORY}?limit=${limit}`,
      {
        method: "GET",
        token,
      }
    );
  },

  getUserSearchHistory: async (
    userId: string,
    token: string,
    limit = 50
  ): Promise<SearchHistoryItem[]> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}/search-history?limit=${limit}`,
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

  // Request Counts
  getRequestCounts: async (token: string): Promise<RequestCounts> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.REQUEST_COUNTS, {
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
};
