# Netflix Torrent Startup

Ngay cap nhat: 2026-04-20

Repo hien uu tien mot huong chay:

```text
NetflixTorrent.exe
  -> starts bundled PostgreSQL/qBittorrent/Prowlarr or Jackett/ffprobe
  -> starts backend jar with SPRING_PROFILES_ACTIVE=worker,desktop
  -> opens desktop GUI
```

Trong luc launcher chua hoan thien, co hai cach test.

## 1. Backend unit/integration test

```powershell
cd D:\project\java_project\Netflix_torrent\Backend
mvn test
```

Build jar:

```powershell
cd D:\project\java_project\Netflix_torrent\Backend
mvn package -DskipTests
```

## 2. Local E2E bang Docker

Dung file `docker-compose.local.yml`. File nay la standalone, khong can `docker-compose.yml` rieng.

```powershell
cd D:\project\java_project\Netflix_torrent
Copy-Item env.example .env
docker compose -f docker-compose.local.yml up -d --build postgres qbittorrent prowlarr jackett backend
```

Cong mac dinh:

| Service | URL |
|---|---|
| Backend | http://127.0.0.1:18080 |
| Backend status | http://127.0.0.1:18080/api/v1/system/status |
| qBittorrent | http://127.0.0.1:18082 |
| Prowlarr | http://127.0.0.1:19696 |
| Jackett | http://127.0.0.1:19117 |
| PostgreSQL | 127.0.0.1:15432 |

Backend container chay profile `worker,desktop`, co ca API va worker trong mot process. Backend va qBittorrent cung mount `/data/media`, nen co the test full workflow download/post-process trong Docker.

## 3. Launcher-style host run

Dung khi muon test backend nhu cach launcher se start:

```powershell
cd D:\project\java_project\Netflix_torrent\Backend
$env:SPRING_PROFILES_ACTIVE="worker,desktop"
$env:DB_URL="jdbc:postgresql://127.0.0.1:15432/netflixtorrent"
$env:DB_USERNAME="netflixtorrent"
$env:DB_PASSWORD="local_postgres_password"
$env:TMDB_API_KEY="YOUR_TMDB_API_KEY"
$env:QBITTORRENT_URL="http://127.0.0.1:18082"
$env:QBITTORRENT_PASSWORD="YOUR_QBITTORRENT_PASSWORD"
$env:PROWLARR_URL="http://127.0.0.1:19696"
$env:PROWLARR_API_KEY="YOUR_PROWLARR_API_KEY"
$env:DOWNLOAD_DEFAULT_SAVE_PATH="$env:LOCALAPPDATA\NetflixTorrent\media\Movies"
$env:APP_LOCAL_TOKEN_ENABLED="true"
$env:APP_LOCAL_TOKEN="replace-with-random-local-token"
mvn spring-boot:run
```

Luu y: voi host run, qBittorrent cung phai tai ve vao chinh Windows path trong `DOWNLOAD_DEFAULT_SAVE_PATH`. Neu qBittorrent dang chay trong Docker va tra path `/data/media`, backend Windows se khong doc duoc file da tai.

## 4. Health/status

```powershell
curl.exe http://127.0.0.1:18080/api/v1/health
curl.exe http://127.0.0.1:18080/actuator/health/readiness
curl.exe http://127.0.0.1:18080/api/v1/system/status
```

`/api/v1/system/status` la endpoint GUI/launcher nen dung de hien thi trang thai database, storage, ffprobe, qBittorrent, Jackett va Prowlarr.
