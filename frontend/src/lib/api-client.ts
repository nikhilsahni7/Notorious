import { getApiUrl } from "@/config/api";

interface ApiOptions extends RequestInit {
  token?: string;
}

export class ApiError extends Error {
  constructor(message: string, public status: number, public data?: { code?: string; error?: string }) {
    super(message);
    this.name = "ApiError";
  }
}

export async function apiRequest<T = unknown>(
  endpoint: string,
  options: ApiOptions = {}
): Promise<T> {
  const { token, ...fetchOptions } = options;

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...fetchOptions.headers,
  };

  // Keep Authorization header as fallback for compatibility
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(getApiUrl(endpoint), {
    ...fetchOptions,
    headers,
    // SECURITY: Include credentials for httpOnly cookies
    credentials: "include",
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));

    // SECURITY: Handle token expired - redirect to login
    if (response.status === 401) {
      const code = errorData.code;
      if (code === "TOKEN_EXPIRED" || code === "TOKEN_MISSING") {
        // Clear any stored data and redirect
        if (typeof window !== "undefined") {
          localStorage.removeItem("knotorious_token");
          localStorage.removeItem("knotorious_user");
          // Redirect to login with message
          window.location.href = "/login?session=expired";
        }
      }
    }

    throw new ApiError(
      errorData.error || `Request failed: ${response.statusText}`,
      response.status,
      errorData
    );
  }

  // Handle 204 No Content responses (no body to parse)
  if (response.status === 204) {
    return undefined as T;
  }

  // Check if response has content before parsing
  const contentLength = response.headers.get("content-length");
  if (contentLength === "0") {
    return undefined as T;
  }

  return response.json();
}
