import type { ApiClient } from "./runtime";
import type {
  CreateStorageProfileRequest,
  StorageProfile,
  UpdateStorageProfileRequest,
} from "./types";

export function createSettingsApi(client: ApiClient) {
  return {
    listStorageProfiles() {
      return client.get<StorageProfile[]>("/api/v1/settings/storage-profiles");
    },
    getStorageProfile(id: number) {
      return client.get<StorageProfile>(`/api/v1/settings/storage-profiles/${id}`);
    },
    createStorageProfile(input: CreateStorageProfileRequest) {
      return client.post<StorageProfile>("/api/v1/settings/storage-profiles", input);
    },
    updateStorageProfile(id: number, input: UpdateStorageProfileRequest) {
      return client.put<StorageProfile>(`/api/v1/settings/storage-profiles/${id}`, input);
    },
    deleteStorageProfile(id: number) {
      return client.delete<null>(`/api/v1/settings/storage-profiles/${id}`);
    },
  };
}

