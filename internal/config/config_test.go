package config

import (
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("QQ_APP_ID", "appid-123")
	t.Setenv("QQ_TOKEN", "token-abc")
	t.Setenv("QQ_SECRET", "secret-xyz")
	t.Setenv("ENV", "prod")
	t.Setenv("HTTP_ADDR", ":9999")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("BOT_DEBUG", "true")
	t.Setenv("LUMIADMIN_API_URL", "https://admin.example.com")
	t.Setenv("LUMIADMIN_API_TOKEN", "token-abc")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != EnvProd {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvProd)
	}
	if cfg.Bot.AppID != "appid-123" {
		t.Errorf("Bot.AppID = %q, want %q", cfg.Bot.AppID, "appid-123")
	}
	if cfg.Bot.Token != "token-abc" {
		t.Errorf("Bot.Token = %q, want %q", cfg.Bot.Token, "token-abc")
	}
	if cfg.Bot.Secret != "secret-xyz" {
		t.Errorf("Bot.Secret = %q, want %q", cfg.Bot.Secret, "secret-xyz")
	}
	if cfg.HTTP.Addr != ":9999" {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":9999")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want %q（prod 环境应为 json）", cfg.Log.Format, "json")
	}
	if !cfg.Bot.Debug {
		t.Errorf("Bot.Debug = %v, want true", cfg.Bot.Debug)
	}
	if cfg.LumiAdmin.APIURL != "https://admin.example.com" {
		t.Errorf("LumiAdmin.APIURL = %q", cfg.LumiAdmin.APIURL)
	}
	if cfg.LumiAdmin.APIToken != "token-abc" {
		t.Errorf("LumiAdmin.APIToken = %q", cfg.LumiAdmin.APIToken)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("QQ_APP_ID", "appid-123")
	t.Setenv("QQ_SECRET", "secret-xyz")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != EnvDev {
		t.Errorf("Env = %q, want %q（默认应为 dev）", cfg.Env, EnvDev)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":8080")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.Log.Format != "console" {
		t.Errorf("Log.Format = %q, want %q（dev 环境应为 console）", cfg.Log.Format, "console")
	}
	if cfg.Bot.Timeout != 5*time.Second {
		t.Errorf("Bot.Timeout = %v, want %v", cfg.Bot.Timeout, 5*time.Second)
	}
	if cfg.Notification.PrivateTarget != "" {
		t.Errorf("PrivateTarget = %q, want default empty", cfg.Notification.PrivateTarget)
	}
	if cfg.LumiAdmin.APIURL != "" || cfg.LumiAdmin.APIToken != "" {
		t.Errorf("LumiAdmin default = (%q, %q), want empty", cfg.LumiAdmin.APIURL, cfg.LumiAdmin.APIToken)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("QQ_APP_ID", "")
	t.Setenv("QQ_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected error for missing QQ_APP_ID / QQ_SECRET, got nil")
	}
}

func TestLoadInvalidEnv(t *testing.T) {
	t.Setenv("QQ_APP_ID", "appid-123")
	t.Setenv("QQ_SECRET", "secret-xyz")
	t.Setenv("ENV", "staging")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected error for invalid ENV, got nil")
	}
}
