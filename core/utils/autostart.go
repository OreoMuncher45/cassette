package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const AutostartDesktopFileName = "cassette.desktop"

// GetAutostartFilePath returns the full path to ~/.config/autostart/cassette.desktop
func GetAutostartFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "autostart", AutostartDesktopFileName), nil
}

// IsAutostartEnabled returns true if cassette.desktop exists in the autostart directory and is enabled.
func IsAutostartEnabled() bool {
	p, err := GetAutostartFilePath()
	if err != nil {
		return false
	}
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		return false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	content := string(data)
	if strings.Contains(content, "X-GNOME-Autostart-enabled=false") {
		return false
	}
	return strings.Contains(content, "Exec=")
}

// FindCassetteBinary resolves the absolute path to the cassette binary.
func FindCassetteBinary() (string, error) {
	if exe, err := os.Executable(); err == nil && exe != "" {
		if fi, err := os.Stat(exe); err == nil && !fi.IsDir() {
			return filepath.Clean(exe), nil
		}
	}
	if p, err := exec.LookPath("cassette"); err == nil && p != "" {
		return filepath.Clean(p), nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		localBin := filepath.Join(home, ".local", "bin", "cassette")
		if fi, err := os.Stat(localBin); err == nil && !fi.IsDir() {
			return localBin, nil
		}
	}
	return "cassette", nil
}

// EnableAutostart generates ~/.config/autostart/cassette.desktop with Terminal=true
func EnableAutostart(customBinPath ...string) error {
	p, err := GetAutostartFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}

	bin := ""
	if len(customBinPath) > 0 && customBinPath[0] != "" {
		bin = customBinPath[0]
	} else {
		bin, _ = FindCassetteBinary()
	}

	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Cassette
GenericName=Spotify Music Player
Comment=Spotify Terminal Deck & Audio Player
Exec=%s
Icon=multimedia-audio-player
Terminal=true
Categories=Audio;Music;Player;ConsoleOnly;
StartupNotify=false
X-GNOME-Autostart-enabled=true
`, bin)

	return os.WriteFile(p, []byte(desktopEntry), 0644)
}

// DisableAutostart removes ~/.config/autostart/cassette.desktop if present.
func DisableAutostart() error {
	p, err := GetAutostartFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ToggleAutostart flips the autostart status and returns the new enabled state.
func ToggleAutostart() (bool, error) {
	if IsAutostartEnabled() {
		err := DisableAutostart()
		return false, err
	}
	err := EnableAutostart()
	return true, err
}
