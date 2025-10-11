import { apiRequest } from "@/lib/api-client";

export interface PasswordChangeRequest {
  id: string;
  user_id: string;
  reason: string;
  status: "pending" | "approved" | "rejected";
  admin_notes?: string;
  created_at: string;
  updated_at: string;
}

export interface PasswordChangeRequestWithUser extends PasswordChangeRequest {
  user_email: string;
  user_name: string;
}

export const passwordService = {
  // User endpoints
  requestPasswordChange: async (reason: string, token: string): Promise<PasswordChangeRequest> => {
    return apiRequest("/api/user/password-change/request", {
      method: "POST",
      body: JSON.stringify({ reason }),
      token,
    });
  },

  getUserPasswordRequests: async (token: string): Promise<PasswordChangeRequest[]> => {
    return apiRequest("/api/user/password-change/requests", {
      method: "GET",
      token,
    });
  },

  // Admin endpoints
  getPasswordChangeRequests: async (token: string, status: string = "pending"): Promise<PasswordChangeRequestWithUser[]> => {
    return apiRequest(`/api/admin/password-change-requests?status=${status}&limit=100`, {
      method: "GET",
      token,
    });
  },

  approvePasswordRequest: async (
    requestId: string,
    newPassword: string,
    adminNotes: string | undefined,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(`/api/admin/password-change-requests/${requestId}/approve`, {
      method: "POST",
      body: JSON.stringify({ new_password: newPassword, admin_notes: adminNotes }),
      token,
    });
  },

  rejectPasswordRequest: async (
    requestId: string,
    adminNotes: string,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(`/api/admin/password-change-requests/${requestId}/reject`, {
      method: "POST",
      body: JSON.stringify({ admin_notes: adminNotes }),
      token,
    });
  },

  changeUserPassword: async (
    userId: string,
    newPassword: string,
    token: string
  ): Promise<{ message: string }> => {
    return apiRequest(`/api/admin/users/${userId}/change-password`, {
      method: "POST",
      body: JSON.stringify({ new_password: newPassword }),
      token,
    });
  },
};

