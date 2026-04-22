package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	AppName        string
	Mode           string
	ActiveProfiles []string
	Server         ServerConfig
	Database       DatabaseConfig
	CORS           CORSConfig
	Auth           AuthConfig
	Network        NetworkConfig
	TMDB           TMDBConfig
	Search         SearchConfig
	QBittorrent    QBittorrentConfig
	Jackett        JackettConfig
	Prowlarr       ProwlarrConfig
	Download       DownloadConfig
	FFprobe        FFprobeConfig
}

type ServerConfig struct {
	Address string
	Port    string
}

type DatabaseConfig struct {
	URL      string
	Username string
	Password string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type AuthConfig struct {
	LocalTokenEnabled bool
	LocalToken        string
}

type NetworkConfig struct {
	BindLocalhostOnly bool
}

type TMDBConfig struct {
	Key            string
	ReadAccessToken string
	BaseURL        string
	ImageBaseURL   string
}

type SearchConfig struct {
	Provider   string
	MaxResults int
	PollRateMS int
}

type QBittorrentConfig struct {
	BaseURL    string
	Username   string
	Password   string
}

type JackettConfig struct {
	BaseURL  string
	APIKey   string
	Indexers string
}

type ProwlarrConfig struct {
	BaseURL string
	APIKey  string
}

type DownloadConfig struct {
	DefaultSavePath  string
	PrepareRateMS    int
	PollRateMS       int
	PostProcessRateMS int
}

type FFprobeConfig struct {
	Path           string
	TimeoutSeconds int
}

func Load() Config {
	return Config{
		AppName:  envString("APP_NAME", "netflix-torrent-backend"),
		Mode:    envString("APP_MODE", "local"),
		ActiveProfiles: activeProfiles(),
		Server: ServerConfig{
			Address: envString("SERVER_ADDRESS", "127.0.0.1"),
			Port:    envString("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL:      envString("DB_URL", "jdbc:postgresql://localhost:5433/netflixtorrent"),
			Username: envString("DB_USERNAME", "postgres"),
			Password: envString("DB_PASSWORD", "postgres123"),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(envString("APP_CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		},
		Auth: AuthConfig{
			LocalTokenEnabled: envBool("APP_LOCAL_TOKEN_ENABLED", false),
			LocalToken:        envString("APP_LOCAL_TOKEN", ""),
		},
		Network: NetworkConfig{
			BindLocalhostOnly: envBool("APP_BIND_LOCALHOST_ONLY", true),
		},
		TMDB: TMDBConfig{
			Key:            envString("TMDB_API_KEY", ""),
			ReadAccessToken: envString("TMDB_READ_ACCESS_TOKEN", ""),
			BaseURL:        envString("TMDB_API_BASE_URL", "https://api.themoviedb.org/3"),
			ImageBaseURL:   envString("TMDB_IMAGE_BASE_URL", "https://image.tmdb.org/t/p"),
		},
		Search: SearchConfig{
			Provider:   strings.ToLower(envString("TORRENT_SEARCH_PROVIDER", "jackett")),
			MaxResults: envInt("TORRENT_SEARCH_MAX_RESULTS", 50),
			PollRateMS: envInt("SEARCH_WORKER_POLL_MS", 3000),
		},
		QBittorrent: QBittorrentConfig{
			BaseURL:  envString("QBITTORRENT_URL", "http://qbittorrent:8082"),
			Username: envString("QBITTORRENT_USERNAME", "admin"),
			Password: envString("QBITTORRENT_PASSWORD", "adminadmin"),
		},
		Jackett: JackettConfig{
			BaseURL:  envString("JACKETT_URL", "http://jackett:9117"),
			APIKey:   envString("JACKETT_API_KEY", ""),
			Indexers: envString("JACKETT_INDEXERS", "all"),
		},
		Prowlarr: ProwlarrConfig{
			BaseURL: envString("PROWLARR_URL", "http://prowlarr:9696"),
			APIKey:  envString("PROWLARR_API_KEY", ""),
		},
		Download: DownloadConfig{
			DefaultSavePath:   envString("DOWNLOAD_DEFAULT_SAVE_PATH", "/data/media"),
			PrepareRateMS:     envInt("DOWNLOAD_PREPARE_RATE_MS", 5000),
			PollRateMS:        envInt("DOWNLOAD_POLL_RATE_MS", 5000),
			PostProcessRateMS: envInt("DOWNLOAD_POSTPROCESS_RATE_MS", 10000),
		},
		FFprobe: FFprobeConfig{
			Path:           envString("FFPROBE_PATH", "ffprobe"),
			TimeoutSeconds: envInt("FFPROBE_TIMEOUT_SECONDS", 30),
		},
	}
}

func (c DatabaseConfig) PostgresURL() (string, error) {
	raw := strings.TrimSpace(c.URL)
	if raw == "" {
		return "", fmt.Errorf("database url is empty")
	}

	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		if parsed.User == nil && c.Username != "" {
			parsed.User = url.UserPassword(c.Username, c.Password)
		}
		return parsed.String(), nil
	}

	const jdbcPrefix = "jdbc:postgresql://"
	if !strings.HasPrefix(raw, jdbcPrefix) {
		return "", fmt.Errorf("unsupported database url %q", raw)
	}

	parsed, err := url.Parse("postgres://" + strings.TrimPrefix(raw, jdbcPrefix))
	if err != nil {
		return "", err
	}
	if c.Username != "" {
		parsed.User = url.UserPassword(c.Username, c.Password)
	}
	query := parsed.Query()
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func activeProfiles() []string {
	value := envString("SPRING_PROFILES_ACTIVE", "")
	if value == "" {
		value = envString("SPRING_PROFILES_DEFAULT", "local")
	}
	return splitCSV(value)
}

func envString(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	var i int
	if _, err := fmt.Sscanf(value, "%d", &i); err != nil {
		return fallback
	}
	return i
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"local"}
	}
	return out
}
