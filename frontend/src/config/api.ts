export const API_CONFIG = {
  BASE_URL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080",
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
    },
    ADMIN: {
      USERS: "/api/admin/users",
      USER_REQUESTS: "/api/admin/user-requests",
      SEARCH_HISTORY: "/api/admin/search-history",
    },
  },
} as const;

export const getApiUrl = (endpoint: string): string => {
  return `${API_CONFIG.BASE_URL}${endpoint}`;
};
