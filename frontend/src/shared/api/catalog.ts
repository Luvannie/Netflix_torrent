import type { ApiClient } from "./runtime";
import type { Genre, Movie, MovieSummary, Page, TMDBMovieDetail } from "./types";

export type DiscoverCatalogInput = {
  genreId?: number;
  actor?: string;
  director?: string;
  year?: string;
  page?: number;
};

export function createCatalogApi(client: ApiClient) {
  return {
    list(page = 0, size = 20) {
      return client.get<Page<MovieSummary>>(`/api/v1/catalog?page=${page}&size=${size}`);
    },
    getById(id: number) {
      return client.get<Movie>(`/api/v1/catalog/${id}`);
    },
    getMovieByTmdbId(tmdbId: number) {
      return client.get<Movie>(`/api/v1/catalog/movies/${tmdbId}`);
    },
    search(query: string) {
      return client.get<TMDBMovieDetail[]>(`/api/v1/catalog/search?query=${encodeURIComponent(query)}`);
    },
    genres() {
      return client.get<Genre[]>("/api/v1/catalog/genres");
    },
    discover(input: DiscoverCatalogInput) {
      const params = new URLSearchParams();
      if (input.genreId) params.set("genreId", String(input.genreId));
      if (input.actor) params.set("actor", input.actor);
      if (input.director) params.set("director", input.director);
      if (input.year) params.set("year", input.year);
      if (input.page !== undefined) params.set("page", String(input.page));
      return client.get<TMDBMovieDetail[]>(`/api/v1/catalog/discover?${params.toString()}`);
    },
  };
}

