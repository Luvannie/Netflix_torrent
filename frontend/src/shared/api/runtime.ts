import { createApiClient } from "./client";
import type { HealthStatus, SystemStatusResponse } from "./types";

export const apiClient = createApiClient();

export type ApiClient = ReturnType<typeof createApiClient>;

export function createRuntimeApi(client: ApiClient) {
  return {
    getHealth() {
      return client.get<HealthStatus>("/api/v1/health");
    },
    getSystemStatus() {
      return client.get<SystemStatusResponse>("/api/v1/system/status");
    },
  };
}

