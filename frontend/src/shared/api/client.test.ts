import { describe, expect, it, vi } from "vitest";
import { createApiClient } from "./client";
import { ApiClientError } from "./errors";

describe("createApiClient", () => {
  it("unwraps backend ApiResponse<T>", async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () =>
        JSON.stringify({
          data: { status: "UP", service: "backend" },
          meta: { timestamp: "2026-04-22T00:00:00Z" },
        }),
    });

    const client = createApiClient({ fetcher: fetcher as never });
    const result = await client.get<{ status: string; service: string }>("/api/v1/health");

    expect(result.status).toBe("UP");
    expect(fetcher).toHaveBeenCalledWith(
      "/api/v1/health",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("throws ApiClientError for backend error envelopes", async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      text: async () =>
        JSON.stringify({
          error: { code: "TOKEN_MISSING", message: "Local token required", details: [] },
          meta: { timestamp: "2026-04-22T00:00:00Z", requestId: "req-1" },
        }),
    });

    const client = createApiClient({ fetcher: fetcher as never });

    await expect(client.post("/api/v1/downloads", {})).rejects.toBeInstanceOf(ApiClientError);
  });
});

