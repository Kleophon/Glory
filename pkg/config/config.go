package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Visibility string   `json:"visibility"` // "public" or "private"
	Friends    []string `json:"friends"`    // Allowed public key hex strings if private
}

func GetConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "glory", "config.json"), nil
}

func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), nil
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig(), nil
	}
	return cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Visibility: "public",
		Friends:    []string{},
	}
}

func (c *Config) Save() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) IsAllowed(peerPubHex string) bool {
	if c.Visibility == "public" {
		return true
	}
	for _, f := range c.Friends {
		if f == peerPubHex {
			return true
		}
	}
	return false
}
