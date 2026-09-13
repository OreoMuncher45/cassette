package artwork

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderNativeHalfBlocks(t *testing.T) {
	// Create a 20x20 test image
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.jpg")

	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 128, B: 0, A: 255})
		}
	}

	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create image file: %v", err)
	}
	if err := jpeg.Encode(f, img, nil); err != nil {
		f.Close()
		t.Fatalf("failed to encode jpeg: %v", err)
	}
	f.Close()

	r := &Renderer{}
	out, err := r.renderNativeHalfBlocks(imgPath, 10, 5)
	if err != nil {
		t.Fatalf("renderNativeHalfBlocks failed: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(lines))
	}
	if !strings.Contains(out, "▀") {
		t.Fatal("expected output to contain half-block character ▀")
	}
	if !strings.Contains(out, "\x1b[38;2;") {
		t.Fatal("expected output to contain 24-bit ANSI truecolor codes")
	}
}
