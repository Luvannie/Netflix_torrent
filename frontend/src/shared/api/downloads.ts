import type { ApiClient } from "./runtime";
import type { CreateDownloadTaskRequest, DownloadTask, Page } from "./types";

export function createDownloadsApi(client: ApiClient) {
  return {
    create(input: CreateDownloadTaskRequest) {
      return client.post<DownloadTask>("/api/v1/downloads", input);
    },
    list(page = 0, size = 20) {
      return client.get<Page<DownloadTask>>(`/api/v1/downloads?page=${page}&size=${size}`);
    },
    get(id: number) {
      return client.get<DownloadTask>(`/api/v1/downloads/${id}`);
    },
    cancel(id: number) {
      return client.post<DownloadTask>(`/api/v1/downloads/${id}/cancel`);
    },
  };
}

