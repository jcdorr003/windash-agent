package config

import (
	"encoding/json"
	"os"

	"github.com/spf13/viper"
)

const (
	DashboardURLDefault = "http://192.168.1.57:3004"
	APIURLDefault       = "ws://192.168.1.57:3005/agent"
	KeychainService     = "com.windash.agent"
)

// Config holds the agent configuration
type Config struct {
	DashboardURL      string `json:"dashboardUrl" mapstructure:"dashboardUrl"`
	APIURL            string `json:"apiUrl" mapstructure:"apiUrl"`
	MetricsIntervalMs int    `json:"metricsIntervalMs" mapstructure:"metricsIntervalMs"`
	OpenOnStart       bool   `json:"openOnStart" mapstructure:"openOnStart"`
	DeviceCode        string `json:"deviceCode,omitempty" mapstructure:"deviceCode"`
	ConfigDir         string `json:"-"`
	LogDir            string `json:"-"`
}

// Load reads configuration from file, environment variables, and defaults
func Load() (*Config, error) {
	// Ensure directories exist first
	if err := EnsureDirs(); err != nil {
		return nil, err
	}

	v := viper.New()

	// Set defaults
	v.SetDefault("dashboardUrl", DashboardURLDefault)
	v.SetDefault("apiUrl", APIURLDefault)
	v.SetDefault("metricsIntervalMs", 2000)
	v.SetDefault("openOnStart", true)

	// Configure config file
	configFile := GetConfigFile()
	v.SetConfigFile(configFile)
	v.SetConfigType("json")

	// Read existing config (ignore error if file doesn't exist)
	_ = v.ReadInConfig()

	// Environment variables override (e.g., WINDASH_DASHBOARD_URL)
	v.SetEnvPrefix("WINDASH")
	v.AutomaticEnv()

	// Unmarshal into struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// Set runtime paths
	cfg.ConfigDir = GetConfigDir()
	cfg.LogDir = GetLogDir()

	// Create default config file if it doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := writeDefaultConfig(configFile); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Save writes the current configuration to file
func (c *Config) Save() error {
	configFile := GetConfigFile()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// writeDefaultConfig creates a new config file with defaults and helpful comments
func writeDefaultConfig(path string) error {
	cfg := &Config{
		DashboardURL:      DashboardURLDefault,
		APIURL:            APIURLDefault,
		MetricsIntervalMs: 2000,
		OpenOnStart:       true,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
