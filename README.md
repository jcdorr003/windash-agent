# WinDash Agent

> Lightweight Windows system monitoring agent - Send real-time PC metrics to your WinDash dashboard

## 🚀 Quick Start

### For Windows Users

1. **Download** the latest `WinDash-Agent.exe` from [Releases](https://github.com/jcdorr003/windash-agent/releases)
2. **Double-click** `WinDash-Agent.exe` to run it
3. **Approve** the device in your browser (opens automatically to `windash.jcdorr3.dev`)
4. **Done!** The agent is now sending metrics to your dashboard

The agent runs in the console window. Press Ctrl+C to stop it.

---

## 📋 What It Does

The WinDash Agent collects and sends these metrics to your dashboard every 2 seconds:

- **CPU** - Total and per-core usage %
- **Memory** - Used and total RAM
- **Disk** - Space used/available for all drives
- **Network** - Upload/download speeds (bytes/sec)
- **System** - Uptime and process count

---

## ⚙️ Configuration

The agent creates a config file on first run:

**Windows**: `%LOCALAPPDATA%\WinDash\agent.json`

```json
{
  "dashboardUrl": "https://windash.jcdorr3.dev",
  "apiUrl": "wss://windash.jcdorr3.dev/agent",
  "metricsIntervalMs": 2000,
  "openOnStart": true
}
```

### Options

- `dashboardUrl` - Your WinDash dashboard URL
- `apiUrl` - WebSocket endpoint for metrics
- `metricsIntervalMs` - How often to collect metrics (minimum 1000ms)
- `openOnStart` - Open dashboard in browser when agent starts

---

## 📝 Logs

Logs are automatically saved and rotated (keeps last 7 days):

**Windows**: `%ProgramData%\WinDash\logs\agent.log`

---

## 🔐 Security

- **Authentication tokens** are stored securely in Windows Credential Manager (DPAPI)
- **All communication** uses WSS (WebSocket Secure) with your backend
- **No sensitive data** is collected - only system performance metrics
- **Open source** - You can review all the code!

---

## 🛠️ For Developers

### Prerequisites

- Go 1.22 or higher (uses Go 1.24 for latest features)
- Git

### Build from Source

```bash
# Clone the repository
git clone https://github.com/jcdorr003/windash-agent.git
cd windash-agent

# Download dependencies
go mod download

# Run in development mode
make dev
# or
go run ./cmd/agent

# Build for Windows
make build-windows

# Build for all platforms
make build-all
```

### Project Structure

```
windash-agent/
├── cmd/agent/           # Main application entry point
├── internal/
│   ├── auth/            # Pairing & token management
│   ├── config/          # Configuration loading
│   ├── metrics/         # System metrics collection
│   ├── ws/              # WebSocket client
│   └── tray/            # System tray (optional)
└── pkg/log/             # Logging utilities
```

### Development Commands

```bash
make dev              # Run in development mode
make build            # Build for current platform
make build-windows    # Build Windows executable
make build-all        # Build for all platforms (requires goreleaser)
make clean            # Clean build artifacts
make lint             # Run linters
make test             # Run tests
make deps             # Download/update dependencies
```

### Build Variables

The build injects version info via ldflags:

```bash
go build -ldflags "-X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%d_%H:%M:%S)"
```

---

## 🔧 Architecture

### Pairing Flow

1. **First Run**: Agent requests device code from backend (currently using mock - returns instant code)
2. **Browser Opens**: User is directed to pairing page at `windash.jcdorr3.dev/pair?code=XXXX-XXXX`
3. **User Approves**: In the WinDash dashboard (backend integration pending)
4. **Token Issued**: Backend issues authentication token
5. **Token Stored**: Securely saved in Windows Credential Manager via DPAPI
6. **Subsequent Runs**: Token reused automatically, no re-pairing needed

### Current Status

- ✅ **Pairing UI Flow**: Opens browser to correct URL
- ⏳ **Backend Integration**: Mock API simulates 6-second approval (replace with real API)
- ✅ **Token Storage**: Windows Credential Manager integration working
- ⏳ **WebSocket**: Client ready, waiting for backend endpoint

### Metrics Collection

- Uses `gopsutil/v4` for cross-platform system metrics
- Collects samples every 2 seconds (configurable via `metricsIntervalMs`)
- Network rates calculated from byte deltas between collections
- Stable `hostId` generated from machine ID (persists across reboots)
- Zero-allocation metric collection for optimal performance

### WebSocket Client

- Auto-reconnect with exponential backoff (1s → 2min) + 20% jitter
- Backpressure handling: drops oldest samples if buffer full (warns every 10 drops)
- Batch sending: sends up to 10 samples per WebSocket message
- Heartbeat: pings every 10 seconds to keep connection alive
- Compression: permessage-deflate enabled
- Graceful shutdown: closes connection cleanly on Ctrl+C

---

## 📦 Release Process

```bash
# Tag a new version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Build release (requires goreleaser)
goreleaser release

# Or build snapshot for testing
goreleaser release --snapshot --clean
```

---

## 🗺️ Roadmap

- [x] Core metrics collection (CPU, RAM, Disk, Network)
- [x] WebSocket client with reconnect
- [x] Secure token storage
- [x] Mock pairing flow
- [ ] Real backend API integration
- [ ] System tray (optional)
- [ ] Auto-update
- [ ] Windows installer
- [ ] Start with OS (autostart)
- [ ] macOS & Linux support

---

## 📄 License

See [LICENSE](LICENSE) file.

---

## 🐛 Troubleshooting

### Agent won't start

- Check logs in `%ProgramData%\WinDash\logs\agent.log`
- Try running with `--debug` flag for verbose output
- Ensure no firewall blocking outbound connections

### Pairing fails

- Verify dashboard URL in config is correct
- Check internet connection
- Try deleting `agent.json` and restarting (re-pairs device)

### Metrics not showing

- Check WebSocket connection in logs
- Verify API URL in config
- Ensure backend is running and accessible

---

## 💬 Support

- GitHub Issues: [jcdorr003/windash-agent/issues](https://github.com/jcdorr003/windash-agent/issues)

---

**Made with ❤️ for Windows PC monitoring**
