package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jcdorr003/windash-agent/internal/config"
	"github.com/pkg/browser"
	"go.uber.org/zap"
)

// PairingAPI defines the interface for device pairing operations
type PairingAPI interface {
	RequestCode(ctx context.Context) (code string, expiresAt time.Time, err error)
	ExchangeCode(ctx context.Context, code string) (token string, err error)
}

// RealPairingAPI implements device pairing with the WinDash backend
type RealPairingAPI struct {
	logger     *zap.SugaredLogger
	httpClient *http.Client
	baseURL    string
}

// NewRealPairingAPI creates a new real pairing API client
func NewRealPairingAPI(logger *zap.SugaredLogger, baseURL string) *RealPairingAPI {
	return &RealPairingAPI{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL, // This should be DashboardURL from config, which is set per env
	}
}

// deviceCodeResponse represents the response from POST /api/device-codes
type deviceCodeResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// deviceTokenResponse represents the response from GET /api/device-token
type deviceTokenResponse struct {
	Token    string `json:"token"`
	HostID   string `json:"hostId"`
	DeviceID string `json:"deviceId"`
}

// RequestCode requests a new device pairing code from the backend
func (r *RealPairingAPI) RequestCode(ctx context.Context) (string, time.Time, error) {
	r.logger.Info("🔐 Requesting device code from backend...")

	url := r.baseURL + "/api/device-codes"
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to decode response: %w", err)
	}

	r.logger.Info("✅ Device code received", "code", result.Code, "expiresAt", result.ExpiresAt.Format("15:04:05"))
	return result.Code, result.ExpiresAt, nil
}

// ExchangeCode polls the backend for device approval and token
func (r *RealPairingAPI) ExchangeCode(ctx context.Context, code string) (string, error) {
	r.logger.Info("🔄 Polling for device approval...")

	url := fmt.Sprintf("%s/api/device-token?code=%s", r.baseURL, code)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				r.logger.Warn("Failed to create request", "error", err)
				continue
			}

			resp, err := r.httpClient.Do(req)
			if err != nil {
				r.logger.Warn("Request failed", "error", err)
				continue
			}

			switch resp.StatusCode {
			case http.StatusOK:
				// Token approved!
				var result deviceTokenResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					resp.Body.Close()
					return "", fmt.Errorf("failed to decode token response: %w", err)
				}
				resp.Body.Close()
				r.logger.Info("✅ Device approved! Token received")
				return result.Token, nil

			case http.StatusNotFound:
				// Still pending
				resp.Body.Close()
				r.logger.Debug("⏳ Waiting for user to approve device...")

			case http.StatusGone:
				// Code expired
				resp.Body.Close()
				return "", fmt.Errorf("device code expired - please restart the agent")

			default:
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				r.logger.Warn("Unexpected status during polling", "status", resp.StatusCode, "body", string(body))
			}
		}
	}
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
	fmt.Println()
	fmt.Println("🆕 First time setup - Let's pair your device!")
	fmt.Println()

	// Request device code from backend
	code, expiresAt, err := api.RequestCode(ctx)
	if err != nil {
		fmt.Printf("\n❌ Failed to request device code from backend:\n")
		fmt.Printf("   Error: %v\n", err)
		fmt.Printf("   Backend URL: %s/api/device-codes\n\n", cfg.DashboardURL)
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
	fmt.Println()
	fmt.Println("✅ Device paired successfully!")
	fmt.Println()

	return token, true, nil
}

// OpenDashboard opens the WinDash dashboard in the default browser
func OpenDashboard(dashboardURL string) error {
	return browser.OpenURL(dashboardURL)
}
