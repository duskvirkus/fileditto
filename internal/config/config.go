package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DBType string `json:"db_type"`           // "sqlite" | "postgres"
	DBPath string `json:"db_path,omitempty"` // SQLite only
	DBDSN  string `json:"db_dsn,omitempty"`  // PostgreSQL only
}

func dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "fileditto"), nil
}

func path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no config file found; run 'fileditto db configure'")
	}
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.DBType == "" {
		return nil, fmt.Errorf("config has no db_type; run 'fileditto db configure'")
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	p := filepath.Join(d, "config.json")
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
