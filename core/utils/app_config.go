package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

const SpotifyClientIDHelpURL = "https://cassette?tab=readme-ov-file#set-up-your-spotify-client-id"

const (
	appConfigFileName           = "config.yml"
	spotifyClientIDPlaceholder  = "your_spotify_app_client_id"
	defaultAppConfigFileContent = "default_source: ytmusic\nauth:\n  client_id: your_spotify_app_client_id\n"
)

var (
	config        AppConfig
	configLoadErr error
	configMu      sync.RWMutex
)

func GetConfig() AppConfig {
	return config
}

func init() {
	config, configLoadErr = LoadConfig()
	if configLoadErr != nil {
		config = getDefaultAppConfig()
	}
}

type AppConfig struct {
	DefaultSource      string `mapstructure:"default_source" yaml:"default_source"`
	SeenWelcomeVersion string `mapstructure:"seen_welcome_version" yaml:"seen_welcome_version,omitempty"`
	LogLevel           string `mapstructure:"log_level" yaml:"log_level,omitempty"`
	Auth               struct {
		ClientID         string `mapstructure:"client_id" yaml:"client_id,omitempty"`
		Host             string `mapstructure:"host" yaml:"host,omitempty"`
		Port             int    `mapstructure:"port" yaml:"port,omitempty"`
		RedirectEndpoint string `mapstructure:"redirect-endpoint" yaml:"redirect-endpoint,omitempty"`
		Timeout          int    `mapstructure:"timeout" yaml:"timeout,omitempty"`
		Keyring          struct {
			Service string `mapstructure:"service" yaml:"service,omitempty"`
			Key     string `mapstructure:"key" yaml:"key,omitempty"`
		} `mapstructure:"keyring" yaml:"keyring,omitempty"`
	} `mapstructure:"auth" yaml:"auth,omitempty"`
	Player struct {
		SeekStepMs int `mapstructure:"seek-step-ms" yaml:"seek-step-ms,omitempty"`
		VolumeStep int `mapstructure:"volume-step" yaml:"volume-step,omitempty"`
	} `mapstructure:"player" yaml:"player,omitempty"`
}

func (c AppConfig) SpotifyClientID() string {
	return strings.TrimSpace(c.Auth.ClientID)
}

func getDefaultAppConfig() AppConfig {
	cfg := AppConfig{}
	cfg.DefaultSource = "ytmusic"
	cfg.LogLevel = "ERROR"
	cfg.Auth.Host = "127.0.0.1"
	cfg.Auth.Port = 8287
	cfg.Auth.RedirectEndpoint = "/callback"
	cfg.Auth.Timeout = 30
	cfg.Auth.Keyring.Service = "cassette"
	cfg.Auth.Keyring.Key = "token-v2"
	cfg.Player.SeekStepMs = 5000
	cfg.Player.VolumeStep = 5
	return cfg
}

// IsYouTubeMusicMode returns true when the active source is YouTube Music.
func IsYouTubeMusicMode() bool {
	src := strings.ToLower(strings.TrimSpace(GetConfig().DefaultSource))
	return src == "" || src == "ytmusic"
}

// SaveConfig writes the current in-memory config back to config.yml.
// Used to persist runtime mutations such as seen_welcome_version.
func SaveConfig(cfg AppConfig) error {
	configMu.Lock()
	defer configMu.Unlock()

	configDir := getConfigDir()
	if configDir == "" {
		return fmt.Errorf("cannot resolve config directory")
	}
	configPath := filepath.Join(configDir, appConfigFileName)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Update the global in-memory copy
	config = cfg
	return nil
}

// SetSeenWelcomeVersion is a convenience to mark the changelog popup as seen.
func SetSeenWelcomeVersion(version string) error {
	cfg := GetConfig()
	cfg.SeenWelcomeVersion = version
	return SaveConfig(cfg)
}

func LoadConfig() (AppConfig, error) {
	configDir, err := ensureAppConfigFile()
	if err != nil {
		return AppConfig{}, err
	}

	v := viper.New()
	applyConfigDefaults(v)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	err = v.ReadInConfig()
	var configErr viper.ConfigFileNotFoundError
	if err != nil && !errors.As(err, &configErr) {
		return AppConfig{}, err
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	var config AppConfig
	err = v.Unmarshal(&config)
	if err != nil {
		return AppConfig{}, err
	}
	return config, nil
}

func ValidateStartupConfig() error {
	if configLoadErr != nil {
		return fmt.Errorf("failed to load config: %w", configLoadErr)
	}
	return validateStartupConfig(config)
}

func validateStartupConfig(cfg AppConfig) error {
	// YouTube Music mode requires zero credentials — skip validation
	src := strings.ToLower(strings.TrimSpace(cfg.DefaultSource))
	if src == "ytmusic" || (src == "" && IsYouTubeMusicMode()) {
		return nil
	}
	if clientID := cfg.SpotifyClientID(); clientID == "" || clientID == spotifyClientIDPlaceholder {
		return fmt.Errorf("missing required config value `auth.client_id`; see %s", SpotifyClientIDHelpURL)
	}
	return nil
}

func ensureAppConfigFile() (string, error) {
	configDir := getConfigDir()
	if configDir == "" {
		return "", fmt.Errorf("failed to resolve user config directory")
	}
	if err := EnsureExists(configDir); err != nil {
		return "", err
	}

	configPath := filepath.Join(configDir, appConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return configDir, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	// Auto-migrate from ~/.config/lazyspotify/config.yml if it exists
	if dir, err := os.UserConfigDir(); err == nil {
		oldConfigPath := filepath.Join(dir, "lazyspotify", appConfigFileName)
		if data, err := os.ReadFile(oldConfigPath); err == nil && len(data) > 0 {
			if err := os.WriteFile(configPath, data, 0644); err == nil {
				return configDir, nil
			}
		}
	}

	if err := os.WriteFile(configPath, []byte(defaultAppConfigFileContent), 0644); err != nil {
		return "", err
	}
	return configDir, nil
}

func getConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	configDir := filepath.Join(dir, "cassette")
	return configDir
}

func SafeGetConfigDir() string {
	configDir := getConfigDir()
	EnsureExists(configDir)
	return configDir
}

func applyConfigDefaults(v *viper.Viper) {
	defaults := getDefaultAppConfig()
	v.SetDefault("default_source", defaults.DefaultSource)
	v.SetDefault("log_level", defaults.LogLevel)
	v.SetDefault("auth.host", defaults.Auth.Host)
	v.SetDefault("auth.port", defaults.Auth.Port)
	v.SetDefault("auth.redirect-endpoint", defaults.Auth.RedirectEndpoint)
	v.SetDefault("auth.timeout", defaults.Auth.Timeout)
	v.SetDefault("auth.keyring.service", defaults.Auth.Keyring.Service)
	v.SetDefault("auth.keyring.key", defaults.Auth.Keyring.Key)
	v.SetDefault("player.seek-step-ms", defaults.Player.SeekStepMs)
	v.SetDefault("player.volume-step", defaults.Player.VolumeStep)
}
