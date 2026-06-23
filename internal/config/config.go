package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"dbconnector/internal/protocol"
)

const EnvConfigPath = "DBCONNECTOR_CONFIG"

type Config struct {
	Defaults Defaults  `json:"defaults"`
	Profiles []Profile `json:"profiles"`
}

type Defaults struct {
	Output     string `json:"output"`
	TimeoutMs  int    `json:"timeoutMs"`
	MaxRows    int    `json:"maxRows"`
	AllowWrite bool   `json:"allowWrite"`
}

type Profile struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DSN         string `json:"dsn,omitempty"`
	DSNEnv      string `json:"dsnEnv,omitempty"`
	Addr        string `json:"addr,omitempty"`
	UsernameEnv string `json:"usernameEnv,omitempty"`
	PasswordEnv string `json:"passwordEnv,omitempty"`
	DB          *int   `json:"db,omitempty"`
	Readonly    bool   `json:"readonly"`
	MaxRows     int    `json:"maxRows,omitempty"`
	TimeoutMs   int    `json:"timeoutMs,omitempty"`
}

type SanitizedProfile struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DSN         bool   `json:"dsnConfigured,omitempty"`
	DSNEnv      string `json:"dsnEnv,omitempty"`
	Addr        string `json:"addr,omitempty"`
	UsernameEnv string `json:"usernameEnv,omitempty"`
	PasswordEnv string `json:"passwordEnv,omitempty"`
	DB          *int   `json:"db,omitempty"`
	Readonly    bool   `json:"readonly"`
	MaxRows     int    `json:"maxRows,omitempty"`
	TimeoutMs   int    `json:"timeoutMs,omitempty"`
}

func DefaultPath() string {
	if configured := strings.TrimSpace(os.Getenv(EnvConfigPath)); configured != "" {
		return ExpandHome(configured)
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".dbconnector", "config.json")
	}
	return filepath.Join(home, ".dbconnector", "config.json")
}

func ExpandHome(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func Load(path string) (*Config, *protocol.Error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, protocol.NewError("CONFIG_NOT_FOUND", "config file not found: "+path, false)
		}
		return nil, protocol.NewError("CONFIG_INVALID", "failed to read config file: "+err.Error(), false)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, protocol.NewError("CONFIG_INVALID", "failed to parse config file: "+err.Error(), false)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func (c *Config) SanitizedProfiles() []SanitizedProfile {
	profiles := make([]SanitizedProfile, 0, len(c.Profiles))
	for _, profile := range c.Profiles {
		profiles = append(profiles, SanitizedProfile{
			Name:        profile.Name,
			Type:        profile.Type,
			DSN:         strings.TrimSpace(profile.DSN) != "",
			DSNEnv:      profile.DSNEnv,
			Addr:        profile.Addr,
			UsernameEnv: profile.UsernameEnv,
			PasswordEnv: profile.PasswordEnv,
			DB:          profile.DB,
			Readonly:    profile.Readonly,
			MaxRows:     profile.MaxRows,
			TimeoutMs:   profile.TimeoutMs,
		})
	}
	return profiles
}

func (c *Config) FindProfile(name string) (*Profile, bool) {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i], true
		}
	}
	return nil, false
}

func applyDefaults(cfg *Config) {
	if cfg.Defaults.Output == "" {
		cfg.Defaults.Output = "json"
	}
	if cfg.Defaults.TimeoutMs == 0 {
		cfg.Defaults.TimeoutMs = 5000
	}
	if cfg.Defaults.MaxRows == 0 {
		cfg.Defaults.MaxRows = 100
	}
}
