import { ApiError } from "@/lib/api-client";
import { SearchResponse, searchService } from "@/services/search.service";
import { SearchFields, SearchOperator } from "@/types/person";
import { useState } from "react";

export function useSearch(token: string | null) {
  const [loading, setLoading] = useState(false);

  const performSearch = async (
    searchFields: SearchFields,
    operator: SearchOperator,
    page: number,
    pageSize: number
  ): Promise<SearchResponse> => {
    if (!token) {
      throw new Error("Authentication required");
    }

    setLoading(true);

    try {
      const queries: string[] = [];
      Object.entries(searchFields).forEach(([field, value]) => {
        if (value.trim()) {
          queries.push(`${field}:${value.trim()}`);
        }
      });

      if (queries.length === 0) {
        throw new Error("Please enter at least one search term");
      }

      const queryString = queries.join(` ${operator} `);
      const from = (page - 1) * pageSize;

      const data = await searchService.search(
        {
          query: queryString,
          fields: [
            "id",
            "oid",
            "name",
            "fname",
            "mobile",
            "alt",
            "email",
            "address",
          ],
          and_or: operator,
          size: pageSize,
          from: from,
        },
        token
      );

      return data;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        throw new Error("Session expired. Please login again.");
      }
      throw error;
    } finally {
      setLoading(false);
    }
  };

  return { performSearch, loading };
}
