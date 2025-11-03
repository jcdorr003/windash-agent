package auth

import (
	"fmt"

	"github.com/denisbrodbeck/machineid"
	"github.com/jcdorr003/windash-agent/internal/config"
	"github.com/zalando/go-keyring"
	"go.uber.org/zap"
)

// TokenStore manages secure storage of authentication tokens
// Uses Windows DPAPI via go-keyring
type TokenStore struct {
	logger *zap.SugaredLogger
}

// NewTokenStore creates a new token store
func NewTokenStore(logger *zap.SugaredLogger) *TokenStore {
	return &TokenStore{logger: logger}
}

// SaveToken stores the authentication token securely in the OS keychain
func (s *TokenStore) SaveToken(deviceID, token string) error {
	s.logger.Debug("Saving token to keychain", "deviceId", deviceID)
	err := keyring.Set(config.KeychainService, deviceID, token)
	if err != nil {
		return fmt.Errorf("keychain save failed: %w", err)
	}
	s.logger.Info("🔐 Token saved securely to Windows Credential Manager")
	return nil
}

// GetToken retrieves the authentication token from the OS keychain
func (s *TokenStore) GetToken(deviceID string) (string, error) {
	s.logger.Debug("Retrieving token from keychain", "deviceId", deviceID)
	token, err := keyring.Get(config.KeychainService, deviceID)
	if err != nil {
		return "", err
	}
	s.logger.Debug("✅ Token retrieved from keychain")
	return token, nil
}

// DeleteToken removes the authentication token from the OS keychain
func (s *TokenStore) DeleteToken(deviceID string) error {
	s.logger.Debug("Deleting token from keychain", "deviceId", deviceID)
	return keyring.Delete(config.KeychainService, deviceID)
}

// GetMachineID returns a stable unique identifier for this machine
func GetMachineID() (string, error) {
	id, err := machineid.ProtectedID(config.AppID)
	if err != nil {
		return "", fmt.Errorf("failed to get machine ID: %w", err)
	}
	return id, nil
}
