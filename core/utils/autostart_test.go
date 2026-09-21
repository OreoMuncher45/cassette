package utils

import (
	"os"
	"strings"
	"testing"
)

func TestAutostartManagement(t *testing.T) {
	// Use temporary directory as USER_CONFIG_DIR
	tmpDir, err := os.MkdirTemp("", "cassette-test-config-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if origConfigDir != "" {
			os.Setenv("XDG_CONFIG_HOME", origConfigDir)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	if IsAutostartEnabled() {
		t.Errorf("expected autostart to be disabled initially")
	}

	testBin := "/usr/bin/cassette"
	if err := EnableAutostart(testBin); err != nil {
		t.Fatalf("EnableAutostart failed: %v", err)
	}

	if !IsAutostartEnabled() {
		t.Errorf("expected autostart to be enabled after EnableAutostart")
	}

	p, err := GetAutostartFilePath()
	if err != nil {
		t.Fatalf("GetAutostartFilePath failed: %v", err)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read desktop file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Exec=") || !strings.Contains(content, testBin) {
		t.Errorf("expected desktop file to contain Exec referencing %s, got:\n%s", testBin, content)
	}
	if !strings.Contains(content, "Terminal=false") {
		t.Errorf("expected desktop file to contain Terminal=false")
	}

	// Test Toggle (should disable)
	enabled, err := ToggleAutostart()
	if err != nil {
		t.Fatalf("ToggleAutostart failed: %v", err)
	}
	if enabled || IsAutostartEnabled() {
		t.Errorf("expected autostart to be disabled after toggle")
	}

	// Test Toggle (should re-enable)
	enabled, err = ToggleAutostart()
	if err != nil {
		t.Fatalf("ToggleAutostart failed: %v", err)
	}
	if !enabled || !IsAutostartEnabled() {
		t.Errorf("expected autostart to be enabled after second toggle")
	}

	// Disable cleanly
	if err := DisableAutostart(); err != nil {
		t.Fatalf("DisableAutostart failed: %v", err)
	}
	if IsAutostartEnabled() {
		t.Errorf("expected autostart to be disabled after DisableAutostart")
	}
}
