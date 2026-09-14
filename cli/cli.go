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
			return
		}
	}

	fmt.Printf("✓ Found librespot at: %s\n", binPath)
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
			if autostart {
				if err := utils.EnableAutostart(); err == nil {
					fmt.Println("✓ Auto-start on boot configured (~/.config/autostart/cassette.desktop)")
				} else {
					fmt.Printf("Notice: could not configure autostart: %v\n", err)
				}
			} else {
				_ = utils.DisableAutostart()
			}
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
	fmt.Println("              Options: --minimal, --full, --autostart, --no-autostart")
	fmt.Println("  auth        Authenticate with Spotify Web API")
	fmt.Println("  version     Print build metadata")
}

