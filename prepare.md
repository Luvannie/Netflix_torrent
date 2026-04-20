# Prepare Local And Launcher Test

Ngay cap nhat: 2026-04-20

Backend da duoc don ve huong desktop launcher:

```text
backend jar + SPRING_PROFILES_ACTIVE=worker,desktop
PostgreSQL sidecar
qBittorrent sidecar
Prowlarr hoac Jackett sidecar
ffprobe bundled
desktop GUI goi backend qua 127.0.0.1
```

Khong con yeu cau Redis, JWT demo auth, Docker production stack, API/worker tach rieng.

## 1. File nen giu de test

| File | Muc dich |
|---|---|
| `Backend/src/main/resources/application-local.yml` | Debug backend local qua IDE/Maven |
| `Backend/src/main/resources/application-worker.yml` | Bat worker processors |
| `Backend/src/main/resources/application-desktop.yml` | Profile launcher target |
| `docker-compose.local.yml` | Local E2E bang Docker, standalone |
| `Dockerfile.local-backend` | Build backend container co ffprobe cho local E2E |
| `env.example` | Template `.env` cho local E2E |
| `STARTUP.md` | Lenh start/test nhanh |
| `BACKEND_ARCHITECTURE_DETAILED.md` | Kien truc backend can code theo |
| `GUI_ARCHITECTURE.md` | Kien truc GUI desktop |
| `DESKTOP_LAUNCHER_ARCHITECTURE.md` | Kien truc launcher/sidecar |
| `TORRENT_SEARCH_WORKFLOW.md` | Workflow tim torrent |
| `DOWNLOAD_WORKER_WORKFLOW.md` | Workflow download/post-process |

## 2. Chay test backend

```powershell
cd D:\project\java_project\Netflix_torrent\Backend
mvn test
```

Build jar:

```powershell
mvn package -DskipTests
```

## 3. Local E2E bang Docker

Dung cach nay de test full workflow khi chua co launcher:

```powershell
cd D:\project\java_project\Netflix_torrent
Copy-Item env.example .env
```

Sua `.env`:

```env
TMDB_API_KEY=...
TORRENT_SEARCH_PROVIDER=prowlarr
PROWLARR_API_KEY=...
QBITTORRENT_PASSWORD=...
```

Start:

```powershell
docker compose -f docker-compose.local.yml up -d --build postgres qbittorrent prowlarr jackett backend
```

Kiem tra:

```powershell
docker compose -f docker-compose.local.yml ps
curl.exe http://127.0.0.1:18080/api/v1/system/status
```

URL:

```text
Backend:     http://127.0.0.1:18080
qBittorrent: http://127.0.0.1:18082
Prowlarr:    http://127.0.0.1:19696
Jackett:     http://127.0.0.1:19117
PostgreSQL:  127.0.0.1:15432
```

Trong Docker E2E, backend va qBittorrent cung thay `/data/media`, nen download worker co the post-process file da tai.

## 4. Launcher-style host test

Dung cach nay de test dung mo hinh launcher se goi backend jar tren host:

```powershell
cd D:\project\java_project\Netflix_torrent\Backend
$env:SPRING_PROFILES_ACTIVE="worker,desktop"
$env:DB_URL="jdbc:postgresql://127.0.0.1:15432/netflixtorrent"
$env:DB_USERNAME="netflixtorrent"
$env:DB_PASSWORD="local_postgres_password"
$env:TMDB_API_KEY="YOUR_TMDB_API_KEY"
$env:TORRENT_SEARCH_PROVIDER="prowlarr"
$env:PROWLARR_URL="http://127.0.0.1:19696"
$env:PROWLARR_API_KEY="YOUR_PROWLARR_API_KEY"
$env:QBITTORRENT_URL="http://127.0.0.1:18082"
$env:QBITTORRENT_USERNAME="admin"
$env:QBITTORRENT_PASSWORD="YOUR_QBITTORRENT_PASSWORD"
$env:DOWNLOAD_DEFAULT_SAVE_PATH="$env:LOCALAPPDATA\NetflixTorrent\media\Movies"
$env:APP_LOCAL_TOKEN_ENABLED="true"
$env:APP_LOCAL_TOKEN="replace-with-random-local-token"
$env:FFPROBE_PATH="ffprobe"
mvn spring-boot:run
```

Quan trong: qBittorrent native/sidecar phai luu file vao dung Windows path trong `DOWNLOAD_DEFAULT_SAVE_PATH`. Neu qBittorrent chay trong Docker va tra `/data/media`, backend host se khong doc duoc file.

## 5. Test workflow

Search TMDB:

```powershell
curl.exe "http://127.0.0.1:18080/api/v1/catalog/search?query=Inception"
```

Tao search job:

```powershell
curl.exe -X POST "http://127.0.0.1:18080/api/v1/search/jobs" -H "Content-Type: application/json" -H "X-App-Local-Token: replace-with-random-local-token" -d "{\"query\":\"Big Buck Bunny 2008\"}"
```

Poll search job:

```powershell
curl.exe "http://127.0.0.1:18080/api/v1/search/jobs/1"
```

Tao download task:

```powershell
curl.exe -X POST "http://127.0.0.1:18080/api/v1/downloads" -H "Content-Type: application/json" -H "X-App-Local-Token: replace-with-random-local-token" -d "{\"searchResultId\":123}"
```

Poll download:

```powershell
curl.exe "http://127.0.0.1:18080/api/v1/downloads/1"
```

Library va stream:

```powershell
curl.exe "http://127.0.0.1:18080/api/v1/library"
curl.exe -H "Range: bytes=0-1048575" -o sample.bin "http://127.0.0.1:18080/api/v1/streams/{mediaFileId}"
```

## 6. Checklist launcher MVP

- [x] Backend chay mot process voi `worker,desktop`.
- [x] Khong can Redis.
- [x] Khong can JWT/demo auth.
- [x] Co local-token cho write endpoints.
- [x] Co `/api/v1/system/status` cho GUI/launcher.
- [x] `ffprobe` doc tu `FFPROBE_PATH`.
- [ ] Launcher start/stop PostgreSQL sidecar.
- [ ] Launcher start/stop qBittorrent sidecar.
- [ ] Launcher start/stop Prowlarr/Jackett sidecar.
- [ ] Launcher sinh va truyen `APP_LOCAL_TOKEN`.
- [ ] Launcher dam bao qBittorrent save path trung `DOWNLOAD_DEFAULT_SAVE_PATH`.
- [ ] GUI desktop goi status endpoint va hien thi loi setup.
