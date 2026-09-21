package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"cassette/buildinfo"
	"cassette/core/player"
	"cassette/core/utils"
)

var oauthRegex = regexp.MustCompile(`https://accounts\.spotify\.com/authorize\S+`)

func Run(args []string) bool {
	switch args[0] {
	case "auth":
		authHandler(args)
	case "play":
		playHandler(args)
	case "version":
		versionHandler(args)
	case "setup", "setup-pc", "player-setup", "install-deps":
		return setupPCHandler(args)
	default:
		printUsage()
	}
	return false
}

func setupPCHandler(args []string) bool {
	fmt.Println("╭──────────────────────────────────────────────────────────────────────────╮")
	fmt.Println("│                        CASSETTE SETUP ASSISTANT                          │")
	fmt.Println("╰──────────────────────────────────────────────────────────────────────────╯")

	mode := ""
	autostart := true // Enabled by default as a convenience
	for _, a := range args[1:] {
		switch a {
		case "--ytmusic", "-y":
			mode = "ytmusic"
		case "--spotify", "-s":
			mode = "spotify"
		case "--dual", "-d":
			mode = "dual"
		case "--minimal", "-m":
			mode = "minimal" // Legacy: same as spotify
		case "--full", "-f":
			mode = "full" // Legacy: same as spotify + chafa
		case "--autostart":
			autostart = true
		case "--no-autostart":
			autostart = false
		}
	}

	// Map legacy modes
	if mode == "minimal" {
		mode = "spotify"
	}
	if mode == "full" {
		mode = "spotify-full"
	}

	if mode == "" {
		reader := bufio.NewReader(os.Stdin)
		for {
			checkMark := "[✔]"
			checkStatus := "ENABLED"
			if !autostart {
				checkMark = "[ ]"
				checkStatus = "DISABLED"
			}

			fmt.Println("\nChoose your setup profile:")
			fmt.Println("")
			fmt.Println("  [1] YouTube Music (Recommended — Free, No Premium/Account Needed)")
			fmt.Println("      • Streams audio via mpv + yt-dlp. Zero API keys required.")
			fmt.Println("      • Studio version preference (Audio Track Videos over music videos).")
			fmt.Println("      • Built-in 24-bit Truecolor ANSI Half-Block Album Art (▀).")
			fmt.Println("")
			fmt.Println("  [2] Spotify (Spotify Connect & Web API)")
			fmt.Println("      • Terminal Spotify Connect audio daemon (librespot).")
			fmt.Println("      • Requires a free Spotify Client ID + Spotify Premium account.")
			fmt.Println("      • Built-in 24-bit Truecolor ANSI Half-Block Album Art (▀).")
			fmt.Println("")
			fmt.Println("  [3] Dual Setup (Both YouTube Music & Spotify)")
			fmt.Println("      • Installs both backends. Toggle source with F3 in the player.")
			fmt.Println("")
			fmt.Printf("  [4] %s Auto-start Cassette on boot (%s)\n", checkMark, checkStatus)
			fmt.Println("      • Launch Cassette automatically when logging into your desktop.")
			fmt.Println("      • Purely local Freedesktop file (~/.config/autostart/cassette.desktop).")
			fmt.Println("")
			fmt.Print("Enter choice [1: YouTube Music / 2: Spotify / 3: Dual / 4: Toggle Autostart] (default: 1): ")

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "4" || strings.EqualFold(input, "autostart") {
				autostart = !autostart
				if autostart {
					fmt.Println("\n--> Auto-start on boot: [✔] ENABLED")
				} else {
					fmt.Println("\n--> Auto-start on boot: [ ] DISABLED")
				}
				continue
			}

			switch input {
			case "2", "spotify":
				mode = "spotify"
			case "3", "dual":
				mode = "dual"
			default:
				mode = "ytmusic"
			}
			break
		}
	}

	autostartStatus := "[✔] ENABLED"
	if !autostart {
		autostartStatus = "[ ] DISABLED"
	}
	fmt.Printf("\n--> Selected Profile: %s\n", strings.ToUpper(mode))
	fmt.Printf("--> Auto-start on boot: %s\n\n", autostartStatus)

	success := true

	// --- YouTube Music dependencies ---
	if mode == "ytmusic" || mode == "dual" {
		success = setupYouTubeMusic() && success
	}

	// --- Spotify dependencies ---
	if mode == "spotify" || mode == "spotify-full" || mode == "dual" {
		if mode == "spotify-full" {
			setupChafa()
		}
		spotifyOk := setupSpotify()
		success = spotifyOk && success
	}

	// --- Write default_source to config ---
	cfg := utils.GetConfig()
	switch mode {
	case "ytmusic":
		cfg.DefaultSource = "ytmusic"
	case "spotify", "spotify-full":
		cfg.DefaultSource = "spotify"
	case "dual":
		cfg.DefaultSource = "ytmusic" // Default to ytmusic, user can toggle with F3
	}
	if err := utils.SaveConfig(cfg); err != nil {
		fmt.Printf("Notice: could not save config: %v\n", err)
	}

	// --- Autostart ---
	if autostart {
		if err := utils.EnableAutostart(); err == nil {
			fmt.Println("✓ Auto-start on boot configured (~/.config/autostart/cassette.desktop)")
		} else {
			fmt.Printf("Notice: could not configure autostart: %v\n", err)
		}
	} else {
		_ = utils.DisableAutostart()
	}

	if !success {
		fmt.Println("\nSetup completed with warnings. Some features may not work until dependencies are installed.")
		return mode == "ytmusic" // YouTube Music can still work without spotify
	}

	fmt.Println("\nLaunching Cassette...")
	time.Sleep(1 * time.Second)
	return true
}

// setupYouTubeMusic installs mpv and yt-dlp if needed.
func setupYouTubeMusic() bool {
	fmt.Println("── YouTube Music Setup ──")

	// Check mpv
	fmt.Println("Checking for 'mpv' (audio playback engine)...")
	mpvPath, err := exec.LookPath("mpv")
	if err != nil {
		fmt.Println("'mpv' not found. Attempting installation...")
		installPackage("mpv")
		mpvPath, err = exec.LookPath("mpv")
		if err != nil {
			fmt.Println("✗ mpv could not be installed. Please install manually: pacman -S mpv")
			return false
		}
	}
	fmt.Printf("✓ mpv found at: %s\n", mpvPath)

	// Check yt-dlp
	fmt.Println("Checking for 'yt-dlp' (YouTube audio extraction)...")
	ytdlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		fmt.Println("'yt-dlp' not found. Attempting installation...")
		installPackage("yt-dlp")
		ytdlpPath, err = exec.LookPath("yt-dlp")
		if err != nil {
			// Try downloading binary directly
			fmt.Println("Trying direct binary download...")
			home, _ := os.UserHomeDir()
			binDir := filepath.Join(home, ".local", "bin")
			_ = os.MkdirAll(binDir, 0755)
			dlCmd := exec.Command("curl", "-L", "-o", filepath.Join(binDir, "yt-dlp"),
				"https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp")
			dlCmd.Stdout = os.Stdout
			dlCmd.Stderr = os.Stderr
			if dlErr := dlCmd.Run(); dlErr == nil {
				_ = os.Chmod(filepath.Join(binDir, "yt-dlp"), 0755)
				ytdlpPath = filepath.Join(binDir, "yt-dlp")
			} else {
				fmt.Println("✗ yt-dlp could not be installed. Please install manually: pacman -S yt-dlp")
				return false
			}
		}
	}
	fmt.Printf("✓ yt-dlp found at: %s\n", ytdlpPath)
	fmt.Println("✓ YouTube Music backend ready (mpv + yt-dlp)")
	fmt.Println("")
	return true
}

// setupSpotify handles the librespot auth flow (existing behavior).
func setupSpotify() bool {
	fmt.Println("── Spotify Setup ──")
	fmt.Println("Checking for 'librespot' (Spotify Connect audio daemon)...")
	binPath, err := player.FindLibrespot()
	if err != nil {
		fmt.Println("librespot binary not found. Attempting auto-install...")
		installPackage("librespot")
		binPath, err = player.FindLibrespot()
		if err != nil {
			fmt.Println("Please install librespot manually: e.g. pacman -S librespot or paru -S librespot")
			return false
		}
	}

	fmt.Printf("✓ Found librespot at: %s\n", binPath)

	// Clean up any lingering background librespot processes using port 5588 or name cassette
	_ = exec.Command("pkill", "-f", "librespot.*cassette").Run()
	time.Sleep(200 * time.Millisecond)

	cacheDir := filepath.Join(utils.SafeGetConfigDir(), "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	credFile := filepath.Join(cacheDir, "credentials.json")
	_ = os.Remove(credFile) // clear stale creds

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
		fmt.Printf("Failed to open stdout pipe: %v\n", err)
		return false
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Failed to open stderr pipe: %v\n", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start librespot: %v\n", err)
		return false
	}

	fmt.Println("\nWaiting for Spotify authorization...")
	lineCh := make(chan string, 100)
	var wg sync.WaitGroup
	wg.Add(2)

	scanStream := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
	}

	go scanStream(stdoutPipe)
	go scanStream(stderrPipe)

	go func() {
		wg.Wait()
		close(lineCh)
	}()

	openedBrowser := false
	authenticated := false

	stopCheck := make(chan struct{})
	credFoundCh := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopCheck:
				return
			case <-ticker.C:
				if fi, err := os.Stat(credFile); err == nil && fi.Size() > 0 {
					select {
					case credFoundCh <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}()

	for {
		select {
		case <-credFoundCh:
			authenticated = true
		case line, ok := <-lineCh:
			if !ok {
				goto checkDone
			}
			if !openedBrowser {
				if match := oauthRegex.FindString(line); match != "" {
					openedBrowser = true
					fmt.Printf("\nSpotify Authorization URL:\n%s\n\nOpening browser automatically...\n", match)
					_ = utils.OpenBrowser(match)
				}
			}
			if strings.Contains(line, "Authenticated as") ||
				strings.Contains(line, "active device is") ||
				strings.Contains(line, "with session <") {
				authenticated = true
			}
		}
		if authenticated {
			break
		}
	}

checkDone:
	close(stopCheck)

	if !authenticated {
		if fi, err := os.Stat(credFile); err == nil && fi.Size() > 0 {
			authenticated = true
		}
	}

	if !authenticated {
		fmt.Println("\nSetup cancelled or Spotify authorization failed.")
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		return false
	}

	// Ensure credentials file is flushed to disk
	for i := 0; i < 20; i++ {
		if fi, err := os.Stat(credFile); err == nil && fi.Size() > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n✓ Authenticated successfully with Spotify!")
	fmt.Println("✓ Cassette audio player is configured and ready.")

	// Stop the setup daemon cleanly so it releases port 5588 and audio sinks
	if cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}

	fmt.Println("")
	return true
}

// setupChafa installs chafa for advanced terminal graphics.
func setupChafa() {
	fmt.Println("Checking for 'chafa'...")
	if _, err := exec.LookPath("chafa"); err != nil {
		fmt.Println("'chafa' not found. Attempting installation...")
		installPackage("chafa")
	}
	if p, err := exec.LookPath("chafa"); err == nil {
		fmt.Printf("✓ chafa is installed at: %s\n", p)
	} else {
		fmt.Println("Notice: chafa could not be auto-installed. Falling back to built-in ANSI album art.")
	}
}

// installPackage tries to install a package using the available package manager.
func installPackage(name string) {
	managers := []struct {
		check string
		args  []string
	}{
		{"pacman", []string{"sudo", "pacman", "-S", "--noconfirm", name}},
		{"paru", []string{"paru", "-S", "--noconfirm", name}},
		{"yay", []string{"yay", "-S", "--noconfirm", name}},
		{"apt", []string{"sudo", "apt", "install", "-y", name}},
		{"dnf", []string{"sudo", "dnf", "install", "-y", name}},
	}

	for _, m := range managers {
		if _, err := exec.LookPath(m.check); err == nil {
			fmt.Printf("Running: %s\n", strings.Join(m.args, " "))
			c := exec.Command(m.args[0], m.args[1:]...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err == nil {
				return
			}
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
	fmt.Println("  setup       Configure music source and local audio playback")
	fmt.Println("              Profiles: --ytmusic (default), --spotify, --dual")
	fmt.Println("              Options: --autostart, --no-autostart")
	fmt.Println("  auth        Authenticate with Spotify Web API")
	fmt.Println("  version     Print build metadata")
}
