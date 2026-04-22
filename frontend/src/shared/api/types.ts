export type ApiMeta = {
  timestamp: string;
  requestId?: string;
};

export type ApiResponse<T> = {
  data: T;
  meta: ApiMeta;
};

export type ApiErrorDetail = {
  field: string;
  message: string;
};

export type ApiErrorResponse = {
  error: {
    code: string;
    message: string;
    details: ApiErrorDetail[];
  };
  meta: ApiMeta;
};

export type Page<T> = {
  content: T[];
  number: number;
  size: number;
  totalElements: number;
  totalPages: number;
  first: boolean;
  last: boolean;
  numberOfElements: number;
  empty: boolean;
};

export type HealthStatus = {
  status: string;
  service: string;
};

export type ComponentStatus = {
  status: "UP" | "DOWN" | "DISABLED";
  message: string;
  details: Record<string, unknown>;
};

export type SystemStatusResponse = {
  overallStatus: "UP" | "DOWN";
  mode: string;
  activeProfiles: string[];
  components: Record<string, ComponentStatus>;
  checkedAt: string;
};

export type Movie = {
  id: number;
  tmdbId: number;
  title: string;
  overview: string;
  posterPath: string;
  backdropPath: string;
  releaseDate: string;
  voteAverage: number;
  voteCount: number;
  popularity: number;
  originalLanguage: string;
  originalTitle: string;
  catalogAddedAt: string;
};

export type MovieSummary = {
  id: number;
  tmdbId: number;
  title: string;
  posterPath: string;
  releaseDate: string;
  voteAverage: number;
  popularity: number;
  catalogAddedAt: string;
};

export type Genre = {
  id: number;
  name: string;
};

export type TMDBMovieDetail = {
  id: number;
  title: string;
  overview: string;
  poster_path: string;
  backdrop_path: string;
  release_date: string;
  vote_average: number;
  vote_count: number;
  popularity: number;
  original_language: string;
  original_title: string;
  genres: Genre[];
};

export type SearchJobStatus =
  | "REQUESTED"
  | "SEARCHING"
  | "SEARCH_READY"
  | "FAILED"
  | "CANCELLED";

export type SearchResult = {
  id: number;
  searchJobId: number;
  guid: string;
  title: string;
  link: string;
  permalink: string;
  size: number;
  pubDate?: string;
  seeders: number;
  leechers: number;
  indexer: string;
  provider: string;
  hash: string;
  score: number;
  createdAt: string;
};

export type SearchJob = {
  id: number;
  query: string;
  status: SearchJobStatus;
  createdAt: string;
  updatedAt: string;
  errorMessage?: string;
  results?: SearchResult[];
};

export type CreateSearchJobRequest = {
  query: string;
};

export type DownloadTaskStatus =
  | "REQUESTED"
  | "SEARCHING"
  | "SEARCH_READY"
  | "QUEUED"
  | "DOWNLOADING"
  | "POST_PROCESSING"
  | "STREAM_READY"
  | "COMPLETED"
  | "FAILED"
  | "CANCELLED";

export type DownloadTask = {
  id: number;
  searchResultId: number;
  torrentHash: string;
  status: DownloadTaskStatus;
  progress: number;
  speed: number;
  peerCount: number;
  createdAt: string;
  updatedAt: string;
};

export type CreateDownloadTaskRequest = {
  searchResultId: number;
};

export type MediaFile = {
  id: number;
  mediaItemId: number;
  filePath: string;
  container?: string;
  codec?: string;
  duration?: number;
  width?: number;
  height?: number;
  size?: number;
};

export type MediaItem = {
  id: number;
  tmdbId?: number;
  title: string;
  year?: number;
  type: string;
  files?: MediaFile[];
  createdAt: string;
};

export type StorageProfile = {
  id: number;
  name: string;
  basePath: string;
  priority: number;
  active: boolean;
};

export type CreateStorageProfileRequest = {
  name: string;
  basePath: string;
  priority?: number;
  active?: boolean;
};

export type UpdateStorageProfileRequest = {
  name?: string;
  basePath?: string;
  priority?: number;
  active?: boolean;
};

