export const queryKeys = {
  health: () => ["health"] as const,
  systemStatus: () => ["systemStatus"] as const,
  catalogList: (page: number, size: number) => ["catalog", "list", page, size] as const,
  catalogSearch: (query: string) => ["catalog", "search", query] as const,
  catalogGenres: () => ["catalog", "genres"] as const,
  catalogDiscover: (filters: Record<string, unknown>) => ["catalog", "discover", filters] as const,
  catalogMovie: (tmdbId: number) => ["catalog", "movie", tmdbId] as const,
  searchJobs: (page: number, size: number, query: string) =>
    ["searchJobs", "list", page, size, query] as const,
  searchJobDetail: (id: number) => ["searchJobs", "detail", id] as const,
  downloads: (page: number, size: number) => ["downloads", "list", page, size] as const,
  downloadDetail: (id: number) => ["downloads", "detail", id] as const,
  library: (page: number, size: number, type?: string) =>
    ["library", "list", page, size, type ?? "all"] as const,
  libraryDetail: (id: number) => ["library", "detail", id] as const,
  storageProfiles: () => ["storageProfiles"] as const,
  storageProfile: (id: number) => ["storageProfile", id] as const,
};

