package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jcdorr003/windash-agent/internal/config"
	"github.com/pkg/browser"
	"go.uber.org/zap"
)

// PairingAPI defines the interface for device pairing operations
// TODO: Replace MockPairingAPI with real HTTP client when backend is ready
type PairingAPI interface {
	RequestCode(ctx context.Context) (code string, expiresAt time.Time, err error)
	ExchangeCode(ctx context.Context, code string) (token string, err error)
}

// MockPairingAPI simulates the pairing flow for development/testing
type MockPairingAPI struct {
	logger *zap.SugaredLogger
}

// NewMockPairingAPI creates a new mock pairing API
func NewMockPairingAPI(logger *zap.SugaredLogger) *MockPairingAPI {
	return &MockPairingAPI{logger: logger}
}

// RequestCode simulates requesting a device code from the backend
func (m *MockPairingAPI) RequestCode(ctx context.Context) (string, time.Time, error) {
	m.logger.Info("🔐 [MOCK] Requesting device code from backend...")
	time.Sleep(500 * time.Millisecond) // Simulate network delay

	// Generate a fake device code (TODO: This will come from your backend)
	code := fmt.Sprintf("%04d-%04d", time.Now().Unix()%10000, time.Now().Unix()%10000)
	expiresAt := time.Now().Add(10 * time.Minute)

	m.logger.Info("✅ [MOCK] Device code generated", "code", code, "expiresAt", expiresAt.Format("15:04:05"))
	return code, expiresAt, nil
}

// ExchangeCode simulates polling for device approval
func (m *MockPairingAPI) ExchangeCode(ctx context.Context, code string) (string, error) {
	m.logger.Info("🔄 [MOCK] Polling for device approval...")

	// Simulate waiting for user to approve in the web dashboard
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
			m.logger.Info("⏳ [MOCK] Waiting for user to approve device...")
		}
	}

	// Generate a mock token (TODO: This will come from your backend after approval)
	token := fmt.Sprintf("mock_token_%d", time.Now().Unix())
	m.logger.Info("✅ [MOCK] Device approved! Token received")

	return token, nil
}

// EnsurePaired ensures the device is paired with the WinDash backend
// Returns (token, firstRun, error)
func EnsurePaired(ctx context.Context, api PairingAPI, store *TokenStore, cfg *config.Config, logger *zap.SugaredLogger) (token string, firstRun bool, err error) {
	// Get device ID
	deviceID, err := GetMachineID()
	if err != nil {
		return "", false, fmt.Errorf("failed to get device ID: %w", err)
	}

	// Check if already paired
	token, err = store.GetToken(deviceID)
	if err == nil && token != "" {
		logger.Debug("Device already paired", "deviceId", deviceID)
		return token, false, nil
	}

	// First run - need to pair
	logger.Info("🆕 First run detected - starting pairing flow...")
	fmt.Println("\n🆕 First time setup - Let's pair your device!\n")

	// Request device code from backend
	code, expiresAt, err := api.RequestCode(ctx)
	if err != nil {
		return "", true, fmt.Errorf("failed to request device code: %w", err)
	}

	// Save device code to config
	cfg.DeviceCode = code
	if err := cfg.Save(); err != nil {
		logger.Warn("Failed to save device code to config", "error", err)
	}

	// Build pairing URL
	pairingURL := fmt.Sprintf("%s/pair?code=%s", cfg.DashboardURL, code)

	// Show user-friendly instructions
	fmt.Printf("🔐 Your pairing code: %s\n\n", code)
	fmt.Printf("📋 To complete setup:\n")
	fmt.Printf("   1. Your browser will open automatically\n")
	fmt.Printf("   2. Log in to your WinDash account\n")
	fmt.Printf("   3. Approve this device\n\n")
	fmt.Printf("⏱️  Code expires at: %s\n\n", expiresAt.Format("15:04:05"))

	logger.Info("🌐 Opening browser for pairing", "url", pairingURL)

	// Open browser
	if err := browser.OpenURL(pairingURL); err != nil {
		logger.Warn("Failed to open browser automatically", "error", err)
		fmt.Printf("⚠️  Could not open browser automatically.\n")
		fmt.Printf("   Please visit: %s\n\n", pairingURL)
	}

	// Poll for token
	fmt.Println("⏳ Waiting for approval...")
	pollCtx, cancel := context.WithDeadline(ctx, expiresAt)
	defer cancel()

	token, err = api.ExchangeCode(pollCtx, code)
	if err != nil {
		return "", true, fmt.Errorf("pairing failed: %w", err)
	}

	// Store token securely
	if err := store.SaveToken(deviceID, token); err != nil {
		return "", true, fmt.Errorf("failed to save token: %w", err)
	}

	logger.Info("✅ Pairing complete!")
	fmt.Println("\n✅ Device paired successfully!\n")

	return token, true, nil
}

// OpenDashboard opens the WinDash dashboard in the default browser
func OpenDashboard(dashboardURL string) error {
	return browser.OpenURL(dashboardURL)
}
