package config

import (
	"strings"
	"testing"
)

func TestConfig_UIDFallbacks(t *testing.T) {
	cfg := &Config{
		UID:        "12345",
		GenshinUID: "88888",
		// HsrUID is empty
		// ZzzUID is empty
	}

	if cfg.GetGenshinUID() != "88888" {
		t.Errorf("expected 88888 for Genshin, got %s", cfg.GetGenshinUID())
	}
	if cfg.GetHsrUID() != "12345" {
		t.Errorf("expected fallback 12345 for HSR, got %s", cfg.GetHsrUID())
	}
	if cfg.GetZzzUID() != "12345" {
		t.Errorf("expected fallback 12345 for ZZZ, got %s", cfg.GetZzzUID())
	}
}

func TestLoadJSON_Config(t *testing.T) {
	jsonStr := `{
		"refresh_interval": 30,
		"uid": "111",
		"genshin_uid": "222",
		"ltoken_v2": "abc",
		"ltuid_v2": "def",
		"dark_mode": true
	}`

	cfg, err := LoadJSON[Config](strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("LoadJSON failed: %v", err)
	}

	if cfg.RefreshInterval != 30 {
		t.Errorf("expected 30 interval, got %d", cfg.RefreshInterval)
	}
	if cfg.GenshinUID != "222" {
		t.Errorf("expected 222 Genshin identifier, got %s", cfg.GenshinUID)
	}
	if cfg.Ltoken != "abc" {
		t.Errorf("expected 'abc' token, got %s", cfg.Ltoken)
	}
}
