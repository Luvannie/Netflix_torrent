# Torrent Source Search Workflow

Ngay cap nhat: 2026-04-19

Tai lieu nay mo ta phan tim nguon torrent hien tai cua backend.

## Muc tieu

Phan search torrent/source khong tai file truc tiep. No chi tim release torrent, luu metadata can thiet vao DB, va chuan bi du lieu de download worker ve sau co the dua release vao qBittorrent.

## Provider ho tro

Backend ho tro 3 che do qua `app.search.provider`:

- `jackett`: chi query Jackett
- `prowlarr`: chi query Prowlarr
- `both`: query ca hai, provider nao fail thi log warning; neu ca hai fail va khong co ket qua thi job fail

Bien moi truong tuong ung:

```env
TORRENT_SEARCH_PROVIDER=jackett
TORRENT_SEARCH_MAX_RESULTS=50
JACKETT_URL=http://jackett:9117
JACKETT_API_KEY=...
JACKETT_INDEXERS=all
PROWLARR_URL=http://prowlarr:9696
PROWLARR_API_KEY=...
```

Voi local non-Docker, URL mac dinh trong `application-local.yml` la:

```env
JACKETT_URL=http://localhost:9117
PROWLARR_URL=http://localhost:9696
```

Voi desktop launcher target, Jackett/Prowlarr se la sidecar bind `127.0.0.1` voi port do launcher quan ly, vi du:

```env
PROWLARR_URL=http://127.0.0.1:19696
JACKETT_URL=http://127.0.0.1:19117
```

Launcher phai luu API key provider vao config desktop ngoai source va restart backend/worker neu provider config thay doi.

## Workflow

1. Client goi `POST /api/v1/search/jobs` voi `query`.
2. API tao `search_jobs` o trang thai `REQUESTED`.
3. Worker profile poll job `REQUESTED` moi 5 giay.
4. Worker chuyen job sang `SEARCHING`.
5. Worker query provider theo `TORRENT_SEARCH_PROVIDER`.
6. Response tu Jackett/Prowlarr duoc map ve mot release model noi bo.
7. Worker normalize, deduplicate, score, va limit theo `TORRENT_SEARCH_MAX_RESULTS`.
8. `SearchJobService.completeWithResults` luu release vao `search_results`.
9. Job chuyen sang `SEARCH_READY`.
10. Worker publish WebSocket event `search.completed`.

Neu provider fail, worker goi `markFailed` va publish `search.failed`.

## Metadata duoc luu

Moi search result luu cac field can cho UI va download flow:

- `title`
- `guid`
- `link`
- `permalink`
- `size`
- `pubDate`
- `seeders`
- `leechers`
- `indexer`
- `provider`
- `hash`
- `score`

`link` uu tien magnet URL neu provider tra ve. `hash` uu tien `InfoHash`, neu khong co thi backend co gang extract `btih` tu magnet/guid.

## Diem da hoan thien trong dot nay

- Them `app.search.provider` va `app.search.max-results`.
- Them config/env/docker cho Jackett va Prowlarr.
- Jackett client map duoc shape JSON thuc te co field viet hoa nhu `Results`, `Title`, `Tracker`, `InfoHash`, `MagnetUri`.
- Prowlarr client map duoc response dang array hoac object co `results`.
- Worker co the chon `jackett`, `prowlarr`, hoac `both`.
- API search result response tra them `permalink`, `provider`, `hash`, va `score`.
- Them tests cho Jackett client, Prowlarr client, va search worker.

## Ket noi voi download worker

Download worker hien doc `search_results.link/permalink/guid` de add torrent vao qBittorrent va dung `search_results.hash` de poll tien do. Vi qBittorrent add API khong tra torrent hash, source search can uu tien luu `InfoHash` hoac extract `btih` tu magnet URL.

## Viec con lai

- Chua co UI/setup flow de lay API key Jackett/Prowlarr tu Web UI.
- Chua co UI chon storage profile khi tao download task.
- Scoring van la heuristic don gian: seeders + freshness. Co the bo sung rules ve quality, ngon ngu, codec, CAM/TS filter, va release group sau.
