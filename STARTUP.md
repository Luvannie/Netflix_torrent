# Netflix Torrent Startup

Ngay cap nhat: 2026-04-21

Repo da hoan thanh migration tu Java sang Go. Backend Go chay song song voi Java trong luc chuyen doi.

## Quy trinh hien tai

```text
NetflixTorrent.exe
  -> starts bundled PostgreSQL/qBittorrent/Prowlarr or Jackett/ffprobe
  -> starts backend executable with profile settings
  -> opens desktop GUI
```

## 1. Backend unit/integration test (Go)

```powershell
cd D:\project\java_project\Netflix_torrent\Backend
go test ./...
```

Build jar (Java - legacy):

```powershell
cd D:\project\java_project\Netflix_torrent\legacy\java-backend
mvn test
mvn package -DskipTests
```

## 2. Local E2E bang Docker (Go)

Dung file `docker-compose.go-local.yml`. File nay la standalone, chay Go backend tren port 18081.

```powershell
cd D:\project\java_project\Netflix_torrent
docker compose -f docker-compose.go-local.yml up -d --build postgres backend-go
```

Cong mac dinh (Go):

| Service | URL |
|---|---|
| Backend-Go | http://127.0.0.1:18081 |
| qBittorrent | http://127.0.0.1:18082 |
| Prowlarr | http://127.0.0.1:19696 |
| Jackett | http://127.0.0.1:19117 |
| PostgreSQL | 127.0.0.1:15433 |

## 3. Chay Java va Go song song

Luc migration, ca hai runtime co the chay dong thoi:

**Java backend (port 8080/18080):**
```powershell
cd D:\project\java_project\Netflix_torrent\legacy\java-backend
$env:SPRING_PROFILES_ACTIVE="worker,desktop"
$env:DB_URL="jdbc:postgresql://127.0.0.1:15433/netflixtorrent"
$env:DB_USERNAME="netflixtorrent"
$env:DB_PASSWORD="local_postgres_password"
$env:APP_LOCAL_TOKEN_ENABLED="true"
$env:APP_LOCAL_TOKEN="token"
$env:TMDB_API_KEY="YOUR_TMDB_API_KEY"
$env:QBITTORRENT_URL="http://127.0.0.1:18082"
$env:QBITTORRENT_PASSWORD="YOUR_QBITTORRENT_PASSWORD"
$env:DOWNLOAD_DEFAULT_SAVE_PATH="$env:LOCALAPPDATA\NetflixTorrent\media\Movies"
mvn spring-boot:run
```

**Go backend (port 18081):**
```powershell
cd D:\project\java_project\Netflix_torrent\Backend
$env:SERVER_PORT="18081"
$env:DB_URL="jdbc:postgresql://127.0.0.1:15433/netflixtorrent"
$env:DB_USERNAME="netflixtorrent"
$env:DB_PASSWORD="local_postgres_password"
$env:APP_LOCAL_TOKEN_ENABLED="true"
$env:APP_LOCAL_TOKEN="token"
$env:SPRING_PROFILES_ACTIVE="worker,desktop"
go run ./cmd/backend
```

## 4. Launcher-style host run (Go)

```powershell
cd D:\project\java_project\Netflix_torrent\Backend
$env:SPRING_PROFILES_ACTIVE="worker,desktop"
$env:DB_URL="jdbc:postgresql://127.0.0.1:15433/netflixtorrent"
$env:DB_USERNAME="netflixtorrent"
$env:DB_PASSWORD="local_postgres_password"
$env:APP_LOCAL_TOKEN_ENABLED="true"
$env:APP_LOCAL_TOKEN="token"
$env:DOWNLOAD_DEFAULT_SAVE_PATH="$env:LOCALAPPDATA\NetflixTorrent\media\Movies"
$env:FFPROBE_PATH="ffprobe"
$env:TORRENT_SEARCH_PROVIDER="prowlarr"
go run ./cmd/backend
```

## 5. Sau khi cutover hoan tat

Sau khi migration hoan tat, launcher se start `Backend/backend.exe` thay vi `java -jar backend.jar`:

```text
NetflixTorrent.exe
  -> starts bundled PostgreSQL/qBittorrent/Prowlarr or Jackett/ffprobe
  -> starts Backend/backend.exe with profile settings
  -> opens desktop GUI
```

## 6. Health/status (Go)

```powershell
curl.exe -H "X-Request-Id: health-check" http://127.0.0.1:18081/api/v1/health
curl.exe -H "X-Request-Id: status-check" http://127.0.0.1:18081/api/v1/system/status
```

## 7. Docker Compose cho Java (legacy)

Neu can chay Java backend trong Docker:

```powershell
cd D:\project\java_project\Netflix_torrent
docker compose -f docker-compose.local.yml up -d --build postgres qbittorrent prowlarr jackett backend
```

Cong mac dinh (Java):

| Service | URL |
|---|---|
| Backend-Java | http://127.0.0.1:18080 |
| qBittorrent | http://127.0.0.1:18082 |
| Prowlarr | http://127.0.0.1:19696 |
| Jackett | http://127.0.0.1:19117 |
| PostgreSQL | 127.0.0.1:15432 |
