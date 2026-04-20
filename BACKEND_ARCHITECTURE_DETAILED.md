# Backend Architecture

Ngay cap nhat: 2026-04-20

Backend hien duoc thiet ke cho desktop launcher, khong phai server deployment.

```text
Desktop GUI
  -> HTTP/WebSocket 127.0.0.1
Backend jar, profiles worker,desktop
  -> PostgreSQL sidecar
  -> qBittorrent sidecar
  -> Prowlarr hoac Jackett sidecar
  -> ffprobe bundled
```

## 1. Runtime Profile

| Muc dich | Profile |
|---|---|
| Unit/integration test | test resources |
| Debug local qua IDE/Maven | `local` |
| Bat worker processors | `worker` |
| Desktop launcher target | `worker,desktop` |
| Local E2E Docker | `worker,desktop` trong container, override env qua `docker-compose.local.yml` |

`worker` chi la profile de kich hoat scheduled processors. Runtime config phai den tu `local` hoac `desktop`.

## 2. Runtime Dependencies

Backend can:

- Java 21 runtime, do launcher bundle.
- PostgreSQL, vi Flyway/JPA dang dua tren schema SQL.
- qBittorrent Web API de add torrent va poll progress.
- Prowlarr hoac Jackett de tim torrent source.
- `ffprobe` de inspect file video sau khi download.

Backend khong can:

- Redis.
- JWT/demo auth.
- Production Docker stack.
- API process va worker process tach rieng.

## 3. Main Modules

| Module | Package | Trach nhiem |
|---|---|---|
| catalog | `catalog`, `integration.tmdb` | Tim phim, map TMDB metadata |
| search | `search` | Tao search job, goi Jackett/Prowlarr, normalize, score, dedupe |
| downloads | `downloads` | Tao download task, add torrent vao qBittorrent, poll progress |
| library | `library` | Tao media item/file sau download, probe bang ffprobe |
| streaming | `streaming` | Range streaming file da san sang |
| settings | `settings` | Storage profile va path validation |
| system | `system` | Status endpoint cho GUI/launcher |
| common | `common` | API response, error response, local-only/local-token filters |
| config | `config` | Spring config, security, websocket, external clients |

## 4. Security Model

Desktop backend chi bind localhost theo mac dinh:

```yaml
server.address: 127.0.0.1
app.network.bind-localhost-only: true
```

Write endpoints co the yeu cau header:

```text
X-App-Local-Token: <launcher-generated-token>
```

Launcher phai sinh token ngau nhien va truyen cho backend bang:

```env
APP_LOCAL_TOKEN_ENABLED=true
APP_LOCAL_TOKEN=<random-token>
```

Protected write paths:

- `POST/PUT/PATCH/DELETE /api/v1/search/jobs/**`
- `POST/PUT/PATCH/DELETE /api/v1/downloads/**`
- `POST/PUT/PATCH/DELETE /api/v1/settings/**`
- `POST/PUT/PATCH/DELETE /api/v1/library/scan/**`

Read endpoints duoc phep goi local khong can token.

## 5. API Surface

| Method | Path | Muc dich |
|---|---|---|
| GET | `/api/v1/health` | Health don gian |
| GET | `/actuator/health/readiness` | Spring readiness |
| GET | `/api/v1/system/status` | Tong hop status cho launcher/GUI |
| GET | `/api/v1/catalog` | List catalog |
| GET | `/api/v1/catalog/{id}` | Movie detail |
| GET | `/api/v1/catalog/search` | Search TMDB |
| GET | `/api/v1/catalog/genres` | Genres |
| POST | `/api/v1/search/jobs` | Tao torrent search job |
| GET | `/api/v1/search/jobs/{id}` | Poll search job |
| POST | `/api/v1/downloads` | Tao download task tu search result |
| GET | `/api/v1/downloads` | List download tasks |
| GET | `/api/v1/downloads/{id}` | Poll download task |
| POST | `/api/v1/downloads/{id}/cancel` | Cancel download task |
| GET | `/api/v1/library` | List library |
| GET | `/api/v1/library/{id}` | Media item detail |
| GET | `/api/v1/streams/{mediaFileId}` | Stream video, ho tro Range |

WebSocket endpoint:

```text
/ws
```

Dung cho progress events khi GUI can realtime.

## 6. Search Workflow

```text
GUI -> POST /api/v1/search/jobs
Backend -> persist SearchJob REQUESTED
SearchJobProcessor -> SEARCHING
SearchJobProcessor -> Jackett/Prowlarr
SearchJobProcessor -> normalize + score + dedupe
SearchJobProcessor -> persist SearchResult
SearchJobProcessor -> SEARCH_READY hoac FAILED
GUI -> GET /api/v1/search/jobs/{id}
```

Search result can du cac field de download worker hoat dong:

- `magnetUri` hoac URL co `btih`.
- `hash` neu provider tra ve.
- `title`, `sizeBytes`, `seeders`, `leechers`.
- `source`, `indexer`, `infoUrl`, `publishDate`.

## 7. Download Workflow

```text
GUI -> POST /api/v1/downloads { searchResultId }
Backend -> persist DownloadTask REQUESTED
DownloadTaskProcessor -> QUEUED
DownloadTaskProcessor -> qBittorrent add torrent
DownloadTaskProcessor -> DOWNLOADING, poll qBittorrent
DownloadTaskProcessor -> POST_PROCESSING
DownloadTaskProcessor -> ffprobe media file
DownloadTaskProcessor -> create MediaItem + MediaFile
DownloadTaskProcessor -> STREAM_READY / COMPLETED
GUI -> stream /api/v1/streams/{mediaFileId}
```

Dieu kien quan trong: qBittorrent save path va backend `DOWNLOAD_DEFAULT_SAVE_PATH` phai tro den cung mot filesystem path ma backend doc duoc.

## 8. Data Ownership

PostgreSQL la source of truth cho:

- Movies/cache metadata can persist.
- Search jobs/results.
- Download tasks/state transitions.
- Media items/files.
- Storage profiles.

Local Spring cache chi de giam call TMDB trong mot process. Cache mat khi backend restart la chap nhan duoc cho desktop.

## 9. Storage Contract

Launcher/GUI phai chon media folder va truyen vao backend:

```env
DOWNLOAD_DEFAULT_SAVE_PATH=C:\Users\<user>\AppData\Local\NetflixTorrent\media\Movies
```

qBittorrent sidecar phai duoc cau hinh download vao cung path do. Neu qBittorrent tra path khac, post-process se fail vi backend khong tim thay file.

## 10. System Status Contract

`GET /api/v1/system/status` tra ve:

- overall status.
- active profiles.
- app mode.
- database status.
- storage writable status.
- ffprobe status.
- qBittorrent reachable/version.
- Jackett/Prowlarr reachable tuy theo provider.

GUI nen dung endpoint nay cho man hinh setup/diagnostics.

## 11. Local E2E Contract

`docker-compose.local.yml` chay:

- `postgres`
- `qbittorrent`
- `prowlarr`
- `jackett`
- `backend`

Backend container chay `worker,desktop` va mount chung `/data/media` voi qBittorrent. Day la cach test full E2E truoc khi launcher quan ly native sidecars.

## 12. Packaging Contract

Launcher can bundle:

- Backend jar tu `Backend/target/backend-0.0.1-SNAPSHOT.jar`.
- JRE 21.
- PostgreSQL runtime/data dir.
- qBittorrent runtime/config dir.
- Prowlarr hoac Jackett runtime/config dir.
- ffprobe binary.
- GUI assets.

Launcher can quan ly:

- port allocation/co dinh tren localhost.
- process lifecycle.
- first-run config.
- local token generation.
- log/data directory under `%LOCALAPPDATA%\NetflixTorrent`.
