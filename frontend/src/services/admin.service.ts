import { API_CONFIG } from "@/config/api";
import { apiRequest } from "@/lib/api-client";

export interface User {
  id: string;
  email: string;
  name: string;
  phone: string;
  role: string;
  daily_search_limit: number;
  searches_used_today: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserRequest {
  id: string;
  email: string;
  name: string;
  phone: string;
  requested_searches_per_day: number;
  status: string;
  created_at: string;
  admin_notes: string;
}

export interface SearchHistoryItem {
  id: string;
  user_id: string;
  user_email?: string;
  user_name?: string;
  query: string;
  total_results: number;
  top_results: Record<string, unknown>[];
  searched_at: string;
}

export interface CreateUserData {
  email: string;
  password: string;
  name: string;
  phone: string;
  daily_search_limit: number;
  is_active: boolean;
}

export interface UpdateUserData {
  name: string;
  phone: string;
  daily_search_limit: number;
  is_active: boolean;
}

export const adminService = {
  // User Management
  listUsers: async (
    token: string,
    params?: { limit?: number; offset?: number; role?: string }
  ): Promise<User[]> => {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());
    if (params?.role) queryParams.append("role", params.role);

    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USERS}?${queryParams.toString()}`,
      { method: "GET", token }
    );
  },

  getUser: async (token: string, userId: string): Promise<User> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}`, {
      method: "GET",
      token,
    });
  },

  createUser: async (token: string, data: CreateUserData): Promise<User> => {
    return apiRequest(API_CONFIG.ENDPOINTS.ADMIN.USERS, {
      method: "POST",
      body: JSON.stringify(data),
      token,
    });
  },

  updateUser: async (
    token: string,
    userId: string,
    data: UpdateUserData
  ): Promise<User> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}`, {
      method: "PUT",
      body: JSON.stringify(data),
      token,
    });
  },

  deleteUser: async (token: string, userId: string): Promise<void> => {
    return apiRequest(`${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}`, {
      method: "DELETE",
      token,
    });
  },

  // User Requests
  listUserRequests: async (
    token: string,
    params?: { limit?: number; offset?: number; status?: string }
  ): Promise<UserRequest[]> => {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());
    if (params?.status) queryParams.append("status", params.status);

    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USER_REQUESTS}?${queryParams.toString()}`,
      { method: "GET", token }
    );
  },

  approveUserRequest: async (
    token: string,
    requestId: string,
    data: { password: string; daily_search_limit: number }
  ): Promise<User> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USER_REQUESTS}/${requestId}/approve`,
      {
        method: "POST",
        body: JSON.stringify(data),
        token,
      }
    );
  },

  rejectUserRequest: async (
    token: string,
    requestId: string,
    reason?: string
  ): Promise<void> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USER_REQUESTS}/${requestId}/reject`,
      {
        method: "POST",
        body: JSON.stringify({ reason: reason || "" }),
        token,
      }
    );
  },

  // Search History
  getSearchHistory: async (
    token: string,
    params?: { limit?: number; offset?: number }
  ): Promise<SearchHistoryItem[]> => {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());

    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.SEARCH_HISTORY}?${queryParams.toString()}`,
      { method: "GET", token }
    );
  },

  getUserSearchHistory: async (
    token: string,
    userId: string,
    params?: { limit?: number; offset?: number }
  ): Promise<SearchHistoryItem[]> => {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());

    return apiRequest(
      `${API_CONFIG.ENDPOINTS.ADMIN.USERS}/${userId}/search-history?${queryParams.toString()}`,
      { method: "GET", token }
    );
  },
};
