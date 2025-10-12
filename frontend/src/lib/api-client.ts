import { getApiUrl } from "@/config/api";

interface ApiOptions extends RequestInit {
  token?: string;
}

export class ApiError extends Error {
  constructor(message: string, public status: number, public data?: unknown) {
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

  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(getApiUrl(endpoint), {
    ...fetchOptions,
    headers,
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
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
