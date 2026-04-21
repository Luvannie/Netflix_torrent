# BackendGo

Go migration target for the Netflix Torrent backend.

During migration this service runs beside the Java backend on port `18081`.
After parity verification it replaces the Java implementation under `Backend/`.

Run tests:

```powershell
go test ./...
```

Run locally:

```powershell
$env:SERVER_PORT="18081"
go run ./cmd/backend
```