package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/netflixtorrent/desktop/internal/config"
	"github.com/netflixtorrent/desktop/internal/processes"
)

const (
	databaseName     = "netflixtorrent"
	databaseUser     = "netflixtorrent"
	qbittorrentUser  = "admin"
	localhostAddress = "127.0.0.1"
)

func PrepareSidecarServices(cfg config.RuntimeConfig) ([]processes.Service, error) {
	cfg = normalizeRuntimeConfig(cfg)
	if err := writePostgresBootstrapContract(cfg); err != nil {
		return nil, err
	}
	if err := writeQBittorrentConfig(cfg); err != nil {
		return nil, err
	}
	if err := writeProviderConfig(cfg); err != nil {
		return nil, err
	}

	return []processes.Service{
		postgresService(cfg),
		qbittorrentService(cfg),
		providerService(cfg),
		backendService(cfg),
	}, nil
}

func BackendEnvironment(cfg config.RuntimeConfig) map[string]string {
	cfg = normalizeRuntimeConfig(cfg)
	return map[string]string{
		"SERVER_ADDRESS":             localhostAddress,
		"SERVER_PORT":                fmt.Sprint(cfg.Ports.Backend),
		"APP_MODE":                   "desktop",
		"APP_LOCAL_TOKEN_ENABLED":    "true",
		"APP_LOCAL_TOKEN":            cfg.LocalToken,
		"APP_BIND_LOCALHOST_ONLY":    "true",
		"DB_URL":                     fmt.Sprintf("jdbc:postgresql://%s:%d/%s", localhostAddress, cfg.Ports.Postgres, databaseName),
		"DB_USERNAME":                databaseUser,
		"DB_PASSWORD":                cfg.DatabasePassword,
		"QBITTORRENT_URL":            fmt.Sprintf("http://%s:%d", localhostAddress, cfg.Ports.QBittorrent),
		"QBITTORRENT_USERNAME":       qbittorrentUser,
		"QBITTORRENT_PASSWORD":       cfg.QBittorrentPassword,
		"PROWLARR_URL":               fmt.Sprintf("http://%s:%d", localhostAddress, cfg.Ports.Prowlarr),
		"JACKETT_URL":                fmt.Sprintf("http://%s:%d", localhostAddress, cfg.Ports.Jackett),
		"TORRENT_SEARCH_PROVIDER":    cfg.SearchProvider,
		"DOWNLOAD_DEFAULT_SAVE_PATH": cfg.DownloadDefaultSavePath,
	}
}

func postgresService(cfg config.RuntimeConfig) processes.Service {
	return processes.Service{
		Name:       "postgres",
		Executable: cfg.Executables.Postgres,
		Args: []string{
			"-D", cfg.Paths.PostgresDataDir,
			"-h", localhostAddress,
			"-p", fmt.Sprint(cfg.Ports.Postgres),
		},
		Environment: map[string]string{
			"PGDATA":        cfg.Paths.PostgresDataDir,
			"POSTGRES_DB":   databaseName,
			"POSTGRES_USER": databaseUser,
			"PGPASSWORD":    cfg.DatabasePassword,
		},
		WorkingDir: cfg.Paths.PostgresDataDir,
	}
}

func qbittorrentService(cfg config.RuntimeConfig) processes.Service {
	return processes.Service{
		Name:       "qbittorrent",
		Executable: cfg.Executables.QBittorrent,
		Args: []string{
			"--profile=" + cfg.Paths.QBittorrentDataDir,
			"--webui-port=" + fmt.Sprint(cfg.Ports.QBittorrent),
		},
		Environment: map[string]string{
			"QBT_PROFILE": cfg.Paths.QBittorrentDataDir,
		},
		WorkingDir: cfg.Paths.QBittorrentDataDir,
	}
}

func providerService(cfg config.RuntimeConfig) processes.Service {
	provider := strings.ToLower(strings.TrimSpace(cfg.SearchProvider))
	if provider == "jackett" {
		return processes.Service{
			Name:       "provider",
			Executable: cfg.Executables.Provider,
			Args:       []string{"--DataFolder", cfg.Paths.JackettDataDir},
			WorkingDir: cfg.Paths.JackettDataDir,
		}
	}

	return processes.Service{
		Name:       "provider",
		Executable: cfg.Executables.Provider,
		Args:       []string{"-data=" + cfg.Paths.ProwlarrDataDir, "-nobrowser"},
		WorkingDir: cfg.Paths.ProwlarrDataDir,
	}
}

func backendService(cfg config.RuntimeConfig) processes.Service {
	return processes.Service{
		Name:        "backend",
		Executable:  cfg.Executables.Backend,
		Environment: BackendEnvironment(cfg),
	}
}

func writePostgresBootstrapContract(cfg config.RuntimeConfig) error {
	if err := os.MkdirAll(cfg.Paths.PostgresDataDir, 0o755); err != nil {
		return err
	}
	content := strings.Join([]string{
		"-- Launcher-owned bootstrap contract for first-run database setup.",
		"-- Execute with APP_DB_PASSWORD supplied by the launcher environment.",
		"DO $$",
		"BEGIN",
		"  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'netflixtorrent') THEN",
		"    CREATE ROLE netflixtorrent WITH LOGIN PASSWORD :'APP_DB_PASSWORD';",
		"  END IF;",
		"END",
		"$$;",
		"CREATE DATABASE netflixtorrent OWNER netflixtorrent;",
		"",
	}, "\n")
	return os.WriteFile(filepath.Join(cfg.Paths.PostgresDataDir, "bootstrap.sql"), []byte(content), 0o600)
}

func writeQBittorrentConfig(cfg config.RuntimeConfig) error {
	configDir := filepath.Join(cfg.Paths.QBittorrentDataDir, "qBittorrent", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	content := strings.Join([]string{
		"[Preferences]",
		"WebUI\\Address=127.0.0.1",
		"WebUI\\Port=" + fmt.Sprint(cfg.Ports.QBittorrent),
		"WebUI\\LocalHostAuth=false",
		"WebUI\\AuthSubnetWhitelist=127.0.0.1",
		"WebUI\\AuthSubnetWhitelistEnabled=true",
		"Downloads\\SavePath=" + cfg.DownloadDefaultSavePath,
		"Downloads\\TempPath=" + filepath.Join(cfg.Paths.QBittorrentDataDir, "temp"),
		"",
	}, "\n")
	return os.WriteFile(filepath.Join(configDir, "qBittorrent.conf"), []byte(content), 0o600)
}

func writeProviderConfig(cfg config.RuntimeConfig) error {
	provider := strings.ToLower(strings.TrimSpace(cfg.SearchProvider))
	if provider == "jackett" {
		if err := os.MkdirAll(cfg.Paths.JackettDataDir, 0o755); err != nil {
			return err
		}
		content := fmt.Sprintf(`{"AllowExternal":false,"BindAddress":"127.0.0.1","Port":%d}`+"\n", cfg.Ports.Jackett)
		return os.WriteFile(filepath.Join(cfg.Paths.JackettDataDir, "ServerConfig.json"), []byte(content), 0o600)
	}

	if err := os.MkdirAll(cfg.Paths.ProwlarrDataDir, 0o755); err != nil {
		return err
	}
	content := strings.Join([]string{
		"<Config>",
		"  <BindAddress>127.0.0.1</BindAddress>",
		"  <Port>" + fmt.Sprint(cfg.Ports.Prowlarr) + "</Port>",
		"  <UrlBase></UrlBase>",
		"</Config>",
		"",
	}, "\n")
	return os.WriteFile(filepath.Join(cfg.Paths.ProwlarrDataDir, "config.xml"), []byte(content), 0o600)
}

func normalizeRuntimeConfig(cfg config.RuntimeConfig) config.RuntimeConfig {
	if cfg.SearchProvider == "" {
		cfg.SearchProvider = "prowlarr"
	}
	if cfg.Ports.Backend == 0 {
		cfg.Ports.Backend = 18080
	}
	if cfg.Ports.Postgres == 0 {
		cfg.Ports.Postgres = 15432
	}
	if cfg.Ports.QBittorrent == 0 {
		cfg.Ports.QBittorrent = 18082
	}
	if cfg.Ports.Prowlarr == 0 {
		cfg.Ports.Prowlarr = 19696
	}
	if cfg.Ports.Jackett == 0 {
		cfg.Ports.Jackett = 19117
	}
	if cfg.DownloadDefaultSavePath == "" {
		cfg.DownloadDefaultSavePath = cfg.Paths.DefaultMediaDir
	}
	return cfg
}
