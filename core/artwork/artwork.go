package artwork

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cassette/core/logger"
	"cassette/core/utils"
)

type Renderer struct {
	client   *http.Client
	cacheDir string
	memCache map[string]string
	mu       sync.RWMutex
}

var (
	defaultRenderer *Renderer
	once            sync.Once
)

func GetRenderer() *Renderer {
	once.Do(func() {
		cacheDir := filepath.Join(utils.SafeGetConfigDir(), "art_cache")
		_ = os.MkdirAll(cacheDir, 0755)

		defaultRenderer = &Renderer{
			client: &http.Client{
				Timeout: 5 * time.Second,
			},
			cacheDir: cacheDir,
			memCache: make(map[string]string),
		}
	})
	return defaultRenderer
}

func (r *Renderer) Render(ctx context.Context, imageURL string, cols, rows int) (string, error) {
	if imageURL == "" || cols <= 0 || rows <= 0 {
		return "", fmt.Errorf("invalid image URL or dimensions")
	}

	key := fmt.Sprintf("%s:%dx%d", imageURL, cols, rows)
	r.mu.RLock()
	if cached, ok := r.memCache[key]; ok {
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	// 1. Download or load from disk
	filePath, err := r.ensureDownloaded(ctx, imageURL)
	if err != nil {
		return "", err
	}

	// 2. If chafa is installed, try chafa
	if _, err := exec.LookPath("chafa"); err == nil {
		out, err := r.renderWithChafa(filePath, cols, rows)
		if err == nil && len(strings.TrimSpace(out)) > 0 {
			r.mu.Lock()
			r.memCache[key] = out
			r.mu.Unlock()
			return out, nil
		}
	}

	// 3. Fallback: Pure Go 24-bit ANSI Half-Block Truecolor Renderer (▀)
	out, err := r.renderNativeHalfBlocks(filePath, cols, rows)
	if err != nil {
		return "", err
	}

	r.mu.Lock()
	r.memCache[key] = out
	r.mu.Unlock()
	return out, nil
}

func (r *Renderer) ensureDownloaded(ctx context.Context, url string) (string, error) {
	h := md5.Sum([]byte(url))
	fileName := hex.EncodeToString(h[:]) + ".jpg"
	targetPath := filepath.Join(r.cacheDir, fileName)

	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return targetPath, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "cassette/1.0 (https://github.com/OreoMuncher45/cassette)")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download artwork (status %d)", resp.StatusCode)
	}

	tmpFile := targetPath + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpFile)
		return "", err
	}
	_ = f.Close()

	_ = os.Rename(tmpFile, targetPath)
	return targetPath, nil
}

func (r *Renderer) renderWithChafa(filePath string, cols, rows int) (string, error) {
	cmd := exec.Command("chafa",
		fmt.Sprintf("--size=%dx%d", cols, rows),
		"--format=symbols",
		"--symbols=half",
		"--dither=none",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func (r *Renderer) renderNativeHalfBlocks(filePath string, cols, rows int) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		logger.Log.Warn().Err(err).Str("file", filePath).Msg("failed to decode image for artwork")
		return "", err
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()
	if imgW == 0 || imgH == 0 {
		return "", fmt.Errorf("empty image bounds")
	}

	// In terminal, 1 cell height = 2 vertical pixels (▀)
	pixelH := rows * 2
	pixelW := cols

	var sb strings.Builder

	for y := 0; y < rows; y++ {
		yTop := (y * 2 * imgH) / pixelH
		yBot := ((y*2 + 1) * imgH) / pixelH

		for x := 0; x < cols; x++ {
			xSrc := (x * imgW) / pixelW

			topCol := img.At(bounds.Min.X+xSrc, bounds.Min.Y+yTop)
			botCol := img.At(bounds.Min.X+xSrc, bounds.Min.Y+yBot)

			r1, g1, b1 := toRGB255(topCol)
			r2, g2, b2 := toRGB255(botCol)

			// \x1b[38;2;r;g;bm = foreground (top pixel ▀)
			// \x1b[48;2;r;g;bm = background (bottom pixel)
			sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", r1, g1, b1, r2, g2, b2))
		}
		sb.WriteString("\x1b[0m")
		if y < rows-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String(), nil
}

func toRGB255(c color.Color) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}
