package config

import (
	"strings"
	"testing"
)

func TestLoadDefaultsMatchJavaLocalProfile(t *testing.T) {
	cfg := Load()
	if cfg.AppName != "netflix-torrent-backend" {
		t.Fatalf("AppName = %q", cfg.AppName)
	}
	if cfg.Mode != "local" {
		t.Fatalf("Mode = %q", cfg.Mode)
	}
	if cfg.Server.Address != "127.0.0.1" || cfg.Server.Port != "8080" {
		t.Fatalf("Server = %#v", cfg.Server)
	}
	if cfg.Database.URL != "jdbc:postgresql://localhost:5433/netflixtorrent" {
		t.Fatalf("Database.URL = %q", cfg.Database.URL)
	}
	if cfg.Search.Provider != "jackett" || cfg.Search.MaxResults != 50 {
		t.Fatalf("Search = %#v", cfg.Search)
	}
	if !cfg.Network.BindLocalhostOnly {
		t.Fatalf("BindLocalhostOnly = false")
	}
}

func TestLoadReadsDesktopLauncherEnvironment(t *testing.T) {
	t.Setenv("APP_MODE", "desktop")
	t.Setenv("SPRING_PROFILES_ACTIVE", "worker,desktop")
	t.Setenv("SERVER_PORT", "18080")
	t.Setenv("DB_URL", "jdbc:postgresql://127.0.0.1:15432/netflixtorrent")
	t.Setenv("DB_USERNAME", "netflixtorrent")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("APP_LOCAL_TOKEN_ENABLED", "true")
	t.Setenv("APP_LOCAL_TOKEN", "token")
	t.Setenv("TORRENT_SEARCH_PROVIDER", "prowlarr")
	t.Setenv("APP_CORS_ALLOWED_ORIGINS", "http://127.0.0.1:18080,http://localhost:18080")

	cfg := Load()
	if cfg.Mode != "desktop" {
		t.Fatalf("Mode = %q", cfg.Mode)
	}
	if strings.Join(cfg.ActiveProfiles, ",") != "worker,desktop" {
		t.Fatalf("ActiveProfiles = %v", cfg.ActiveProfiles)
	}
	if !cfg.Auth.LocalTokenEnabled || cfg.Auth.LocalToken != "token" {
		t.Fatalf("Auth = %#v", cfg.Auth)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins = %v", cfg.CORS.AllowedOrigins)
	}
}

func TestPostgresURLConvertsJDBC(t *testing.T) {
	cfg := DatabaseConfig{
		URL:      "jdbc:postgresql://127.0.0.1:15432/netflixtorrent",
		Username: "netflixtorrent",
		Password: "secret",
	}
	got, err := cfg.PostgresURL()
	if err != nil {
		t.Fatalf("PostgresURL error = %v", err)
	}
	want := "postgres://netflixtorrent:secret@127.0.0.1:15432/netflixtorrent?sslmode=disable"
	if got != want {
		t.Fatalf("PostgresURL() = %q, want %q", got, want)
	}
}

func TestPostgresURLKeepsNativeFormat(t *testing.T) {
	cfg := DatabaseConfig{
		URL:      "postgres://dbuser:dbpass@localhost:5432/app?sslmode=require",
		Username: "ignored",
		Password: "ignored",
	}
	got, err := cfg.PostgresURL()
	if err != nil {
		t.Fatalf("PostgresURL error = %v", err)
	}
	want := "postgres://dbuser:dbpass@localhost:5432/app?sslmode=require"
	if got != want {
		t.Fatalf("PostgresURL() = %q, want %q", got, want)
	}
}

func TestSearchProviderNormalize(t *testing.T) {
	t.Setenv("TORRENT_SEARCH_PROVIDER", "PROWLARR")
	cfg := Load()
	if cfg.Search.Provider != "prowlarr" {
		t.Fatalf("Provider = %q", cfg.Search.Provider)
	}
}

func TestActiveProfilesFromSPRING_PROFILES_ACTIVE(t *testing.T) {
	t.Setenv("SPRING_PROFILES_ACTIVE", "local,worker")
	cfg := Load()
	if len(cfg.ActiveProfiles) != 2 {
		t.Fatalf("ActiveProfiles = %v", cfg.ActiveProfiles)
	}
}