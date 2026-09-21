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

// detectTerminalEmulator finds an installed terminal emulator for autostart.
// Returns the terminal command and its argument flag for running a child process.
func detectTerminalEmulator() (termBin string, execFlag string) {
	// Check $TERMINAL environment variable first
	if envTerm := os.Getenv("TERMINAL"); envTerm != "" {
		if p, err := exec.LookPath(envTerm); err == nil {
			return p, "-e"
		}
	}

	// Ordered preference list of terminal emulators
	terminals := []struct {
		name     string
		execFlag string
	}{
		{"konsole", "-e"},
		{"kitty", "-e"},
		{"alacritty", "-e"},
		{"wezterm", "start --"},
		{"foot", ""},
		{"gnome-terminal", "--"},
		{"xfce4-terminal", "-e"},
		{"xterm", "-e"},
	}

	for _, t := range terminals {
		if p, err := exec.LookPath(t.name); err == nil {
			return p, t.execFlag
		}
	}

	return "", ""
}

// EnableAutostart generates ~/.config/autostart/cassette.desktop that explicitly
// launches a terminal emulator running Cassette.
//
// Bug fix: Previously the .desktop file used Terminal=true, which causes desktop
// environments (KDE Plasma, GNOME) to run the process silently in the background
// without allocating a TTY. Bubble Tea then exits or hangs because it has no terminal.
//
// The fix detects the installed terminal emulator and writes an Exec= line that
// explicitly spawns the terminal with Cassette as its child process, with Terminal=false.
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

	// Build the Exec line with an explicit terminal emulator wrapper
	execLine := buildTerminalExecLine(bin)

	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Cassette
GenericName=Cassette Music Player
Comment=Cassette Retro Terminal Deck & Audio Player
Exec=%s
Icon=multimedia-audio-player
Terminal=false
Categories=Audio;Music;Player;ConsoleOnly;
StartupNotify=false
X-GNOME-Autostart-enabled=true
`, execLine)

	return os.WriteFile(p, []byte(desktopEntry), 0644)
}

// buildTerminalExecLine creates a shell wrapper that detects the terminal at runtime.
// If we can detect the terminal now, we use it directly. Otherwise we use a shell
// one-liner that probes multiple terminals at boot.
func buildTerminalExecLine(cassetteBin string) string {
	termBin, execFlag := detectTerminalEmulator()

	if termBin != "" {
		// Direct launch with detected terminal
		if execFlag != "" {
			return fmt.Sprintf("%s %s %s", termBin, execFlag, cassetteBin)
		}
		// foot uses positional args, no flag
		return fmt.Sprintf("%s %s", termBin, cassetteBin)
	}

	// Fallback: shell one-liner that probes terminals at boot time
	return fmt.Sprintf(`sh -c 'if command -v konsole >/dev/null 2>&1; then exec konsole -e "$0"; elif command -v kitty >/dev/null 2>&1; then exec kitty -e "$0"; elif command -v alacritty >/dev/null 2>&1; then exec alacritty -e "$0"; elif command -v foot >/dev/null 2>&1; then exec foot "$0"; elif command -v gnome-terminal >/dev/null 2>&1; then exec gnome-terminal -- "$0"; else exec xterm -e "$0"; fi' %s`, cassetteBin)
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
