# GUI Architecture

Ngay cap nhat: 2026-04-20

Tai lieu nay mo ta GUI cho huong desktop launcher:

```text
NetflixTorrent.exe
  -> native launcher shell
  -> WebView UI
  -> Java Spring Boot backend
  -> sidecar services
```

## 1. Ket luan stack GUI

Khuyen nghi cho ban Windows dau tien:

```text
.NET 8 WPF launcher + WebView2 + React TypeScript Vite UI
```

Ly do:

- Backend Java khong yeu cau GUI cung Java. Backend giao tiep qua HTTP REST va STOMP WebSocket tren `127.0.0.1`.
- .NET/WPF quan ly process Windows tot: start/stop Java, qBittorrent, PostgreSQL, Prowlarr/Jackett.
- WebView2 render UI web hien dai, du de dung React/TanStack Query/video player.
- .NET co DPAPI de bao ve local secrets nhu local token, DB password, qBittorrent password.
- WebView2 co API inject script/token vao page ma khong can ghi secret vao bundled JS.
- Windows-first phu hop muc tieu `.exe`.

Khong chon JavaFX cho phase dau:

- Tuong thich voi Java backend la tot, nhung launcher/process supervision, Windows credential storage, WebView hien dai, va installer Windows thuong lam bang .NET de hon.

Neu sau nay can cross-platform:

```text
Tauri + React
```

Nhung phase dau nen chon `.NET WPF + WebView2` de giam rui ro desktop packaging tren Windows.

## 2. Compatibility with Java backend

GUI va backend chi can contract network:

```text
GUI WebView
  -> REST HTTP http://127.0.0.1:<backendPort>/api/v1/...
  -> STOMP WebSocket ws://127.0.0.1:<backendPort>/ws
  -> stream video http://127.0.0.1:<backendPort>/api/v1/streams/{mediaFileId}
```

Do do backend co the chay Java/Spring Boot, launcher co the chay .NET, UI co the chay React. Chung khong can cung runtime.

Backend desktop profile target:

```text
SPRING_PROFILES_ACTIVE=worker,desktop
```

GUI must add header for write APIs:

```http
X-App-Local-Token: <launcher-generated-token>
```

## 3. Process ownership

`NetflixTorrent.exe` la native process dau tien.

Responsibilities:

- single instance lock
- first-run wizard host
- allocate ports
- generate and persist local secrets
- start/stop sidecars
- start Java backend with bundled JRE
- wait backend readiness
- open WebView2 window
- inject runtime config into UI
- collect logs
- graceful shutdown

UI React khong tu start process. React chi goi API va hien state.

## 4. Recommended project layout

Target layout:

```text
desktop-launcher/
  NetflixTorrent.Launcher/
    App.xaml
    MainWindow.xaml
    Services/
      ProcessSupervisor.cs
      PortAllocator.cs
      SecretStore.cs
      BackendSupervisor.cs
      SidecarSupervisor.cs
      FirstRunService.cs
      AppConfigWriter.cs
    WebView/
      WebViewBridge.cs
    Packaging/
      app.manifest

Frontend/
  src/
    app/
      App.tsx
      router.tsx
      queryClient.ts
    features/
      setup/
      catalog/
      search/
      downloads/
      library/
      player/
      settings/
      diagnostics/
    shared/
      api/
      websocket/
      components/
      hooks/
      types/
      utils/
```

Build output:

```text
Frontend/dist -> packaged into launcher resources or copied beside launcher
Backend/target/backend-*.jar -> packaged into app/backend/backend.jar
Bundled JRE -> app/runtime/jre
ffmpeg -> app/tools/ffmpeg
sidecars -> app/sidecars
```

## 5. UI loading strategy

Preferred phase-dau strategy:

```text
Launcher starts backend.
Backend serves React static assets.
WebView2 navigates to http://127.0.0.1:<backendPort>/
```

Pros:

- Same origin as API and WebSocket.
- CORS complexity low.
- Stream/video URLs are same host.
- Easier to debug with browser.

Requirement:

- Backend must be configured later to serve static frontend assets from `classpath:/static` or external `ui/` directory.

Alternative:

```text
WebView2 loads local file/app assets.
React calls backend API on 127.0.0.1.
```

Use this only if backend static serving becomes inconvenient. It requires stricter CORS handling.

## 6. Runtime config injection

Launcher injects config before page code runs:

```javascript
window.__NETFLIX_TORRENT__ = {
  apiBaseUrl: "http://127.0.0.1:18080",
  wsUrl: "ws://127.0.0.1:18080/ws",
  localToken: "...",
  appVersion: "0.1.0",
  mode: "desktop"
}
```

React reads this once at startup and configures:

- API client base URL
- WebSocket client URL
- local token header
- feature flags

Do not bake local token into bundled JS.

## 7. Navigation map

Initial GUI pages:

| Route | Screen | Purpose |
|---|---|---|
| `/setup` | First-run wizard | media folder, TMDB key, provider setup |
| `/` | Home/catalog | search movies and show recommendations later |
| `/catalog/search` | Catalog search | TMDB title search |
| `/catalog/discover` | Discover | filters: genre, actor, director, year |
| `/movie/:tmdbId` | Movie detail | metadata, start torrent source search |
| `/search/jobs/:jobId` | Source results | release list and quality info |
| `/downloads` | Downloads list | all tasks |
| `/downloads/:taskId` | Download detail | progress and errors |
| `/library` | Library | completed media |
| `/library/:mediaItemId` | Media detail | files and play action |
| `/player/:mediaFileId` | Player | HTML5 video from stream endpoint |
| `/settings` | Settings | storage, provider, qBittorrent, diagnostics |
| `/diagnostics` | Diagnostics | health checks, logs, export bug report |

## 8. Feature modules

### setup

Owns:

- first-run status
- media folder selection
- TMDB key entry
- provider selection: Prowlarr or Jackett
- sidecar health validation
- storage profile creation

Desktop launcher native bridge may be needed for:

- choose folder dialog
- open sidecar web UI
- export logs
- restart sidecar

### catalog

Owns:

- movie search
- genres
- discover filters
- movie detail view

Backend APIs:

```text
GET /api/v1/catalog/search?query=
GET /api/v1/catalog/genres
GET /api/v1/catalog/discover
GET /api/v1/catalog/movies/{tmdbId}
```

### search

Owns:

- create torrent search job
- poll job
- subscribe to `/topic/search/jobs/{jobId}`
- show releases
- select release for download

Backend APIs:

```text
POST /api/v1/search/jobs
GET /api/v1/search/jobs/{jobId}
```

### downloads

Owns:

- create download task
- display task progress
- cancel task
- subscribe to `/topic/downloads/{taskId}`

Backend APIs:

```text
POST /api/v1/downloads
GET /api/v1/downloads
GET /api/v1/downloads/{taskId}
POST /api/v1/downloads/{taskId}/cancel
```

### library/player

Owns:

- list completed media
- media detail
- playback
- subscribe to `/topic/media/ready`

Backend APIs:

```text
GET /api/v1/library
GET /api/v1/library/{mediaItemId}
GET /api/v1/streams/{mediaFileId}
```

Player:

```html
<video src="http://127.0.0.1:<port>/api/v1/streams/{mediaFileId}" controls />
```

The browser/WebView handles Range requests for seek.

### settings/diagnostics

Owns:

- storage profiles
- provider status
- qBittorrent status
- app paths
- log export
- restart sidecar actions

Backend APIs currently available:

```text
GET/POST/PUT/DELETE /api/v1/settings/storage-profiles
GET /api/v1/health
GET /actuator/health/readiness
GET /api/v1/system/status
```

Additional backend/launcher bridge APIs are needed for full diagnostics.

## 9. API client rules

Use a single API client wrapper:

```text
shared/api/httpClient.ts
```

Responsibilities:

- base URL from `window.__NETFLIX_TORRENT__`
- include `X-App-Local-Token` for write requests
- parse `ApiResponse<T>`
- normalize error response
- add `X-Request-Id` if useful

Write methods:

```text
POST, PUT, PATCH, DELETE
```

Must include:

```http
X-App-Local-Token
```

Do not let feature components call `fetch` directly.

## 10. State management

Use:

```text
TanStack Query for server state
React local state for forms/view state
Small app store only for runtime config and UI shell state
```

Query keys:

```text
["catalog", "search", query]
["catalog", "genres"]
["catalog", "discover", filters]
["searchJob", jobId]
["downloads"]
["downloadTask", taskId]
["library"]
["mediaItem", mediaItemId]
["health"]
```

WebSocket event handlers should invalidate or patch TanStack Query cache. They should not become the only source of truth.

Examples:

- `download.progress` patch `["downloadTask", taskId]`.
- `download.completed` invalidate `["library"]` and `["downloadTask", taskId]`.
- `search.completed` invalidate `["searchJob", jobId]`.

## 11. WebSocket client rules

Use STOMP over native WebSocket.

Desktop auth disabled:

- no JWT required; write requests use `X-App-Local-Token` generated by the launcher
- local token is not currently validated by WebSocket backend
- connect to `/ws`

Production auth enabled:

- send `Authorization: Bearer <token>` in STOMP CONNECT headers

Reconnect:

- exponential backoff
- after reconnect, refetch active jobs/downloads
- never assume missed events were delivered

## 12. Desktop bridge contract

React can call native launcher through WebView2 host object for OS-level actions.

Proposed bridge methods:

```text
chooseMediaFolder(): Promise<string>
openExternalUrl(url): Promise<void>
openSidecarUi(service): Promise<void>
restartSidecar(service): Promise<void>
getRuntimeInfo(): Promise<RuntimeInfo>
exportLogs(): Promise<string>
shutdownApp(): Promise<void>
```

React must not directly manipulate filesystem paths except through backend or native bridge.

## 13. First-run wizard flow

```text
App starts
  -> launcher loads config
  -> if config incomplete, open /setup
  -> user selects media folder
  -> user enters TMDB key
  -> user chooses Prowlarr or Jackett
  -> launcher starts provider sidecar
  -> user configures/test indexer
  -> launcher writes provider API key/config
  -> backend health validates dependencies
  -> setup complete
  -> route to catalog
```

Setup completion conditions:

- backend health ready
- media folder exists and writable
- TMDB key accepted or at least saved for later validation
- qBittorrent sidecar reachable
- provider sidecar reachable
- storage profile exists

## 14. Visual and UX rules

Follow product-first UI:

- First screen after setup should be usable catalog/search, not marketing.
- Downloads must show clear state and error reason.
- Search results must show title, size, seeders, leechers, provider/indexer, score/hash availability.
- If result lacks hash/btih, disable download and explain reason.
- Settings should expose health and logs because sidecars are failure-prone.
- Streaming screen should handle 404/not-ready with a recoverable message and link back to download/library detail.

## 15. GUI data types

Frontend should mirror backend DTOs.

Core types:

```ts
type ApiResponse<T> = {
  data: T
  meta: {
    timestamp: string
    requestId?: string
  }
}

type SearchJobStatus =
  | "REQUESTED"
  | "SEARCHING"
  | "SEARCH_READY"
  | "FAILED"
  | "CANCELLED"

type DownloadTaskStatus =
  | "REQUESTED"
  | "SEARCHING"
  | "SEARCH_READY"
  | "QUEUED"
  | "DOWNLOADING"
  | "POST_PROCESSING"
  | "STREAM_READY"
  | "COMPLETED"
  | "FAILED"
  | "CANCELLED"
```

Do not invent frontend-only status names unless mapped in one place.

## 16. Error handling

Global UI error categories:

| Category | Example | UI action |
|---|---|---|
| Backend unavailable | Java process not ready | show startup screen, retry |
| Sidecar unavailable | qBittorrent down | show diagnostics action |
| Provider auth missing | Jackett/Prowlarr API key missing | route to setup/settings |
| Search failed | provider timeout | allow retry |
| Download failed | no hash, qBittorrent error | show reason and allow new release |
| Stream not ready | codec unsupported | show reason, future transcode action |

Every long-running action should show:

- current state
- latest error reason
- retry/cancel action where valid

## 17. Launcher packaging contract

Installer/package must include:

```text
NetflixTorrent.exe
backend/backend.jar
runtime/jre/**
tools/ffmpeg/ffprobe.exe
tools/ffmpeg/ffmpeg.exe
sidecars/postgres/**
sidecars/qbittorrent/**
sidecars/prowlarr-or-jackett/**
ui/**
```

User data must be outside install dir:

```text
%LOCALAPPDATA%\NetflixTorrent\
  config\
  data\
  media\
  logs\
  sidecars\
```

Launcher must never write secrets into the frontend `dist` folder.

## 18. Required backend additions for GUI completeness

GUI can start with current APIs. Backend now exposes:

```text
GET /api/v1/system/status
```

A polished desktop still needs these backend/launcher additions:

- `GET /api/v1/settings/integrations` and update endpoints for TMDB/provider/qBittorrent.
- first-run setup completion endpoint or launcher bridge state.
- storage profile default creation from selected folder.
- provider test endpoint.
- qBittorrent test endpoint.
- log export through launcher bridge.
- static UI serving or agreed local asset strategy.

## 19. GUI coding checklist

Before adding a GUI feature:

- [ ] Backend endpoint exists or bridge method is defined.
- [ ] DTO type mirrors backend response.
- [ ] Write request includes local token.
- [ ] Query key and invalidation are defined.
- [ ] Loading, empty, error, and success states are implemented.
- [ ] Long-running job has polling fallback.
- [ ] WebSocket event only patches/refetches server state.
- [ ] No secrets are stored in localStorage unless explicitly approved.
- [ ] Logs/diagnostics path is available for sidecar-related failures.
