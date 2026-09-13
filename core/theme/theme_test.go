package theme

import (
	"path/filepath"
	"testing"
)

func TestThemeManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "settings.json")

	mgr := &Manager{
		settings: Settings{
			Scheme: SchemeRetroCyan,
			Effect: EffectStatic,
			Speed:  SpeedMedium,
		},
		artPrimary:   "#00f5d4",
		artSecondary: "#ffd166",
		configFile:   cfgPath,
	}

	// Test static primary color
	p := mgr.PrimaryHex()
	if p != "#00f5d4" {
		t.Fatalf("expected #00f5d4, got %s", p)
	}

	// Test effect switching to Rainbow
	mgr.SetEffect(EffectRainbow)
	mgr.Tick()
	pRainbow := mgr.PrimaryHex()
	if len(pRainbow) != 7 || pRainbow[0] != '#' {
		t.Fatalf("expected valid hex rainbow color, got %s", pRainbow)
	}

	// Test effect switching to Breathing
	mgr.SetEffect(EffectBreathing)
	pBreath := mgr.PrimaryHex()
	if len(pBreath) != 7 || pBreath[0] != '#' {
		t.Fatalf("expected valid hex breathing color, got %s", pBreath)
	}

	// Test Album Art Scheme
	mgr.SetScheme(SchemeAlbumArt)
	mgr.SetEffect(EffectStatic)
	if mgr.PrimaryHex() != "#00f5d4" {
		t.Fatalf("expected art default #00f5d4, got %s", mgr.PrimaryHex())
	}

	// Test Persistence
	if err := mgr.SaveSettings(); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	mgr2 := &Manager{configFile: cfgPath}
	mgr2.LoadSettings()
	if mgr2.settings.Scheme != SchemeAlbumArt {
		t.Fatalf("expected loaded scheme to be album_art, got %s", mgr2.settings.Scheme)
	}
}
