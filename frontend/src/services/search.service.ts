import { API_CONFIG } from "@/config/api";
import { apiRequest } from "@/lib/api-client";
import { SearchOperator, Person } from "@/types/person";

export interface SearchRequest {
  query: string;
  fields: string[];
  and_or: SearchOperator;
  size: number;
  from: number;
}

export interface SearchResponse {
  total: number;
  results: Person[];
  took_ms: number;
  searches_used_today?: number;
  daily_search_limit?: number;
  searches_remaining?: number;
  is_duplicate?: boolean;
}

export interface SearchHistoryItem {
  id: string;
  user_id: string;
  query: string;
  total_results: number;
  top_results: Person[];
  searched_at: string;
}

export const searchService = {
  search: async (
    data: SearchRequest,
    token: string
  ): Promise<SearchResponse> => {
    return apiRequest(API_CONFIG.ENDPOINTS.SEARCH.BASE, {
      method: "POST",
      body: JSON.stringify(data),
      token,
    });
  },

  performSearch: async (
    data: SearchRequest,
    token: string
  ): Promise<SearchResponse> => {
    return apiRequest(API_CONFIG.ENDPOINTS.SEARCH.BASE, {
      method: "POST",
      body: JSON.stringify(data),
      token,
    });
  },

  suggest: async (query: string, token: string): Promise<SearchResponse> => {
    return apiRequest(
      `${API_CONFIG.ENDPOINTS.SEARCH.SUGGEST}?q=${encodeURIComponent(query)}`,
      {
        method: "GET",
        token,
      }
    );
  },

  getHistory: async (token: string): Promise<SearchHistoryItem[]> => {
    return apiRequest(API_CONFIG.ENDPOINTS.USER.SEARCH_HISTORY, {
      method: "GET",
      token,
    });
  },
};
