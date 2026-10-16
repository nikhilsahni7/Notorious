export const API_CONFIG = {
  BASE_URL: (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080").replace(/^"|"$/g, ""),
  ENDPOINTS: {
    AUTH: {
      LOGIN: "/auth/login",
      REQUEST_ACCESS: "/auth/request-access",
    },
    SEARCH: {
      BASE: "/search",
      SUGGEST: "/search/suggest",
    },
    USER: {
      PROFILE: "/api/user/profile",
      SEARCH_HISTORY: "/api/user/search-history",
      METADATA: "/api/user/metadata",
      HEARTBEAT: "/api/user/heartbeat",
    },
    ADMIN: {
      USERS: "/api/admin/users",
      BULK_UPDATE_USERS: "/api/admin/users/bulk-update",
      ONLINE_USERS: "/api/admin/users/online",
      USER_REQUESTS: "/api/admin/user-requests",
      SEARCH_HISTORY: "/api/admin/search-history",
      PASSWORD_CHANGE_REQUESTS: "/api/admin/password-change-requests",
      SESSIONS: "/api/admin/sessions",
      USER_SESSIONS: "/api/admin/user-sessions",
      DASHBOARD_STATS: "/api/admin/dashboard-stats",
      USER_STATS: "/api/admin/stats/user",
      SYSTEM_STATS: "/api/admin/stats/system",
    },
  },
} as const;

export const getApiUrl = (endpoint: string): string => {
  return `${API_CONFIG.BASE_URL}${endpoint}`;
};
