# WinDash Agent - AI Coding Guide

## Architecture Overview

**WinDash Agent** is a lightweight Go system monitoring agent that collects Windows PC metrics (CPU, memory, disk, network) and streams them to a dashboard via WebSocket. It's designed for Windows-first deployment (macOS/Linux support post-MVP).

### Core Components

- **`cmd/agent/main.go`**: Entry point - orchestrates pairing, metrics collection, WebSocket connection
- **`internal/auth/`**: Device pairing flow with mock API (backend integration pending) + secure token storage via Windows DPAPI
- **`internal/metrics/`**: Collects system metrics using `gopsutil/v4` every 2s (configurable)
- **`internal/ws/`**: WebSocket client with auto-reconnect (exponential backoff), backpressure handling, and batch sending (up to 10 samples/msg)
- **`internal/config/`**: Configuration from `%LOCALAPPDATA%\WinDash\agent.json`, environment variables (`WINDASH_*`), and defaults
- **`pkg/log/`**: Dual-output logging (colorized console + JSON file) with rotation via `lumberjack`

### Key Data Flow

```
Metrics Collector → Channel → Backpressure Buffer → WebSocket Client → Backend
     (every 2s)       (100 cap)    (drops oldest)        (batches 10)
```

## Critical Patterns

### 1. Real Pairing API Integration

`internal/auth/pairing.go` implements `RealPairingAPI` that integrates with the backend:
- `RequestCode()` → POST to `https://windash.jcdorr3.dev/api/device-codes` for device code
- `ExchangeCode()` → Poll `https://windash.jcdorr3.dev/api/device-token?code=<code>` every 2s until approved
- Returns 404 (pending), 410 (expired), or 200 with token (approved)
- `MockPairingAPI` still available for offline development/testing

### 2. Versioned Metrics Schema

All metrics use `SampleV1` struct with `V: 1` field for forward compatibility. When adding fields, create `SampleV2` to avoid breaking backend parsers.

### 3. WebSocket Backpressure

`ws/backpressure.go` drops oldest samples when buffer full (warns every 10 drops). Never blocks metric collection. Adjust `bufferSize` in `ws/client.go` if backend lags.

### 4. Network Rate Calculation

`metrics/collector.go` stores previous sample's byte counters to compute `TxBps`/`RxBps`. First sample always has 0 rates.

### 5. Platform-Specific Paths

`config/paths.go` uses `%LOCALAPPDATA%` and `%ProgramData%` on Windows. Has fallback for non-Windows dev environments (stores in `~/.config` and `~/.local/state`).

## Build & Development

### Common Commands (See `Makefile`)

```bash
make dev              # Run with hot reload via `go run`
make build-windows    # Cross-compile for Windows (CGO_ENABLED=0)
make lint             # go vet ./...
make deps             # go mod download && go mod tidy
```

### Build Version Injection

Version info embedded via ldflags in `Makefile`:
```bash
-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)
```
Check with `./WinDash-Agent.exe --version`

### Debugging

- Run with `--debug` flag for verbose logs
- Logs: `%ProgramData%\WinDash\logs\agent.log` (7-day rotation)
- Config: `%LOCALAPPDATA%\WinDash\agent.json`

## Project Conventions

1. **Emoji Logging**: Use emojis in user-facing messages (`main.go`) and logs for readability (e.g., `🚀 Agent starting`, `⚠️ Backpressure`)
2. **Sugared Logger**: All code uses `zap.SugaredLogger` for structured logging with key-value pairs
3. **Context Propagation**: Pass `context.Context` to all goroutines for clean shutdown on Ctrl+C
4. **No Tests Yet**: Test files don't exist (add when backend integration is ready)
5. **CGO Disabled**: All builds use `CGO_ENABLED=0` for static binaries (required for Windows distribution)

## Integration Points

### Backend API (Production Ready)

Real pairing endpoints in `internal/auth/pairing.go`:
- Device code request: `POST https://windash.jcdorr3.dev/api/device-codes` → `{"code": "ABCD-1234", "expiresAt": "..."}`
- Token exchange: `GET https://windash.jcdorr3.dev/api/device-token?code=<code>` (poll every 2s until approved)
  - 404 = still pending
  - 410 = expired (5-min timeout)
  - 200 = approved with `{"token": "..."}`

### WebSocket Protocol (`wss://windash.jcdorr3.dev/agent`)

**Agent → Server:**
```json
{"type": "metrics", "samples": [{"v": 1, "ts": "...", "hostId": "...", ...}]}
```

**Server → Agent (Control Messages):**
```json
{"type": "setRate", "intervalMs": 5000}  // Change collection interval
{"type": "pause"}                         // Stop metrics collection
{"type": "resume"}                        // Resume metrics collection
```

Control message handling stubbed in `ws/client.go` (marked with `[TODO]`).

## Post-MVP Features (See TODOs)

- System tray (`internal/tray/tray.go` skeleton exists)
- Runtime metrics interval adjustment (control message handler stubbed)
- macOS/Linux platform support (update `config/paths.go`)
- Windows code signing (`.goreleaser.yaml` placeholder)
- Auto-update mechanism

## Common Gotchas

- **Machine ID**: Generated via `machineid.ProtectedID()` - stable across reboots, unique per machine
- **Token Storage**: Uses `go-keyring` which maps to Windows Credential Manager (`com.windash.agent` service)
- **Disk Metrics**: Only reports non-removable partitions (`disk.Partitions(false)`)
- **WebSocket Compression**: Enabled via `permessage-deflate` for bandwidth efficiency
- **Config File**: Auto-created on first run with defaults. Delete to reset pairing.
