package librespot

import (
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"go.yaml.in/yaml/v3"
	"strings"
	"testing"
)

func TestDaemonOwnsAuthentication(t *testing.T) {
	cfg := makeLibrespotConfig(utils.AppConfig{})
	if cfg.Credentials.Type != "device_auth" || cfg.ZeroconfEnabled {
		t.Fatalf("credentials = %+v", cfg.Credentials)
	}
	encoded, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "access_token") || strings.Contains(string(encoded), "spotify_token") {
		t.Fatal("config contains legacy OAuth credentials")
	}
}
