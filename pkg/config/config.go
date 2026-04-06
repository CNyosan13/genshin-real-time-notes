package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"resin/pkg/logging"
)

const VERSION = "v0.0.8" // Set via ldflags

// Config holds HoyoLAB credentials and app settings.
// GenshinUID, HsrUID, ZzzUID are optional per-game overrides.
// If unset, the legacy UID field is used as fallback.
type Config struct {
	RefreshInterval int    `json:"refresh_interval"`
	// Legacy single-game UID (kept for backward compatibility)
	UID             string `json:"uid"`
	// Per-game UIDs for the unified hoyo binary
	GenshinUID      string `json:"genshin_uid,omitempty"`
	HsrUID          string `json:"hsr_uid,omitempty"`
	ZzzUID          string `json:"zzz_uid,omitempty"`
	Ltoken          string `json:"ltoken_v2"`
	Ltuid           string `json:"ltuid_v2"`
	DarkMode        bool   `json:"dark_mode"`
	ResinNotifyThreshold   int `json:"resin_notify_threshold"`
	StaminaNotifyThreshold int `json:"stamina_notify_threshold"`
	ChargeNotifyThreshold int `json:"charge_notify_threshold"`
	MaxResin               int `json:"max_resin"`
	MaxStamina             int `json:"max_stamina"`
	MaxCharge              int `json:"max_charge"`
}

// GetGenshinUID returns the Genshin UID, falling back to the legacy UID.
func (c *Config) GetGenshinUID() string {
	if c.GenshinUID != "" {
		return c.GenshinUID
	}
	return c.UID
}

// GetHsrUID returns the HSR UID, falling back to the legacy UID.
func (c *Config) GetHsrUID() string {
	if c.HsrUID != "" {
		return c.HsrUID
	}
	return c.UID
}

// GetZzzUID returns the ZZZ UID, falling back to the legacy UID.
func (c *Config) GetZzzUID() string {
	if c.ZzzUID != "" {
		return c.ZzzUID
	}
	return c.UID
}

func LoadJSON[T any](reader io.Reader) (*T, error) {
	var cfg T
	bytesValue, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytesValue, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return &cfg, nil
}

func WriteConfig(cfg *Config, configPath string) error {
	var bt []byte
	var err error
	if bt, err = json.MarshalIndent(cfg, "", "    "); err != nil {
		return err
	}
	if err = os.WriteFile(configPath, bt, 0755); err != nil {
		return err
	}
	return nil
}

func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); err != nil {
		logging.Fail("Unable to read config %s", configPath)
		return nil, err
	}
	var cfg Config
	jsonFile, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	bytesValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytesValue, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", configPath, err)
	}

	// Ensure at least one second of wait time before refresh
	cfg.RefreshInterval = max(1, cfg.RefreshInterval)

	return &cfg, nil
}
