package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"nabu/internal/config"
)

// Theme defines an accent color pair.
type Theme struct {
	Name    string
	Primary lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
}

var themes = []Theme{
	{"Green", lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}, lipgloss.AdaptiveColor{Light: "#047857", Dark: "#10B981"}},
	{"Amber", lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}, lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#F59E0B"}},
	{"Teal", lipgloss.AdaptiveColor{Light: "#0D9488", Dark: "#2DD4BF"}, lipgloss.AdaptiveColor{Light: "#0F766E", Dark: "#14B8A6"}},
	{"Blue", lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"}, lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#3B82F6"}},
	{"Purple", lipgloss.AdaptiveColor{Light: "#D946EF", Dark: "#EE6FF8"}, lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}},
}

var currentTheme int

func init() {
	currentTheme = loadThemeIndex()
}

func activeTheme() Theme                      { return themes[currentTheme] }
func accentColor() lipgloss.AdaptiveColor     { return activeTheme().Primary }
func secondaryAccent() lipgloss.AdaptiveColor { return activeTheme().Secondary }

// CycleTheme advances to the next theme, persists it, and returns its name.
func CycleTheme() string {
	currentTheme = (currentTheme + 1) % len(themes)
	saveTheme(activeTheme().Name)
	return activeTheme().Name
}

// ThemeName returns the current theme name.
func ThemeName() string {
	return activeTheme().Name
}

func themePath() string {
	return filepath.Join(config.GetConfigDir(), "theme")
}

func loadThemeIndex() int {
	data, err := os.ReadFile(themePath())
	if err != nil {
		return 0
	}
	name := strings.TrimSpace(string(data))
	for i, t := range themes {
		if t.Name == name {
			return i
		}
	}
	return 0
}

func saveTheme(name string) {
	os.MkdirAll(config.GetConfigDir(), 0700)
	os.WriteFile(themePath(), []byte(name+"\n"), 0644)
}

// Neutral tones (constant, not theme-dependent)
var (
	colorWhite   = lipgloss.AdaptiveColor{Light: "#1A1A2E", Dark: "#FAFAFA"}
	colorSubtle  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}
	colorDim     = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}
	colorSurface = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"}
	colorOverlay = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#111827"}
)

// Semantic colors (constant)
var (
	colorGreen  = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}
	colorBlue   = lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"}
	colorYellow = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}
	colorRed    = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}
	colorOrange = lipgloss.AdaptiveColor{Light: "#EA580C", Dark: "#FB923C"}
	colorCyan   = lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#22D3EE"}
	colorPink   = lipgloss.AdaptiveColor{Light: "#DB2777", Dark: "#F472B6"}
	colorStar   = lipgloss.AdaptiveColor{Light: "#EAB308", Dark: "#FACC15"}
)

// --- Accent-dependent styles (functions, not vars) ---

func panelTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
}

var panelTitleDimStyle = lipgloss.NewStyle().Foreground(colorMuted)

func filterActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(accentColor()).
		Bold(true).
		Padding(0, 1).
		Underline(true)
}

var (
	filterBarStyle = lipgloss.NewStyle().Align(lipgloss.Center)

	filterInactiveStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Padding(0, 1)

	filterSepStyle = lipgloss.NewStyle().Foreground(colorDim)
)

var (
	statusBarStyle  = lipgloss.NewStyle().Foreground(colorSubtle)
	addressBarStyle = lipgloss.NewStyle().Foreground(colorMuted).Align(lipgloss.Right)
)

func modalStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(secondaryAccent()).
		Padding(1, 2)
}

func modalTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
}

var modalHintStyle = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

var headerStyle = lipgloss.NewStyle().Foreground(colorSubtle).Bold(true)

func selectedIndicator() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
}

// Method color helpers
func methodStyle(method string) lipgloss.Style {
	switch method {
	case "GET":
		return lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	case "POST":
		return lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	case "PUT":
		return lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	case "DELETE":
		return lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	case "PATCH":
		return lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
	case "TUNNEL":
		return lipgloss.NewStyle().Foreground(colorMuted)
	default:
		return lipgloss.NewStyle().Foreground(colorWhite)
	}
}

func statusStyle(code int) lipgloss.Style {
	switch {
	case code >= 200 && code < 300:
		return lipgloss.NewStyle().Foreground(colorGreen)
	case code >= 300 && code < 400:
		return lipgloss.NewStyle().Foreground(colorYellow)
	case code >= 400 && code < 500:
		return lipgloss.NewStyle().Foreground(colorOrange)
	case code >= 500:
		return lipgloss.NewStyle().Foreground(colorRed)
	default:
		return lipgloss.NewStyle().Foreground(colorSubtle)
	}
}

// Detail content styles
var (
	detailMethodStyle = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	detailStatusStyle = lipgloss.NewStyle().Foreground(colorCyan)
	detailHeaderStyle = lipgloss.NewStyle().Foreground(colorBlue)
	detailErrorStyle  = lipgloss.NewStyle().Foreground(colorRed)
	detailGrayStyle   = lipgloss.NewStyle().Foreground(colorSubtle)
)

// List item styles (for modals)
var (
	enabledItemStyle  = lipgloss.NewStyle().Foreground(colorGreen)
	disabledItemStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

func listSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
}

func listCursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
}

func buttonActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(accentColor()).
		Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1A1A2E"}).
		Padding(0, 2).
		Bold(true)
}

var buttonInactiveStyle = lipgloss.NewStyle().
	Foreground(colorSubtle).
	Padding(0, 2).
	Border(lipgloss.NormalBorder()).
	BorderForeground(colorDim)

// Badge styles
var (
	pauseBadge = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}).
			Background(colorOrange).
			Bold(true).
			Padding(0, 1)

	mappedLocalBadge = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)

	mappedRemoteBadge = lipgloss.NewStyle().Foreground(colorPink).Bold(true)

	alertBadge = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
)
