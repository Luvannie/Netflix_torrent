import type { ApiClient } from "./runtime";
import type { CreateSearchJobRequest, Page, SearchJob } from "./types";

export function createSearchJobsApi(client: ApiClient) {
  return {
    create(input: CreateSearchJobRequest) {
      return client.post<number>("/api/v1/search/jobs", input);
    },
    list(page = 0, size = 20, query = "") {
      const params = new URLSearchParams({
        page: String(page),
        size: String(size),
      });
      if (query) {
        params.set("query", query);
      }
      return client.get<Page<SearchJob>>(`/api/v1/search/jobs?${params.toString()}`);
    },
    get(id: number) {
      return client.get<SearchJob>(`/api/v1/search/jobs/${id}`);
    },
    process(id: number) {
      return client.post<null>(`/api/v1/search/jobs/${id}/process`);
    },
    cancel(id: number) {
      return client.delete<null>(`/api/v1/search/jobs/${id}`);
    },
  };
}

