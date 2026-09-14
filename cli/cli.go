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
		if a == "--minimal" || a == "-m" {
			mode = "minimal"
		} else if a == "--full" || a == "-f" {
			mode = "full"
		} else if a == "--autostart" {
			autostart = true
		} else if a == "--no-autostart" {
			autostart = false
		}
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
			fmt.Println("  [1] Minimal Setup (Recommended)")
			fmt.Println("      • Terminal Spotify Connect audio daemon (librespot).")
			fmt.Println("      • Built-in 24-bit Truecolor ANSI Half-Block Album Art (▀).")
			fmt.Println("      • Zero extra dependencies needed. Works natively out of the")
			fmt.Println("        box in Konsole, Alacritty, Kitty, WezTerm, and any terminal.")
			fmt.Println("")
			fmt.Println("  [2] Full Setup")
			fmt.Println("      • Everything in Minimal (librespot + built-in ANSI art).")
			fmt.Println("      • Installs 'chafa' (Char Fast Art) for advanced terminal")
			fmt.Println("        sub-block dithering & multi-protocol scaling.")
			fmt.Println("")
			fmt.Printf("  [3] %s Auto-start Cassette on boot (%s)\n", checkMark, checkStatus)
			fmt.Println("      • Launch Cassette automatically when logging into your desktop.")
			fmt.Println("      • Purely local Freedesktop file (~/.config/autostart/cassette.desktop).")
			fmt.Println("")
			fmt.Print("Enter choice [1: Minimal / 2: Full / 3: Toggle Autostart] (default: 1): ")

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "3" || strings.EqualFold(input, "autostart") {
				autostart = !autostart
				if autostart {
					fmt.Println("\n--> Auto-start on boot: [✔] ENABLED")
				} else {
					fmt.Println("\n--> Auto-start on boot: [ ] DISABLED")
				}
				continue
			}

			if input == "2" || strings.EqualFold(input, "full") {
				mode = "full"
			} else {
				mode = "minimal"
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

	if mode == "full" {
		fmt.Println("Checking for 'chafa'...")
		if _, err := exec.LookPath("chafa"); err != nil {
			fmt.Println("'chafa' not found. Attempting installation...")
			if _, err := exec.LookPath("pacman"); err == nil {
				c := exec.Command("sudo", "pacman", "-S", "--noconfirm", "chafa")
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				_ = c.Run()
			} else if _, err := exec.LookPath("paru"); err == nil {
				c := exec.Command("paru", "-S", "--noconfirm", "chafa")
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				_ = c.Run()
			} else if _, err := exec.LookPath("yay"); err == nil {
				c := exec.Command("yay", "-S", "--noconfirm", "chafa")
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				_ = c.Run()
			} else if _, err := exec.LookPath("apt"); err == nil {
				c := exec.Command("sudo", "apt", "install", "-y", "chafa")
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				_ = c.Run()
			}
		}
		if p, err := exec.LookPath("chafa"); err == nil {
			fmt.Printf("✓ chafa is installed at: %s\n", p)
		} else {
			fmt.Println("Notice: chafa could not be auto-installed. Falling back to built-in ANSI album art.")
		}
	}

	fmt.Println("Checking for 'librespot' (Spotify Connect audio daemon)...")
	binPath, err := player.FindLibrespot()
	if err != nil {
		fmt.Println("librespot binary not found. Attempting auto-install...")
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
		} else if _, err := exec.LookPath("yay"); err == nil {
			c := exec.Command("yay", "-S", "--noconfirm", "librespot")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			_ = c.Run()
		}
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
	if autostart {
		if err := utils.EnableAutostart(); err == nil {
			fmt.Println("✓ Auto-start on boot configured (~/.config/autostart/cassette.desktop)")
		} else {
			fmt.Printf("Notice: could not configure autostart: %v\n", err)
		}
	} else {
		_ = utils.DisableAutostart()
	}

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

	fmt.Println("\nLaunching Cassette...")
	time.Sleep(1 * time.Second)
	return true
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
	fmt.Println("              Options: --minimal, --full, --autostart, --no-autostart")
	fmt.Println("  auth        Authenticate with Spotify Web API")
	fmt.Println("  version     Print build metadata")
}

