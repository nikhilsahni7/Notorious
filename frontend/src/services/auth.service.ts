import { API_CONFIG } from "@/config/api";
import { apiRequest } from "@/lib/api-client";

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: {
    id: string;
    email: string;
    name: string;
    role: string;
    daily_search_limit: number;
    searches_used_today: number;
    is_active: boolean;
  };
}

export interface AccessRequestData {
  email: string;
  name: string;
  phone: string;
  requested_searches_per_day: number;
}

export const authService = {
  login: async (data: LoginRequest): Promise<LoginResponse> => {
    return apiRequest(API_CONFIG.ENDPOINTS.AUTH.LOGIN, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  requestAccess: async (
    data: AccessRequestData
  ): Promise<{ message: string }> => {
    return apiRequest(API_CONFIG.ENDPOINTS.AUTH.REQUEST_ACCESS, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },
};
