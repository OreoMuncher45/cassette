package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"cassette/buildinfo"
	"cassette/core/player"
	"cassette/core/utils"
)

var oauthRegex = regexp.MustCompile(`https://accounts\.spotify\.com/authorize\S+`)

func Run(args []string) {
	switch args[0] {
	case "auth":
		authHandler(args)
	case "play":
		playHandler(args)
	case "version":
		versionHandler(args)
	case "setup", "setup-pc", "player-setup", "install-deps":
		setupPCHandler(args)
	default:
		printUsage()
	}
}

func setupPCHandler(args []string) {
	fmt.Println("=== Setting up Cassette local PC audio player (librespot) ===")
	binPath, err := player.FindLibrespot()
	if err != nil {
		fmt.Println("Error: librespot binary not found.")
		fmt.Println("Attempting auto-install...")
		if _, err := exec.LookPath("pacman"); err == nil {
			fmt.Println("Running: sudo pacman -S --noconfirm librespot")
			c := exec.Command("sudo", "pacman", "-S", "--noconfirm", "librespot")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			_ = c.Run()
		} else if _, err := exec.LookPath("paru"); err == nil {
			c := exec.Command("paru", "-S", "--noconfirm", "librespot")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			_ = c.Run()
		}
		binPath, err = player.FindLibrespot()
		if err != nil {
			fmt.Println("Please install librespot manually: e.g. pacman -S librespot or paru -S librespot")
			return
		}
	}

	fmt.Printf("Found librespot at: %s\n", binPath)
	cacheDir := filepath.Join(utils.SafeGetConfigDir(), "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.Remove(filepath.Join(cacheDir, "credentials.json")) // clear stale creds

	cmd := exec.Command(binPath,
		"--name", "cassette",
		"--backend", "pulseaudio",
		"--device-type", "computer",
		"--enable-oauth",
		"--oauth-port", "5588",
		"--cache", cacheDir,
	)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Failed to open pipe: %v\n", err)
		return
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start librespot: %v\n", err)
		return
	}

	fmt.Println("\nWaiting for Spotify authorization...")
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if match := oauthRegex.FindString(line); match != "" {
			fmt.Printf("\nAuthorization URL:\n%s\n\nOpening browser automatically...\n", match)
			_ = utils.OpenBrowser(match)
		}
		if strings.Contains(line, "Authenticated as") {
			fmt.Printf("\n✓ %s\n", line)
			fmt.Println("✓ Cassette successfully connected to Spotify!")
			fmt.Println("✓ 'cassette' is now active in your Spotify devices list.")
			fmt.Println("Run 'cassette' to start playing music through your terminal.")
			_ = cmd.Process.Signal(os.Interrupt)
			return
		}
	}
}

func authHandler(args []string) {
	if len(args) > 1 {
		printUsage()
		return
	}
}

func playHandler(args []string) {
	if len(args) != 2 {
		printUsage()
		return
	}
}

func versionHandler(args []string) {
	if len(args) != 1 {
		printUsage()
		return
	}

	_ = buildinfo.PrintVersion(os.Stdout)
}

func printUsage() {
	fmt.Println("Usage: cassette <command>")
	fmt.Println("Flags:")
	fmt.Println("  --version   Print build metadata")
	fmt.Println("Commands:")
	fmt.Println("  setup       Authenticate and configure local PC audio playback ('cassette' device)")
	fmt.Println("  auth        Authenticate with Spotify Web API")
	fmt.Println("  version     Print build metadata")
}
