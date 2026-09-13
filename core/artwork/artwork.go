package artwork

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
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
	// 1. High-definition block + sextants in 24-bit Truecolor.
	// Provides 2x3 sub-pixel resolution with 100% solid geometric elements (no ASCII punctuation or letters).
	cmd := exec.Command("chafa",
		"--probe=off",
		"-c", "full",
		"--format=symbols",
		"--symbols=block+sextant",
		fmt.Sprintf("--size=%dx%d", cols, rows),
		filePath,
	)
	out, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return strings.TrimRight(string(out), "\n"), nil
	}

	// 2. Fallback: Universal quadrant + half blocks (2x2 subpixels)
	cmd = exec.Command("chafa",
		"--probe=off",
		"-c", "full",
		"--format=symbols",
		"--symbols=quad+half",
		fmt.Sprintf("--size=%dx%d", cols, rows),
		filePath,
	)
	out, err = cmd.Output()
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

	pixelH := rows * 2
	pixelW := cols

	var sb strings.Builder

	for y := 0; y < rows; y++ {
		yTopStart := (y * 2 * imgH) / pixelH
		yTopEnd := ((y*2 + 1) * imgH) / pixelH
		yBotStart := ((y*2 + 1) * imgH) / pixelH
		yBotEnd := ((y*2 + 2) * imgH) / pixelH

		for x := 0; x < cols; x++ {
			xStart := (x * imgW) / pixelW
			xEnd := ((x + 1) * imgW) / pixelW

			r1, g1, b1 := averageAreaRGB(img, bounds.Min.X+xStart, bounds.Min.X+xEnd, bounds.Min.Y+yTopStart, bounds.Min.Y+yTopEnd)
			r2, g2, b2 := averageAreaRGB(img, bounds.Min.X+xStart, bounds.Min.X+xEnd, bounds.Min.Y+yBotStart, bounds.Min.Y+yBotEnd)

			sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", r1, g1, b1, r2, g2, b2))
		}
		sb.WriteString("\x1b[0m")
		if y < rows-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String(), nil
}

func averageAreaRGB(img image.Image, x0, x1, y0, y1 int) (uint8, uint8, uint8) {
	if x1 <= x0 {
		x1 = x0 + 1
	}
	if y1 <= y0 {
		y1 = y0 + 1
	}

	var sumR, sumG, sumB, count uint64
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sumR += uint64(r >> 8)
			sumG += uint64(g >> 8)
			sumB += uint64(b >> 8)
			count++
		}
	}

	if count == 0 {
		return 0, 0, 0
	}
	return uint8(sumR / count), uint8(sumG / count), uint8(sumB / count)
}
