import { ApiClientError } from "./errors";
import type { ApiErrorResponse, ApiResponse } from "./types";

type Fetcher = typeof fetch;

function isJsonResponse(value: unknown): value is ApiResponse<unknown> | ApiErrorResponse {
  return typeof value === "object" && value !== null;
}

export function createApiClient(options?: { fetcher?: Fetcher }) {
  const fetcher = options?.fetcher ?? fetch;

  async function request<T>(path: string, init: RequestInit): Promise<T> {
    const response = await fetcher(path, {
      headers: {
        "Content-Type": "application/json",
        ...(init.headers ?? {}),
      },
      ...init,
    });

    const text = await response.text();
    const json = text ? (JSON.parse(text) as unknown) : null;

    if (!response.ok) {
      if (isJsonResponse(json)) {
        throw ApiClientError.fromResponse(json as ApiErrorResponse, response.status);
      }
      throw new ApiClientError({
        status: response.status,
        code: "HTTP_ERROR",
        message: `Request failed with HTTP ${response.status}`,
      });
    }

    if (json === null) {
      return null as T;
    }

    return (json as ApiResponse<T>).data;
  }

  return {
    get<T>(path: string) {
      return request<T>(path, { method: "GET" });
    },
    post<T>(path: string, data?: unknown) {
      return request<T>(path, {
        method: "POST",
        body: data === undefined ? undefined : JSON.stringify(data),
      });
    },
    put<T>(path: string, data: unknown) {
      return request<T>(path, {
        method: "PUT",
        body: JSON.stringify(data),
      });
    },
    delete<T>(path: string) {
      return request<T>(path, { method: "DELETE" });
    },
  };
}

