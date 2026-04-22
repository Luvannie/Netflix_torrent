# Production Readiness Tasks

## Muc tieu

Checklist nay tong hop cac dau viec con thieu de project dat muc `ready for production` theo huong:

- `Desktop/` la Wails launcher/native shell
- `frontend/` la GUI React/TypeScript
- `Backend/` la localhost API/WebSocket runtime
- Postgres, qBittorrent, Prowlarr/Jackett, ffprobe duoc quan ly nhu sidecars local

Tai lieu tham chieu chinh:

- `docs/superpowers/specs/2026-04-22-wails-gui-architecture-design.md`
- `docs/superpowers/specs/2026-04-22-windows-packaging-design.md`
- `docs/superpowers/plans/2026-04-22-wails-gui-foundation.md`
- `DESKTOP_LAUNCHER_ARCHITECTURE.md`
- `BACKEND_ARCHITECTURE_DETAILED.md`

## Hien trang

- `Backend/`: da co core API va runtime flow co ban
- `Desktop/`: da co runtime foundation slice gom config store, instance lock, process manager, diagnostics, bootstrap state, proxy token injection, va startup health coordination
- `frontend/`: moi o muc shell + typed API contracts, chua co UI flow that
- Packaging/release: chua co

## Milestone 1: Desktop Runtime Foundation

- [x] Dung Wails app that trong `Desktop/`, khong chi la scaffold `main.go/app.go`
- [x] Tao startup lifecycle day du: start PostgreSQL, qBittorrent, Prowlarr/Jackett, `backend.exe`
- [x] Hoan thien graceful shutdown cho backend va tat ca sidecars
- [x] Tao single-instance lock cho desktop app
- [x] Hoan thien `internal/proxy` de proxy REST va stream toi backend
- [x] Inject `X-App-Local-Token` trong Desktop proxy cho protected write APIs
- [x] Hoan thien native bridge: chon folder, mo log, restart sidecar, quit app
- [x] Hoan thien runtime config loader/saver trong `Desktop/internal/config`
- [x] Luu secrets local an toan bang DPAPI hoac co che tuong duong tren Windows
- [x] Hoan thien diagnostics snapshot va startup failure reporting

Ghi chu tien do:

- Da co `startup health coordination`: launcher chi chuyen `READY` sau khi backend health-check thanh cong, va chuyen `FAILED` neu health-check timeout/fail.
- Da co `os/exec` process runner, graceful shutdown cho child processes, native bridge mac dinh, proxy stream pass-through, va local secret store dung DPAPI tren Windows.
- `go test ./...` trong `Desktop/` da pass sau khi them Wails runtime wiring.
- `wails` CLI chua duoc cai trong moi truong local nay, nen build/package bang lenh `wails build` chua duoc verify truc tiep trong session.

## Milestone 2: First-Run va Runtime Ownership

- [ ] Tao directory layout trong `%LOCALAPPDATA%\\NetflixTorrent`
- [ ] Sinh local token, DB password, qBittorrent password o first run
- [ ] Bootstrap Postgres data directory va database `netflixtorrent`
- [ ] Bootstrap qBittorrent config va WebUI localhost bind
- [ ] Bootstrap Prowlarr hoac Jackett config
- [ ] Dong bo `DOWNLOAD_DEFAULT_SAVE_PATH` giua backend va qBittorrent
- [ ] Tao config schema versioning cho launcher config
- [ ] Hoan thien recovery flow neu sidecar hoac backend khoi dong that bai
- [ ] Hoan thien setup-complete flag va startup gate giua `StartupShell` / `SetupShell` / `AppShell`

## Milestone 3: Frontend App Shell va Navigation

- [ ] Dung router that cho `startup`, `setup`, `catalog`, `search-jobs`, `downloads`, `library`, `player`, `settings`, `diagnostics`
- [ ] Dung layout that cho `StartupShell`
- [ ] Dung layout that cho `SetupShell`
- [ ] Dung layout that cho `AppShell`
- [ ] Noi `frontend` voi Wails bindings that thay vi chi co contract types
- [ ] Noi `frontend` voi Desktop proxy cho REST calls bang relative URLs
- [ ] Noi `frontend` voi backend `/ws` cho realtime adapter
- [ ] Dung query client, query keys, loading states, empty states, error states
- [ ] Dung global app bootstrap state va runtime state

## Milestone 4: User Flows Chinh

### Setup

- [ ] Chon media root bang native folder picker
- [ ] Nhap TMDB credentials
- [ ] Chon provider: Prowlarr hoac Jackett
- [ ] Nhap provider credentials
- [ ] Validate `GET /api/v1/system/status`
- [ ] Tao hoac xac nhan storage profile mac dinh

### Catalog va Search

- [ ] Dung catalog landing page tu `GET /api/v1/catalog`
- [ ] Dung search page tu `GET /api/v1/catalog/search`
- [ ] Dung discover page tu `GET /api/v1/catalog/discover`
- [ ] Dung movie detail page tu `GET /api/v1/catalog/movies/{tmdbId}`
- [ ] Tao search job tu `POST /api/v1/search/jobs`
- [ ] Poll search job tu `GET /api/v1/search/jobs/{id}`
- [ ] Hien thi search results va cho user chon release

### Downloads

- [ ] Tao download task tu `POST /api/v1/downloads`
- [ ] Dung downloads list tu `GET /api/v1/downloads`
- [ ] Dung download detail tu `GET /api/v1/downloads/{id}`
- [ ] Dung cancel flow tu `POST /api/v1/downloads/{id}/cancel`
- [ ] Hien thi progress, speed, peer count, terminal status, error state

### Library va Player

- [ ] Dung library list tu `GET /api/v1/library`
- [ ] Dung media detail tu `GET /api/v1/library/{id}`
- [ ] Dung delete library item voi confirm dialog
- [ ] Dung player screen tu `GET /api/v1/streams/{id}`
- [ ] Dung metadata screen tu `GET /api/v1/streams/{id}/info`

### Settings va Diagnostics

- [ ] Dung storage profile CRUD UI
- [ ] Tach ro launcher-owned settings va backend-owned settings
- [ ] Dung diagnostics screen tu `GET /api/v1/system/status`
- [ ] Tao action restart backend / restart qBittorrent / restart provider
- [ ] Tao open logs va export logs flow

## Milestone 5: Backend Hardening Cho Desktop Production

- [ ] Hoan thien event publishing cho search jobs tren event bus
- [ ] Hoan thien event publishing cho downloads tren event bus
- [ ] Noi event bus vao `main.go` de GUI co realtime that
- [ ] Tighten auth cho destructive routes, dac biet library delete
- [ ] Chot contract cho integration settings: launcher-owned hay backend-owned APIs
- [ ] Them backend APIs neu can cho provider test / qBittorrent test / integration summary
- [ ] Xac nhan worker flow chay on dinh khi dung native sidecars, khong chi Docker local
- [ ] Xac nhan stream/player path handling dung voi filesystem path that tren Windows
- [ ] Hoan thien error messages de GUI co the hien thi ro rang cho user

## Milestone 6: Security

- [ ] Dam bao backend va sidecars chi bind `127.0.0.1`
- [ ] Dam bao local token khong bi expose trong frontend bundle
- [ ] Dam bao secrets khong duoc luu plaintext neu khong can thiet
- [ ] Dam bao qBittorrent WebUI khong mo ra LAN theo mac dinh
- [ ] Confirm lai protected write path coverage o backend
- [ ] Review logging de khong lo token/password vao logs
- [ ] Review uninstall/delete flow de tranh xoa nham custom user paths

## Milestone 7: Packaging va Installer

- [ ] Tao artifact `setup.exe` hoac `.msi`
- [ ] Bundle `Desktop`, `backend.exe`, Postgres, qBittorrent, Prowlarr/Jackett, ffprobe
- [ ] Hoan thien install layout trong `%ProgramFiles%`
- [ ] Hoan thien mutable state layout trong `%LOCALAPPDATA%`
- [ ] Hoan thien first-run bootstrap trong ban cai dat
- [ ] Hoan thien upgrade flow giu nguyen user data
- [ ] Hoan thien uninstall flow co tuy chon giu hoac xoa data/media
- [ ] Code-sign `setup.exe` va `.msi`

## Milestone 8: Testing va QA

- [ ] Chay `go test ./...` cho `Backend/`
- [x] Chay `go test ./...` cho `Desktop/`
- [ ] Dung frontend unit tests cho API client, bindings adapter, query flows, route logic
- [x] Dung integration tests cho Desktop proxy va bootstrap logic
- [ ] Dung E2E test cho startup -> setup -> catalog -> search -> download -> library -> player
- [ ] Test fresh install tren may Windows sach
- [ ] Test upgrade tu version cu
- [ ] Test uninstall giu data
- [ ] Test uninstall xoa toan bo data
- [ ] Test port collision, qBittorrent fail, Postgres fail, provider auth fail

## Milestone 9: Release va Van Hanh

- [ ] Tao release pipeline build artifact cho Windows
- [ ] Tao versioning strategy cho Desktop / frontend / backend bundle
- [ ] Tao changelog va release notes
- [ ] Hoan thien logging strategy cho Desktop, backend, sidecars
- [ ] Hoan thien support bundle/export logs cho bug report
- [ ] Hoan thien docs cho setup local, debug, packaging, release

## Definition of Done

Project co the coi la `ready for production` khi:

- [ ] User co the cai dat bang mot installer duy nhat
- [ ] App khoi dong duoc tren Windows ma khong can setup thu cong Postgres/qBittorrent
- [ ] GUI hoan thanh duoc setup, search, download, library, player, settings, diagnostics
- [ ] Backend + Desktop + frontend co test pass o muc can thiet
- [ ] Upgrade/uninstall an toan
- [ ] Secrets, logging, localhost-only binding, va auth da duoc review
- [ ] Release artifact duoc sign va co smoke test tren may sach

## Thu tu uu tien de lam tiep

1. Hoan thien `Desktop/` thanh Wails app that
2. Hoan thien `StartupShell` + `SetupShell` + runtime bootstrap
3. Hoan thien `catalog/search/download/library/player` trong `frontend/`
4. Hoan thien backend hardening cho realtime, auth, diagnostics
5. Hoan thien packaging + installer + release pipeline
