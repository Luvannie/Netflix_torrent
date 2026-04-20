# Download Worker Workflow

Ngay cap nhat: 2026-04-19

Tai lieu nay mo ta luong worker tai video tu search result sang qBittorrent, post-process, va dua file vao media library.

## Dieu kien dau vao

Download worker xu ly `download_tasks` duoc tao tu `search_results`.

Search result phai co:

- `link`, `permalink`, hoac `guid` de lam torrent/magnet URL
- `hash` hoac magnet URL co `btih`

Ly do can `hash`: qBittorrent `/api/v2/torrents/add` thuong chi tra `"Ok."`, khong tra torrent hash. Backend can hash de poll `/api/v2/torrents/info?hashes=...`, cancel, va reconcile task sau restart.

## Cau hinh

```env
QBITTORRENT_URL=http://qbittorrent:8082
QBITTORRENT_USERNAME=admin
QBITTORRENT_PASSWORD=...
DOWNLOAD_DEFAULT_SAVE_PATH=/data/media
```

Worker chon save path theo thu tu:

1. Storage profile active co priority nho nhat.
2. `DOWNLOAD_DEFAULT_SAVE_PATH`.

Path nay phai ghi duoc tu backend/worker va cung phai la path qBittorrent thay duoc. Trong Docker Compose hien tai, ca worker va qBittorrent cung mount `/data/media`.

Trong desktop launcher target, khong dung `/data/media`. Launcher phai truyen cung mot Windows absolute path cho:

- `DOWNLOAD_DEFAULT_SAVE_PATH`
- qBittorrent save path
- storage profile mac dinh cua backend

Vi du:

```text
C:\Users\<user>\AppData\Local\NetflixTorrent\media\Movies
```

Neu qBittorrent luu file o path khac voi path worker doc duoc, post-process se fail vi worker khong tim thay file de chay `ffprobe`.

## State Machine Thuc Thi

Worker profile chay 3 scheduled loop:

1. Prepare loop:
   - xu ly `REQUESTED`, `SEARCHING`, `SEARCH_READY`, `QUEUED`
   - load `SearchResult`
   - resolve download URL va torrent hash
   - tao thu muc save path
   - add torrent vao qBittorrent neu chua ton tai
   - chuyen task sang `DOWNLOADING`

2. Poll loop:
   - xu ly `DOWNLOADING`
   - poll qBittorrent status bang torrent hash
   - cap nhat progress, speed, peer count
   - neu qBittorrent bao complete/seeding thi chuyen sang `POST_PROCESSING`
   - neu qBittorrent bao error/missing file thi mark `FAILED`

3. Post-process loop:
   - xu ly `POST_PROCESSING`
   - resolve file da tai tu qBittorrent `content_path`/`save_path`
   - neu path la folder, chon video file lon nhat
   - chay `ffprobe`
   - tao `media_items` va `media_files`
   - check stream readiness
   - neu ready: `STREAM_READY -> COMPLETED`
   - neu khong ready: `FAILED`

## Gioi Han Hien Tai

- Chua co transcode fallback. File codec/container khong direct-stream duoc se fail readiness.
- Chua co retry/backoff rieng cho tung task ngoai scheduled loop.
- Chua co field `save_path` trong `download_tasks`; worker resolve lai tu qBittorrent status va config.
- Chua co UI chon storage profile khi tao download task; worker dung active profile uu tien cao nhat hoac default path.
