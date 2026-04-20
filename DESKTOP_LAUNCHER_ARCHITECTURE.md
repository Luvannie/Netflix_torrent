# Desktop Launcher Architecture

Ngay cap nhat: 2026-04-20

Muc tieu phan phoi:

```text
NetflixTorrent.exe
  -> bundled GUI
  -> bundled JRE
  -> backend jar, profiles worker,desktop
  -> PostgreSQL/qBittorrent/Prowlarr or Jackett sidecars
  -> ffprobe
```

## 1. Launcher Responsibilities

Launcher la process cha. No phai:

- Tao app data directories.
- Sinh local token.
- Start PostgreSQL sidecar.
- Start qBittorrent sidecar.
- Start Prowlarr hoac Jackett sidecar.
- Start backend jar voi `SPRING_PROFILES_ACTIVE=worker,desktop`.
- Cho backend healthy.
- Mo desktop GUI.
- Stop child processes khi app thoat.

## 2. Directory Layout

De xuat:

```text
%LOCALAPPDATA%\NetflixTorrent\
  config\
    application-desktop.yml
  data\
    postgres\
    qbittorrent\
    prowlarr\
    jackett\
  logs\
  media\
    Movies\
```

Installer/app bundle:

```text
NetflixTorrent\
  NetflixTorrent.exe
  runtime\jre\
  backend\backend.jar
  sidecars\
    postgres\
    qbittorrent\
    prowlarr\
    jackett\
    ffmpeg\ffprobe.exe
  gui\
```

## 3. Backend Launch Command

Shape:

```powershell
runtime\jre\bin\java.exe `
  -jar backend\backend.jar `
  --spring.profiles.active=worker,desktop `
  --spring.config.additional-location=file:%LOCALAPPDATA%\NetflixTorrent\config\application-desktop.yml
```

Required env/config:

```env
SERVER_ADDRESS=127.0.0.1
SERVER_PORT=18080
DB_URL=jdbc:postgresql://127.0.0.1:15432/netflixtorrent
DB_USERNAME=netflixtorrent
DB_PASSWORD=<generated-or-installer-secret>
QBITTORRENT_URL=http://127.0.0.1:18082
QBITTORRENT_USERNAME=admin
QBITTORRENT_PASSWORD=<qbit-password>
PROWLARR_URL=http://127.0.0.1:19696
PROWLARR_API_KEY=<api-key>
DOWNLOAD_DEFAULT_SAVE_PATH=%LOCALAPPDATA%\NetflixTorrent\media\Movies
APP_LOCAL_TOKEN_ENABLED=true
APP_LOCAL_TOKEN=<random-token>
FFPROBE_PATH=<bundle>\sidecars\ffmpeg\ffprobe.exe
```

## 4. Sidecar Ports

| Sidecar | Default port |
|---|---|
| Backend | `18080` |
| PostgreSQL | `15432` |
| qBittorrent Web UI | `18082` |
| Prowlarr | `19696` |
| Jackett | `19117` |

Tat ca phai bind `127.0.0.1`.

## 5. First Run Flow

1. Kiem tra data directories.
2. Neu chua co config, tao config mac dinh.
3. Sinh DB password/local token.
4. Start PostgreSQL va apply Flyway qua backend startup.
5. Start qBittorrent.
6. Start Prowlarr hoac Jackett.
7. Start backend.
8. Goi `/api/v1/system/status`.
9. Neu service nao DOWN, GUI hien wizard sua loi.

## 6. GUI Contract

GUI khong tu thao tac DB hay sidecar truc tiep. GUI chi goi backend:

- `GET /api/v1/system/status`
- catalog/search/download/library/stream endpoints.
- WebSocket `/ws` neu can realtime progress.

Write request gui can gui:

```text
X-App-Local-Token: <launcher token>
```

## 7. Local E2E Before Launcher

Dung:

```powershell
docker compose -f docker-compose.local.yml up -d --build postgres qbittorrent prowlarr jackett backend
```

Local Docker E2E khac launcher o cho sidecars chay trong container. No van giu backend mot process voi `worker,desktop`, va backend/qBittorrent share `/data/media` de test download/post-process.

## 8. Remaining Launcher Work

- Tao launcher app.
- Bundle JRE va sidecars.
- Quan ly process lifecycle.
- Persist runtime config.
- First-run setup wizard.
- Log viewer/diagnostics UI.
- Installer/updater.
