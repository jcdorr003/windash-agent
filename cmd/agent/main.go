package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jcdorr003/windash-agent/internal/auth"
	"github.com/jcdorr003/windash-agent/internal/config"
	"github.com/jcdorr003/windash-agent/internal/metrics"
	"github.com/jcdorr003/windash-agent/internal/ws"
	"github.com/jcdorr003/windash-agent/pkg/log"
)

var (
	// Build-time variables (set via ldflags)
	version   = "dev"
	buildTime = "unknown"
	goVersion = "unknown"
)

func main() {
	// Parse command-line flags
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	versionFlag := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// Show version and exit
	if *versionFlag {
		fmt.Printf("WinDash Agent %s\n", version)
		fmt.Printf("Built: %s\n", buildTime)
		fmt.Printf("Go: %s\n", goVersion)
		os.Exit(0)
	}

	// Initialize logger
	logger := log.New(*debugFlag)
	defer logger.Sync()

	// Welcome message
	logger.Info("🚀 WinDash Agent starting", "version", version)
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║       WinDash Agent v" + version + "          ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", "error", err)
	}

	logger.Info("📁 Configuration loaded",
		"configDir", cfg.ConfigDir,
		"logDir", cfg.LogDir,
		"metricsInterval", fmt.Sprintf("%dms", cfg.MetricsIntervalMs),
	)

	// Ensure directories exist
	if err := config.EnsureDirs(); err != nil {
		logger.Fatal("Failed to create directories", "error", err)
	}

	// Initialize pairing components
	pairingAPI := auth.NewRealPairingAPI(logger, cfg.DashboardURL)
	tokenStore := auth.NewTokenStore(logger)

	// Ensure device is paired
	token, firstRun, err := auth.EnsurePaired(context.Background(), pairingAPI, tokenStore, cfg, logger)
	if err != nil {
		logger.Fatal("Pairing failed", "error", err)
	}

	// Open browser if configured
	if cfg.OpenOnStart {
		if err := auth.OpenDashboard(cfg.DashboardURL); err != nil {
			logger.Warn("Failed to open browser", "error", err)
		} else {
			if firstRun {
				logger.Info("✨ Opened dashboard for first-time setup")
			} else {
				logger.Info("🌐 Opened dashboard")
			}
		}
	}

	// Get host information
	hostID, err := metrics.GetHostID()
	if err != nil {
		logger.Fatal("Failed to get host ID", "error", err)
	}

	logger.Info("🖥️  Host identified", "hostId", hostID)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics collector
	collector := metrics.NewCollector(
		logger,
		hostID,
		time.Duration(cfg.MetricsIntervalMs)*time.Millisecond,
	)
	sampleChan := make(chan *metrics.SampleV1, 100)

	go collector.Start(ctx, sampleChan)

	// Start WebSocket client
	wsClient := ws.NewClient(cfg.APIURL, token, hostID, logger)
	go wsClient.Run(ctx, sampleChan)

	// Success message
	logger.Info("✅ Agent running successfully")
	fmt.Println("✅ WinDash Agent is running!")
	fmt.Println("📊 Sending metrics to your dashboard")
	fmt.Println("🌐 Dashboard:", cfg.DashboardURL)
	fmt.Printf("📈 Collecting metrics every %dms\n", cfg.MetricsIntervalMs)
	fmt.Println("\nPress Ctrl+C to stop")
	fmt.Printf("\n📝 Logs: %s\\agent.log\n\n", cfg.LogDir)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	logger.Info("👋 Shutting down gracefully...")
	fmt.Println("\n\n👋 Shutting down...")

	cancel()
	time.Sleep(500 * time.Millisecond) // Give goroutines time to clean up

	logger.Info("✅ Goodbye!")
	fmt.Println("✅ Stopped. Goodbye!")
}
