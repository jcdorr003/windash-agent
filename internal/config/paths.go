package config

import (
	"os"
	"path/filepath"
)

const (
	AppName = "WinDash"
	AppID   = "windash-agent"
)

// GetConfigDir returns the configuration directory
// Windows: %LOCALAPPDATA%\WinDash
// TODO: Add macOS/Linux support post-MVP
func GetConfigDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		// Fallback for non-Windows during development
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", AppID)
	}
	return filepath.Join(localAppData, AppName)
}

// GetLogDir returns the log directory
// Windows: %ProgramData%\WinDash\logs
// TODO: Add macOS/Linux support post-MVP
func GetLogDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		// Fallback for non-Windows during development
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "state", AppID, "logs")
	}
	return filepath.Join(programData, AppName, "logs")
}

// GetConfigFile returns the full path to the config file
func GetConfigFile() string {
	return filepath.Join(GetConfigDir(), "agent.json")
}

// EnsureDirs creates config and log directories if they don't exist
func EnsureDirs() error {
	dirs := []string{GetConfigDir(), GetLogDir()}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
