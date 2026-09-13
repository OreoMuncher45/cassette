package theme

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"sync"

	"cassette/core/logger"
	"cassette/core/utils"
	"charm.land/lipgloss/v2"
)

type SchemeID string

const (
	SchemeAlbumArt   SchemeID = "album_art"
	SchemeRetroCyan  SchemeID = "retro_cyan"
	SchemeCyberpunk  SchemeID = "cyberpunk"
	SchemeSynthwave  SchemeID = "synthwave"
	SchemeMatrix     SchemeID = "matrix"
	SchemeDracula    SchemeID = "dracula"
	SchemeNord       SchemeID = "nord"
	SchemeAmber      SchemeID = "amber"
	SchemeTokyoNight SchemeID = "tokyo_night"
	SchemeSolarized  SchemeID = "solarized"
	SchemePastel     SchemeID = "pastel"
	SchemeCustom     SchemeID = "custom"
)

type EffectID string

const (
	EffectStatic    EffectID = "static"
	EffectBreathing EffectID = "breathing"
	EffectRainbow   EffectID = "rainbow"
	EffectPulse     EffectID = "pulse"
)

type SpeedID string

const (
	SpeedSlow   SpeedID = "slow"
	SpeedMedium SpeedID = "medium"
	SpeedFast   SpeedID = "fast"
	SpeedUltra  SpeedID = "ultra"
)

type Settings struct {
	Scheme          SchemeID `json:"scheme"`
	Effect          EffectID `json:"effect"`
	Speed           SpeedID  `json:"speed"`
	CustomPrimary   string   `json:"custom_primary,omitempty"`
	CustomSecondary string   `json:"custom_secondary,omitempty"`
}

type Manager struct {
	mu              sync.RWMutex
	settings        Settings
	tickCount       uint64
	artPrimary      string // Hex e.g. "#00f5d4"
	artSecondary    string // Hex e.g. "#ff79c6"
	customPrimary   string
	customSecondary string
	configFile   string
}

var (
	instance *Manager
	once     sync.Once
)

func Get() *Manager {
	once.Do(func() {
		cfgDir := utils.SafeGetConfigDir()
		configFile := filepath.Join(cfgDir, "settings.json")
		instance = &Manager{
			settings: Settings{
				Scheme: SchemeRetroCyan,
				Effect: EffectStatic,
				Speed:  SpeedMedium,
			},
			artPrimary:   "#00f5d4",
			artSecondary: "#ffd166",
			configFile:   configFile,
		}
		instance.LoadSettings()
	})
	return instance
}

func (m *Manager) GetSettings() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

func (m *Manager) SetScheme(s SchemeID) {
	m.mu.Lock()
	m.settings.Scheme = s
	m.mu.Unlock()
	_ = m.SaveSettings()
}

func (m *Manager) SetEffect(e EffectID) {
	m.mu.Lock()
	m.settings.Effect = e
	m.mu.Unlock()
	_ = m.SaveSettings()
}

func (m *Manager) SetSpeed(s SpeedID) {
	m.mu.Lock()
	m.settings.Speed = s
	m.mu.Unlock()
	_ = m.SaveSettings()
}

func (m *Manager) Tick() {
	m.mu.Lock()
	m.tickCount++
	m.mu.Unlock()
}

func (m *Manager) LoadSettings() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err == nil {
		if s.Scheme != "" {
			m.settings.Scheme = s.Scheme
		}
		if s.Effect != "" {
			m.settings.Effect = s.Effect
		}
		if s.Speed != "" {
			m.settings.Speed = s.Speed
		}
	}
}

func (m *Manager) SaveSettings() error {
	m.mu.RLock()
	s := m.settings
	m.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configFile, data, 0644)
}

// ExtractPaletteFromImage extracts the most vibrant accent color from an album art image.
func (m *Manager) ExtractPaletteFromImage(filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return
	}

	// Sample pixels in a grid to find the most colorful / vibrant pixel
	var bestHex string
	var bestSat float64 = -1.0
	var secHex string
	var secSat float64 = -1.0

	stepX := max(1, w/24)
	stepY := max(1, h/24)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			h, s, v := rgbToHsv(r8, g8, b8)
			_ = h
			// Ignore pure black, pure white, and dull grays
			if v < 0.25 || v > 0.95 || s < 0.25 {
				continue
			}

			score := s * 1.5 + v*0.5
			if score > bestSat {
				secSat = bestSat
				secHex = bestHex
				bestSat = score
				bestHex = fmt.Sprintf("#%02x%02x%02x", r8, g8, b8)
			} else if score > secSat {
				secSat = score
				secHex = fmt.Sprintf("#%02x%02x%02x", r8, g8, b8)
			}
		}
	}

	if bestHex == "" {
		bestHex = "#00f5d4"
	}
	if secHex == "" || secHex == bestHex {
		secHex = "#ffd166"
	}

	m.mu.Lock()
	m.artPrimary = bestHex
	m.artSecondary = secHex
	m.mu.Unlock()

	logger.Log.Debug().Str("primary", bestHex).Str("secondary", secHex).Msg("extracted album art palette")
}

func (m *Manager) PrimaryColor() color.Color {
	return lipgloss.Color(m.PrimaryHex())
}

func (m *Manager) PrimaryHex() string {
	m.mu.RLock()
	scheme := m.settings.Scheme
	effect := m.settings.Effect
	speed := m.settings.Speed
	tick := m.tickCount
	artPrim := m.artPrimary
	custPrim := m.settings.CustomPrimary
	m.mu.RUnlock()

	if effect == EffectRainbow {
		return rainbowHex(tick, speed, 0)
	}

	baseHex := getBasePrimary(scheme, artPrim, custPrim)

	if effect == EffectBreathing {
		return breathHex(baseHex, tick, speed)
	}
	if effect == EffectPulse {
		return pulseHex(baseHex, tick, speed)
	}

	return baseHex
}

func (m *Manager) SecondaryColor() color.Color {
	return lipgloss.Color(m.SecondaryHex())
}

func (m *Manager) SecondaryHex() string {
	m.mu.RLock()
	scheme := m.settings.Scheme
	effect := m.settings.Effect
	speed := m.settings.Speed
	tick := m.tickCount
	artSec := m.artSecondary
	custSec := m.settings.CustomSecondary
	m.mu.RUnlock()

	if effect == EffectRainbow {
		return rainbowHex(tick, speed, 120)
	}

	baseHex := getBaseSecondary(scheme, artSec, custSec)

	if effect == EffectBreathing {
		return breathHex(baseHex, tick+10, speed)
	}
	if effect == EffectPulse {
		return pulseHex(baseHex, tick+12, speed)
	}

	return baseHex
}

func (m *Manager) BorderColor() color.Color {
	return lipgloss.Color(m.BorderHex())
}

func (m *Manager) BorderHex() string {
	m.mu.RLock()
	effect := m.settings.Effect
	speed := m.settings.Speed
	tick := m.tickCount
	m.mu.RUnlock()

	if effect == EffectRainbow {
		return rainbowHex(tick, speed, 240)
	}
	if effect == EffectBreathing {
		return breathDimHex(m.PrimaryHex(), tick, speed)
	}
	if effect == EffectPulse {
		return pulseDimHex(m.PrimaryHex(), tick, speed)
	}
	return "242"
}

func (m *Manager) DimBorderColor() color.Color {
	return lipgloss.Color("238")
}

func (m *Manager) ActiveLyricsColor() color.Color {
	return m.PrimaryColor()
}

func (m *Manager) ReelColor() color.Color {
	return m.PrimaryColor()
}

func (m *Manager) HeaderColor() color.Color {
	return m.SecondaryColor()
}

func getBasePrimary(scheme SchemeID, artHex, custHex string) string {
	switch scheme {
	case SchemeAlbumArt:
		if artHex != "" {
			return artHex
		}
		return "#00f5d4"
	case SchemeRetroCyan:
		return "#00f5d4"
	case SchemeCyberpunk:
		return "#ff007f"
	case SchemeSynthwave:
		return "#bd00ff"
	case SchemeMatrix:
		return "#00ff41"
	case SchemeDracula:
		return "#bd93f9"
	case SchemeNord:
		return "#88c0d0"
	case SchemeAmber:
		return "#ffb000"
	case SchemeTokyoNight:
		return "#7aa2f7"
	case SchemeSolarized:
		return "#2aa198"
	case SchemePastel:
		return "#ff9a9e"
	case SchemeCustom:
		if custHex != "" {
			return custHex
		}
		return "#00f5d4"
	default:
		return "#00f5d4"
	}
}

func getBaseSecondary(scheme SchemeID, artHex, custHex string) string {
	switch scheme {
	case SchemeAlbumArt:
		if artHex != "" {
			return artHex
		}
		return "#ffd166"
	case SchemeRetroCyan:
		return "#ffd166"
	case SchemeCyberpunk:
		return "#00f0ff"
	case SchemeSynthwave:
		return "#ff5e00"
	case SchemeMatrix:
		return "#39ff14"
	case SchemeDracula:
		return "#ff79c6"
	case SchemeNord:
		return "#81a1c1"
	case SchemeAmber:
		return "#ffcf56"
	case SchemeTokyoNight:
		return "#bb9af7"
	case SchemeSolarized:
		return "#b58900"
	case SchemePastel:
		return "#a1c4fd"
	case SchemeCustom:
		if custHex != "" {
			return custHex
		}
		return "#ffd166"
	default:
		return "#ffd166"
	}
}

func rainbowHex(tick uint64, speed SpeedID, offsetDegrees float64) string {
	multiplier := 4.0
	switch speed {
	case SpeedSlow:
		multiplier = 2.0
	case SpeedFast:
		multiplier = 8.0
	case SpeedUltra:
		multiplier = 16.0
	}

	hue := math.Mod(float64(tick)*multiplier+offsetDegrees, 360.0)
	r, g, b := hsvToRgb(hue, 0.9, 0.95)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func breathHex(hex string, tick uint64, speed SpeedID) string {
	var r, g, b uint8
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	h, s, v := rgbToHsv(r, g, b)

	multiplier := 0.12
	switch speed {
	case SpeedSlow:
		multiplier = 0.06
	case SpeedFast:
		multiplier = 0.24
	case SpeedUltra:
		multiplier = 0.48
	}

	// Sine wave breathing between 0.45 and 1.0
	cycle := (math.Sin(float64(tick)*multiplier) + 1.0) / 2.0 // 0.0 to 1.0
	v = 0.45 + cycle*0.55

	rOut, gOut, bOut := hsvToRgb(h, s, v)
	return fmt.Sprintf("#%02x%02x%02x", rOut, gOut, bOut)
}

func pulseHex(hex string, tick uint64, speed SpeedID) string {
	var r, g, b uint8
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	h, s, v := rgbToHsv(r, g, b)

	multiplier := 0.18
	switch speed {
	case SpeedSlow:
		multiplier = 0.09
	case SpeedFast:
		multiplier = 0.36
	case SpeedUltra:
		multiplier = 0.72
	}

	beat := math.Abs(math.Sin(float64(tick) * multiplier))
	v = 0.35 + math.Pow(beat, 2)*0.65

	rOut, gOut, bOut := hsvToRgb(h, s, v)
	return fmt.Sprintf("#%02x%02x%02x", rOut, gOut, bOut)
}

func pulseDimHex(hex string, tick uint64, speed SpeedID) string {
	var r, g, b uint8
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	h, s, _ := rgbToHsv(r, g, b)

	multiplier := 0.18
	switch speed {
	case SpeedSlow:
		multiplier = 0.09
	case SpeedFast:
		multiplier = 0.36
	case SpeedUltra:
		multiplier = 0.72
	}

	beat := math.Abs(math.Sin(float64(tick) * multiplier))
	v := 0.20 + math.Pow(beat, 2)*0.40

	rOut, gOut, bOut := hsvToRgb(h, s*0.6, v)
	return fmt.Sprintf("#%02x%02x%02x", rOut, gOut, bOut)
}

func breathDimHex(hex string, tick uint64, speed SpeedID) string {
	var r, g, b uint8
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	h, s, _ := rgbToHsv(r, g, b)

	multiplier := 0.12
	switch speed {
	case SpeedSlow:
		multiplier = 0.06
	case SpeedFast:
		multiplier = 0.24
	}

	cycle := (math.Sin(float64(tick)*multiplier) + 1.0) / 2.0
	v := 0.25 + cycle*0.35 // subtle glow on borders

	rOut, gOut, bOut := hsvToRgb(h, s*0.6, v)
	return fmt.Sprintf("#%02x%02x%02x", rOut, gOut, bOut)
}

func rgbToHsv(r, g, b uint8) (h, s, v float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	maxC := math.Max(rf, math.Max(gf, bf))
	minC := math.Min(rf, math.Min(gf, bf))
	delta := maxC - minC

	v = maxC
	if maxC == 0 {
		s = 0
	} else {
		s = delta / maxC
	}

	if delta == 0 {
		h = 0
	} else if maxC == rf {
		h = math.Mod((gf-bf)/delta, 6.0) * 60.0
	} else if maxC == gf {
		h = ((bf-rf)/delta + 2.0) * 60.0
	} else {
		h = ((rf-gf)/delta + 4.0) * 60.0
	}

	if h < 0 {
		h += 360.0
	}
	return
}

func hsvToRgb(h, s, v float64) (r, g, b uint8) {
	c := v * s
	x := c * (1.0 - math.Abs(math.Mod(h/60.0, 2.0)-1.0))
	m := v - c

	var rf, gf, bf float64
	switch {
	case h >= 0 && h < 60:
		rf, gf, bf = c, x, 0
	case h >= 60 && h < 120:
		rf, gf, bf = x, c, 0
	case h >= 120 && h < 180:
		rf, gf, bf = 0, c, x
	case h >= 180 && h < 240:
		rf, gf, bf = 0, x, c
	case h >= 240 && h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}

	r = uint8(math.Round((rf + m) * 255.0))
	g = uint8(math.Round((gf + m) * 255.0))
	b = uint8(math.Round((bf + m) * 255.0))
	return
}
