package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Data     DataConfig     `json:"data"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port string `json:"port"`
	Host string `json:"host"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path string `json:"path"`
}

// DataConfig represents data configuration
type DataConfig struct {
	RawDataFolder           string      `json:"raw_data_folder"`
	TagListFile             string      `json:"tag_list_file"` // Deprecated: tag list is now from DB; this field is unused
	ValueRange              *ValueRange `json:"value_range,omitempty"`
	UseSequentialGeneration bool        `json:"use_sequential_generation"`
	GenerationStartTime     string      `json:"generation_start_time"`
	GenerationEndTime       string      `json:"generation_end_time"`
}

// ValueRange represents the range for random value generation
type ValueRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	// Default config path
	if configPath == "" {
		configPath = "config.json"
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults if not specified
	if config.Server.Port == "" {
		config.Server.Port = "8888"
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Database.Path == "" {
		config.Database.Path = "insightsim.db"
	}
	if config.Data.RawDataFolder == "" {
		config.Data.RawDataFolder = "raw_data"
	}
	if config.Data.TagListFile == "" {
		config.Data.TagListFile = "raw_data/tag_list.json"
	}
	// Set default value range if not specified
	if config.Data.ValueRange == nil {
		config.Data.ValueRange = &ValueRange{
			Min: 1.0,
			Max: 10000.0,
		}
	}
	// Validate value range
	if config.Data.ValueRange.Min >= config.Data.ValueRange.Max {
		return nil, fmt.Errorf("invalid value range: min (%f) must be less than max (%f)", config.Data.ValueRange.Min, config.Data.ValueRange.Max)
	}
	// Set default generation times if not specified
	if config.Data.GenerationStartTime == "" {
		config.Data.GenerationStartTime = "2025-12-01T00:00:00"
	}
	if config.Data.GenerationEndTime == "" {
		config.Data.GenerationEndTime = "2026-01-31T23:59:59"
	}

	return &config, nil
}

// LoadConfigWithDefaults loads config with fallback to defaults if file doesn't exist
func LoadConfigWithDefaults(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config.json"
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		// If file doesn't exist, return default config
		if os.IsNotExist(err) {
			return &Config{
				Server: ServerConfig{
					Port: "8888",
					Host: "0.0.0.0",
				},
				Database: DatabaseConfig{
					Path: "insightsim.db",
				},
				Data: DataConfig{
					RawDataFolder: "raw_data",
					TagListFile:   "raw_data/tag_list.json",
					ValueRange: &ValueRange{
						Min: 1.0,
						Max: 10000.0,
					},
					UseSequentialGeneration: false,
					GenerationStartTime:     "2025-12-01T00:00:00",
					GenerationEndTime:       "2026-01-31T23:59:59",
				},
			}, nil
		}
		return nil, err
	}

	return config, nil
}
