package config

import (
	"encoding/json"
	"os"

	"github.com/spf13/viper"
)

const (
	EnvDefault                  = "remoteprod"
	DashboardURLLocalDev        = "http://localhost:5173"
	APIURLLocalDev              = "ws://localhost:3001/agent"
	DashboardURLLocalProd       = "http://localhost:3000"
	APIURLLocalProd             = "ws://localhost:3001/agent"
	DashboardURLRemoteProd      = "https://windash.jcdorr3.net"
	APIURLRemoteProd            = "wss://windash.jcdorr3.net/agent"
	DashboardURLLocalDockerProd = "http://192.168.1.57:3004"
	APIURLLocalDockerProd       = "ws://192.168.1.57:3005/agent"
	KeychainService             = "com.windash.agent"
)

// Config holds the agent configuration
type Config struct {
	Env               string `json:"env" mapstructure:"env"`
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
	v.SetDefault("env", EnvDefault)
	v.SetDefault("metricsIntervalMs", 2000)
	v.SetDefault("openOnStart", true)

	// Configure config file
	configFile := GetConfigFile()
	v.SetConfigFile(configFile)
	v.SetConfigType("json")

	// Read existing config (ignore error if file doesn't exist)
	_ = v.ReadInConfig()

	// Environment variables override (e.g., WINDASH_ENV)
	v.SetEnvPrefix("WINDASH")
	v.AutomaticEnv()

	// Unmarshal into struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// Set endpoints based on env, unless overridden in config
	switch cfg.Env {
	case "localdev":
		if cfg.DashboardURL == "" {
			cfg.DashboardURL = DashboardURLLocalDev
		}
		if cfg.APIURL == "" {
			cfg.APIURL = APIURLLocalDev
		}
	case "localprod":
		if cfg.DashboardURL == "" {
			cfg.DashboardURL = DashboardURLLocalProd
		}
		if cfg.APIURL == "" {
			cfg.APIURL = APIURLLocalProd
		}
	case "localdockerprod":
		if cfg.DashboardURL == "" {
			cfg.DashboardURL = DashboardURLLocalDockerProd
		}
		if cfg.APIURL == "" {
			cfg.APIURL = APIURLLocalDockerProd
		}
	case "remoteprod":
		if cfg.DashboardURL == "" {
			cfg.DashboardURL = DashboardURLRemoteProd
		}
		if cfg.APIURL == "" {
			cfg.APIURL = APIURLRemoteProd
		}
	default:
		// fallback to remoteprod
		if cfg.DashboardURL == "" {
			cfg.DashboardURL = DashboardURLRemoteProd
		}
		if cfg.APIURL == "" {
			cfg.APIURL = APIURLRemoteProd
		}
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
		Env:               EnvDefault,
		DashboardURL:      DashboardURLRemoteProd,
		APIURL:            APIURLRemoteProd,
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
