import type { ApiClient } from "./runtime";
import type { MediaFile, MediaItem, Page } from "./types";

export function createLibraryApi(client: ApiClient) {
  return {
    list(page = 0, size = 20, type?: string) {
      const params = new URLSearchParams({
        page: String(page),
        size: String(size),
      });
      if (type) {
        params.set("type", type);
      }
      return client.get<Page<MediaItem>>(`/api/v1/library?${params.toString()}`);
    },
    get(id: number) {
      return client.get<MediaItem>(`/api/v1/library/${id}`);
    },
    delete(id: number) {
      return client.delete<null>(`/api/v1/library/${id}`);
    },
    streamInfo(id: number) {
      return client.get<MediaFile>(`/api/v1/streams/${id}/info`);
    },
    streamUrl(id: number) {
      return `/api/v1/streams/${id}`;
    },
  };
}

