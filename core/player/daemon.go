package player

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

	"cassette/core/logger"
	"cassette/core/utils"
)

type LocalDaemon struct {
	cmd      *exec.Cmd
	cacheDir string
	logPath  string
	running  bool
	authURL  string
	mu       sync.Mutex
}

var (
	oauthRegex = regexp.MustCompile(`https://accounts\.spotify\.com/authorize\S+`)
)

func FindLibrespot() (string, error) {
	if path, err := exec.LookPath("librespot"); err == nil {
		return path, nil
	}
	for _, fallback := range []string{"/usr/bin/librespot", "/usr/local/bin/librespot", "/home/shehriyar/.local/bin/librespot"} {
		if fi, err := os.Stat(fallback); err == nil && !fi.IsDir() {
			return fallback, nil
		}
	}
	return "", fmt.Errorf("librespot binary not found")
}

func StartLocalDaemon() *LocalDaemon {
	binPath, err := FindLibrespot()
	if err != nil {
		logger.Log.Info().Msg("librespot not found on system; operating in remote control mode only")
		return nil
	}

	configDir := utils.SafeGetConfigDir()
	cacheDir := filepath.Join(configDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	logPath := filepath.Join(configDir, "daemon.log")

	d := &LocalDaemon{
		cacheDir: cacheDir,
		logPath:  logPath,
	}

	d.startProcess(binPath)
	return d
}

func (d *LocalDaemon) startProcess(binPath string) {
	credFile := filepath.Join(d.cacheDir, "credentials.json")
	hasCreds := false
	if fi, err := os.Stat(credFile); err == nil && fi.Size() > 0 {
		hasCreds = true
	}

	args := []string{
		"--name", "cassette",
		"--backend", "pulseaudio",
		"--device-type", "computer",
		"--cache", d.cacheDir,
		"--enable-oauth",
		"--oauth-port", "5588",
	}

	cmd := exec.Command(binPath, args...)

	logFile, err := os.OpenFile(d.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to open daemon.log")
	}

	rStdout, errOut := cmd.StdoutPipe()
	rStderr, errErr := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		logger.Log.Error().Err(err).Msg("failed to start librespot local daemon")
		return
	}

	d.mu.Lock()
	d.cmd = cmd
	d.running = true
	d.mu.Unlock()

	logger.Log.Info().Str("path", binPath).Bool("has_creds", hasCreds).Msg("started local librespot daemon")

	var reader io.Reader
	if errOut == nil && errErr == nil {
		reader = io.MultiReader(rStdout, rStderr)
	} else if errOut == nil {
		reader = rStdout
	} else if errErr == nil {
		reader = rStderr
	}

	if reader != nil {
		go func() {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				if logFile != nil {
					_, _ = fmt.Fprintln(logFile, line)
				}
				if match := oauthRegex.FindString(line); match != "" {
					d.mu.Lock()
					d.authURL = match
					d.mu.Unlock()
					logger.Log.Info().Str("url", match).Msg("detected librespot OAuth URL; opening browser")
					_ = utils.OpenBrowser(match)
				}
				if strings.Contains(line, "Authenticated as") {
					logger.Log.Info().Str("line", line).Msg("librespot successfully authenticated")
				}
				if strings.Contains(line, "INVALID_CREDENTIALS") || strings.Contains(line, "BadCredentials") {
					logger.Log.Warn().Msg("librespot credentials invalid; removing credentials.json and re-authenticating")
					_ = os.Remove(filepath.Join(d.cacheDir, "credentials.json"))
				}
			}
			d.mu.Lock()
			d.running = false
			d.mu.Unlock()
		}()
	}
}

func (d *LocalDaemon) Stop() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cmd != nil && d.cmd.Process != nil && d.running {
		logger.Log.Info().Msg("stopping local librespot daemon")
		_ = d.cmd.Process.Signal(os.Interrupt)

		done := make(chan error, 1)
		go func() {
			done <- d.cmd.Wait()
		}()

		select {
		case <-done:
		case <-time.After(1500 * time.Millisecond):
			_ = d.cmd.Process.Kill()
		}
		d.running = false
	}
}

func (d *LocalDaemon) IsRunning() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

func (d *LocalDaemon) GetAuthURL() string {
	if d == nil {
		return ""
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.authURL
}
