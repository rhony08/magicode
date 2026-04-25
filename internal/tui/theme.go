// Package tui provides theme support for the terminal user interface.
// Themes are based on OpenCode's theme system with seed colors and resolved palettes.
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// ===========================================
// Theme Type Alias
// ===========================================

// Theme is an alias for types.Theme for backward compatibility
type Theme = types.Theme

// ThemeStyles is an alias for types.ThemeStyles for backward compatibility
type ThemeStyles = types.ThemeStyles

// ===========================================
// Desktop Theme Types (for JSON loading)
// ===========================================

// HexColor represents a hex color string (e.g., "#7C3AED")
type HexColor string

// ThemeSeedColors are the base colors used to generate a theme palette
type ThemeSeedColors struct {
	Neutral     HexColor `json:"neutral"`
	Primary     HexColor `json:"primary"`
	Success     HexColor `json:"success"`
	Warning     HexColor `json:"warning"`
	Error       HexColor `json:"error"`
	Info        HexColor `json:"info"`
	Interactive HexColor `json:"interactive"`
	DiffAdd     HexColor `json:"diffAdd"`
	DiffDelete  HexColor `json:"diffDelete"`
}

// ThemePaletteColors are the resolved colors for rendering
type ThemePaletteColors struct {
	Neutral     HexColor  `json:"neutral"`
	Ink         HexColor  `json:"ink"`
	Primary     HexColor  `json:"primary"`
	Success     HexColor  `json:"success"`
	Warning     HexColor  `json:"warning"`
	Error       HexColor  `json:"error"`
	Info        HexColor  `json:"info"`
	Accent      *HexColor `json:"accent,omitempty"`
	Interactive *HexColor `json:"interactive,omitempty"`
	DiffAdd     *HexColor `json:"diffAdd,omitempty"`
	DiffDelete  *HexColor `json:"diffDelete,omitempty"`
}

// ThemeVariant represents a light or dark theme variant
type ThemeVariant struct {
	Seeds     *ThemeSeedColors    `json:"seeds,omitempty"`
	Palette   *ThemePaletteColors `json:"palette,omitempty"`
	Overrides map[string]HexColor `json:"overrides,omitempty"`
}

// DesktopTheme represents a full theme with light and dark variants
type DesktopTheme struct {
	Schema *string      `json:"$schema,omitempty"`
	Name   string       `json:"name"`
	ID     string       `json:"id"`
	Light  ThemeVariant `json:"light"`
	Dark   ThemeVariant `json:"dark"`
}

// ===========================================
// Built-in Themes
// ===========================================

// DefaultThemes contains the priority built-in themes
// Based on OpenCode's theme registry
var DefaultThemes = map[string]Theme{
	"default": {
		Name:         "Default",
		ID:           "default",
		IsDark:       true,
		Primary:      lipgloss.Color("#7C3AED"), // Purple
		Secondary:    lipgloss.Color("#2563EB"), // Blue
		Accent:       lipgloss.Color("#10B981"), // Green
		Error:        lipgloss.Color("#EF4444"),
		Warning:      lipgloss.Color("#F59E0B"),
		Success:      lipgloss.Color("#10B981"),
		Info:         lipgloss.Color("#3B82F6"),
		Text:         lipgloss.Color("#E5E7EB"),
		TextMuted:    lipgloss.Color("#9CA3AF"),
		TextBold:     lipgloss.Color("#FFFFFF"),
		Background:   lipgloss.Color("#1F2937"),
		PanelBg:      lipgloss.Color("#374151"),
		ElementBg:    lipgloss.Color("#4B5563"),
		MenuBg:       lipgloss.Color("#111827"),
		Border:       lipgloss.Color("#374151"),
		BorderActive: lipgloss.Color("#7C3AED"),
		Added:        lipgloss.Color("#22C55E"),
		Removed:      lipgloss.Color("#EF4444"),
		AddedBg:      lipgloss.Color("#14532D"),
		RemovedBg:    lipgloss.Color("#7F1D1D"),
		Heading:      lipgloss.Color("#7C3AED"),
		Link:         lipgloss.Color("#3B82F6"),
		Code:         lipgloss.Color("#FBBF24"),
		CodeBg:       lipgloss.Color("#1F2937"),
		BlockQuote:   lipgloss.Color("#9CA3AF"),
		Comment:      lipgloss.Color("#6B7280"),
		Keyword:      lipgloss.Color("#7C3AED"),
		Function:     lipgloss.Color("#10B981"),
		String:       lipgloss.Color("#FBBF24"),
		Number:       lipgloss.Color("#F59E0B"),
		Spinner:      lipgloss.Color("#7C3AED"),
		Progress:     lipgloss.Color("#7C3AED"),
	},
	"catppuccin": {
		Name:         "Catppuccin",
		ID:           "catppuccin",
		IsDark:       true,
		Primary:      lipgloss.Color("#CBA6F7"), // Mauve
		Secondary:    lipgloss.Color("#89B4FA"), // Blue
		Accent:       lipgloss.Color("#A6E3A1"), // Green
		Error:        lipgloss.Color("#F38BA8"), // Red
		Warning:      lipgloss.Color("#FAB387"), // Orange
		Success:      lipgloss.Color("#A6E3A1"), // Green
		Info:         lipgloss.Color("#89DCEB"), // Teal
		Text:         lipgloss.Color("#CDD6F4"), // Text
		TextMuted:    lipgloss.Color("#A6ADC8"), // Subtext0
		TextBold:     lipgloss.Color("#F5E0DC"), // Flamingo
		Background:   lipgloss.Color("#1E1E2E"), // Base
		PanelBg:      lipgloss.Color("#313244"), // Surface0
		ElementBg:    lipgloss.Color("#45475A"), // Surface1
		MenuBg:       lipgloss.Color("#181825"), // Crust
		Border:       lipgloss.Color("#45475A"), // Surface1
		BorderActive: lipgloss.Color("#CBA6F7"), // Mauve
		Added:        lipgloss.Color("#A6E3A1"),
		Removed:      lipgloss.Color("#F38BA8"),
		AddedBg:      lipgloss.Color("#1E3A1E"),
		RemovedBg:    lipgloss.Color("#3A1E1E"),
		Heading:      lipgloss.Color("#CBA6F7"),
		Link:         lipgloss.Color("#89B4FA"),
		Code:         lipgloss.Color("#FAB387"),
		CodeBg:       lipgloss.Color("#313244"),
		BlockQuote:   lipgloss.Color("#A6ADC8"),
		Comment:      lipgloss.Color("#6C7086"), // Overlay0
		Keyword:      lipgloss.Color("#CBA6F7"),
		Function:     lipgloss.Color("#A6E3A1"),
		String:       lipgloss.Color("#F9E2AF"), // Yellow
		Number:       lipgloss.Color("#FAB387"),
		Spinner:      lipgloss.Color("#CBA6F7"),
		Progress:     lipgloss.Color("#CBA6F7"),
	},
	"dracula": {
		Name:         "Dracula",
		ID:           "dracula",
		IsDark:       true,
		Primary:      lipgloss.Color("#BD93F9"), // Purple
		Secondary:    lipgloss.Color("#8BE9FD"), // Cyan
		Accent:       lipgloss.Color("#50FA7B"), // Green
		Error:        lipgloss.Color("#FF5555"), // Red
		Warning:      lipgloss.Color("#F1FA8C"), // Yellow
		Success:      lipgloss.Color("#50FA7B"),
		Info:         lipgloss.Color("#8BE9FD"),
		Text:         lipgloss.Color("#F8F8F2"),
		TextMuted:    lipgloss.Color("#6272A4"),
		TextBold:     lipgloss.Color("#FF79C6"), // Pink
		Background:   lipgloss.Color("#282A36"),
		PanelBg:      lipgloss.Color("#44475A"),
		ElementBg:    lipgloss.Color("#6272A4"),
		MenuBg:       lipgloss.Color("#21222C"),
		Border:       lipgloss.Color("#44475A"),
		BorderActive: lipgloss.Color("#BD93F9"),
		Added:        lipgloss.Color("#50FA7B"),
		Removed:      lipgloss.Color("#FF5555"),
		AddedBg:      lipgloss.Color("#2A4A2A"),
		RemovedBg:    lipgloss.Color("#4A2A2A"),
		Heading:      lipgloss.Color("#BD93F9"),
		Link:         lipgloss.Color("#8BE9FD"),
		Code:         lipgloss.Color("#F1FA8C"),
		CodeBg:       lipgloss.Color("#44475A"),
		BlockQuote:   lipgloss.Color("#6272A4"),
		Comment:      lipgloss.Color("#6272A4"),
		Keyword:      lipgloss.Color("#FF79C6"),
		Function:     lipgloss.Color("#50FA7B"),
		String:       lipgloss.Color("#F1FA8C"),
		Number:       lipgloss.Color("#BD93F9"),
		Spinner:      lipgloss.Color("#BD93F9"),
		Progress:     lipgloss.Color("#BD93F9"),
	},
	"tokyonight": {
		Name:         "Tokyo Night",
		ID:           "tokyonight",
		IsDark:       true,
		Primary:      lipgloss.Color("#7AA2F7"), // Blue
		Secondary:    lipgloss.Color("#2AC3DE"), // Cyan
		Accent:       lipgloss.Color("#FF9E64"), // Orange
		Error:        lipgloss.Color("#F7768E"), // Red
		Warning:      lipgloss.Color("#E0AF68"), // Yellow
		Success:      lipgloss.Color("#9ECE6A"), // Green
		Info:         lipgloss.Color("#2AC3DE"),
		Text:         lipgloss.Color("#C0CAF5"),
		TextMuted:    lipgloss.Color("#565F89"),
		TextBold:     lipgloss.Color("#C0CAF5"),
		Background:   lipgloss.Color("#1A1B26"),
		PanelBg:      lipgloss.Color("#24283B"),
		ElementBg:    lipgloss.Color("#414868"),
		MenuBg:       lipgloss.Color("#16161E"),
		Border:       lipgloss.Color("#3B4261"),
		BorderActive: lipgloss.Color("#7AA2F7"),
		Added:        lipgloss.Color("#9ECE6A"),
		Removed:      lipgloss.Color("#F7768E"),
		AddedBg:      lipgloss.Color("#2A3A2A"),
		RemovedBg:    lipgloss.Color("#3A2A2A"),
		Heading:      lipgloss.Color("#7AA2F7"),
		Link:         lipgloss.Color("#2AC3DE"),
		Code:         lipgloss.Color("#BB9AF7"), // Magenta
		CodeBg:       lipgloss.Color("#24283B"),
		BlockQuote:   lipgloss.Color("#565F89"),
		Comment:      lipgloss.Color("#565F89"),
		Keyword:      lipgloss.Color("#BB9AF7"),
		Function:     lipgloss.Color("#9ECE6A"),
		String:       lipgloss.Color("#E0AF68"),
		Number:       lipgloss.Color("#FF9E64"),
		Spinner:      lipgloss.Color("#7AA2F7"),
		Progress:     lipgloss.Color("#7AA2F7"),
	},
	"nord": {
		Name:         "Nord",
		ID:           "nord",
		IsDark:       true,
		Primary:      lipgloss.Color("#88C0D0"), // Frost (cyan)
		Secondary:    lipgloss.Color("#81A1C1"), // Frost (blue)
		Accent:       lipgloss.Color("#A3BE8C"), // Aurora (green)
		Error:        lipgloss.Color("#BF616A"), // Aurora (red)
		Warning:      lipgloss.Color("#EBCB8B"), // Aurora (yellow)
		Success:      lipgloss.Color("#A3BE8C"),
		Info:         lipgloss.Color("#88C0D0"),
		Text:         lipgloss.Color("#ECEFF4"), // Snowstorm
		TextMuted:    lipgloss.Color("#D8DEE9"),
		TextBold:     lipgloss.Color("#ECEFF4"),
		Background:   lipgloss.Color("#2E3440"), // Polar Night
		PanelBg:      lipgloss.Color("#3B4252"),
		ElementBg:    lipgloss.Color("#4C566A"),
		MenuBg:       lipgloss.Color("#242933"),
		Border:       lipgloss.Color("#4C566A"),
		BorderActive: lipgloss.Color("#88C0D0"),
		Added:        lipgloss.Color("#A3BE8C"),
		Removed:      lipgloss.Color("#BF616A"),
		AddedBg:      lipgloss.Color("#2A3A2A"),
		RemovedBg:    lipgloss.Color("#3A2A2A"),
		Heading:      lipgloss.Color("#88C0D0"),
		Link:         lipgloss.Color("#81A1C1"),
		Code:         lipgloss.Color("#EBCB8B"),
		CodeBg:       lipgloss.Color("#3B4252"),
		BlockQuote:   lipgloss.Color("#D8DEE9"),
		Comment:      lipgloss.Color("#616E88"),
		Keyword:      lipgloss.Color("#81A1C1"),
		Function:     lipgloss.Color("#A3BE8C"),
		String:       lipgloss.Color("#EBCB8B"),
		Number:       lipgloss.Color("#B48EAD"), // Aurora (purple)
		Spinner:      lipgloss.Color("#88C0D0"),
		Progress:     lipgloss.Color("#88C0D0"),
	},
	"gruvbox": {
		Name:         "Gruvbox",
		ID:           "gruvbox",
		IsDark:       true,
		Primary:      lipgloss.Color("#D3869B"), // Purple
		Secondary:    lipgloss.Color("#83A598"), // Blue
		Accent:       lipgloss.Color("#B8BB26"), // Green
		Error:        lipgloss.Color("#FB4934"), // Red
		Warning:      lipgloss.Color("#FABD2F"), // Yellow
		Success:      lipgloss.Color("#B8BB26"),
		Info:         lipgloss.Color("#83A598"),
		Text:         lipgloss.Color("#EBDBB2"), // fg
		TextMuted:    lipgloss.Color("#A89984"), // gray
		TextBold:     lipgloss.Color("#FBF1C7"), // bg0 (bright)
		Background:   lipgloss.Color("#282828"), // bg
		PanelBg:      lipgloss.Color("#3C3836"), // bg1
		ElementBg:    lipgloss.Color("#504945"), // bg2
		MenuBg:       lipgloss.Color("#1D2021"), // bg0 (dark)
		Border:       lipgloss.Color("#504945"),
		BorderActive: lipgloss.Color("#D3869B"),
		Added:        lipgloss.Color("#B8BB26"),
		Removed:      lipgloss.Color("#FB4934"),
		AddedBg:      lipgloss.Color("#2A3A2A"),
		RemovedBg:    lipgloss.Color("#3A2A2A"),
		Heading:      lipgloss.Color("#D3869B"),
		Link:         lipgloss.Color("#83A598"),
		Code:         lipgloss.Color("#FE8019"), // Orange
		CodeBg:       lipgloss.Color("#3C3836"),
		BlockQuote:   lipgloss.Color("#A89984"),
		Comment:      lipgloss.Color("#928374"),
		Keyword:      lipgloss.Color("#FB4934"),
		Function:     lipgloss.Color("#B8BB26"),
		String:       lipgloss.Color("#FABD2F"),
		Number:       lipgloss.Color("#D3869B"),
		Spinner:      lipgloss.Color("#FABD2F"),
		Progress:     lipgloss.Color("#FABD2F"),
	},
	"onehalf": {
		Name:         "One Half",
		ID:           "onehalf",
		IsDark:       true,
		Primary:      lipgloss.Color("#C678DD"), // Purple
		Secondary:    lipgloss.Color("#61AFEF"), // Blue
		Accent:       lipgloss.Color("#98C379"), // Green
		Error:        lipgloss.Color("#E06C75"), // Red
		Warning:      lipgloss.Color("#E5C07B"), // Yellow/Orange
		Success:      lipgloss.Color("#98C379"),
		Info:         lipgloss.Color("#56B6C2"), // Cyan
		Text:         lipgloss.Color("#DCDFE4"),
		TextMuted:    lipgloss.Color("#5C6370"),
		TextBold:     lipgloss.Color("#DCDFE4"),
		Background:   lipgloss.Color("#282C34"),
		PanelBg:      lipgloss.Color("#2C323C"),
		ElementBg:    lipgloss.Color("#3E4451"),
		MenuBg:       lipgloss.Color("#21252B"),
		Border:       lipgloss.Color("#3E4451"),
		BorderActive: lipgloss.Color("#61AFEF"),
		Added:        lipgloss.Color("#98C379"),
		Removed:      lipgloss.Color("#E06C75"),
		AddedBg:      lipgloss.Color("#2A3A2A"),
		RemovedBg:    lipgloss.Color("#3A2A2A"),
		Heading:      lipgloss.Color("#C678DD"),
		Link:         lipgloss.Color("#61AFEF"),
		Code:         lipgloss.Color("#E5C07B"),
		CodeBg:       lipgloss.Color("#2C323C"),
		BlockQuote:   lipgloss.Color("#5C6370"),
		Comment:      lipgloss.Color("#5C6370"),
		Keyword:      lipgloss.Color("#C678DD"),
		Function:     lipgloss.Color("#61AFEF"),
		String:       lipgloss.Color("#98C379"),
		Number:       lipgloss.Color("#D19A66"),
		Spinner:      lipgloss.Color("#61AFEF"),
		Progress:     lipgloss.Color("#61AFEF"),
	},
	"solarized": {
		Name:         "Solarized",
		ID:           "solarized",
		IsDark:       true,
		Primary:      lipgloss.Color("#268BD2"), // Blue
		Secondary:    lipgloss.Color("#2AA198"), // Cyan
		Accent:       lipgloss.Color("#859900"), // Green
		Error:        lipgloss.Color("#DC322F"), // Red
		Warning:      lipgloss.Color("#B58900"), // Yellow
		Success:      lipgloss.Color("#859900"),
		Info:         lipgloss.Color("#268BD2"),
		Text:         lipgloss.Color("#839496"), // base0
		TextMuted:    lipgloss.Color("#657B83"), // base01
		TextBold:     lipgloss.Color("#EEE8D5"), // base2 (light)
		Background:   lipgloss.Color("#002B36"), // base03
		PanelBg:      lipgloss.Color("#073642"), // base02
		ElementBg:    lipgloss.Color("#094559"),
		MenuBg:       lipgloss.Color("#001E26"),
		Border:       lipgloss.Color("#094559"),
		BorderActive: lipgloss.Color("#268BD2"),
		Added:        lipgloss.Color("#859900"),
		Removed:      lipgloss.Color("#DC322F"),
		AddedBg:      lipgloss.Color("#2A3A2A"),
		RemovedBg:    lipgloss.Color("#3A2A2A"),
		Heading:      lipgloss.Color("#268BD2"),
		Link:         lipgloss.Color("#2AA198"),
		Code:         lipgloss.Color("#B58900"),
		CodeBg:       lipgloss.Color("#073642"),
		BlockQuote:   lipgloss.Color("#657B83"),
		Comment:      lipgloss.Color("#586E75"), // base01 (muted)
		Keyword:      lipgloss.Color("#CB4B16"), // Orange
		Function:     lipgloss.Color("#268BD2"),
		String:       lipgloss.Color("#2AA198"),
		Number:       lipgloss.Color("#D33682"), // Magenta
		Spinner:      lipgloss.Color("#268BD2"),
		Progress:     lipgloss.Color("#268BD2"),
	},
}

// ===========================================
// Theme Registry
// ===========================================

// ThemeRegistry manages theme loading and lookup
type ThemeRegistry struct {
	themes    map[string]Theme
	customDir string // Directory for custom themes
}

// NewThemeRegistry creates a new theme registry with built-in themes
func NewThemeRegistry() *ThemeRegistry {
	return &ThemeRegistry{
		themes: DefaultThemes,
	}
}

// SetCustomDir sets the directory for custom theme files
func (r *ThemeRegistry) SetCustomDir(dir string) {
	r.customDir = dir
}

// Get retrieves a theme by ID
func (r *ThemeRegistry) Get(id string) (Theme, bool) {
	theme, ok := r.themes[id]
	return theme, ok
}

// List returns all available theme IDs
func (r *ThemeRegistry) List() []string {
	ids := make([]string, 0, len(r.themes))
	for id := range r.themes {
		ids = append(ids, id)
	}
	return ids
}

// ListWithNames returns all themes with their names
func (r *ThemeRegistry) ListWithNames() []struct{ ID, Name string } {
	result := make([]struct{ ID, Name string }, 0, len(r.themes))
	for id, theme := range r.themes {
		result = append(result, struct{ ID, Name string }{ID: id, Name: theme.Name})
	}
	return result
}

// LoadCustom loads custom themes from JSON files in the custom directory
func (r *ThemeRegistry) LoadCustom() error {
	if r.customDir == "" {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(r.customDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to scan custom themes: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var dt DesktopTheme
		if err := json.Unmarshal(data, &dt); err != nil {
			continue // Skip invalid JSON
		}

		if dt.ID == "" {
			dt.ID = strings.TrimSuffix(filepath.Base(file), ".json")
		}

		// Convert DesktopTheme to Theme (use dark variant by default)
		theme := convertDesktopTheme(dt, true)
		r.themes[dt.ID] = theme
	}

	return nil
}

// convertDesktopTheme converts a DesktopTheme to a Theme for rendering
func convertDesktopTheme(dt DesktopTheme, useDark bool) Theme {
	variant := dt.Dark
	if !useDark {
		variant = dt.Light
	}

	theme := Theme{
		Name:   dt.Name,
		ID:     dt.ID,
		IsDark: useDark,
	}

	// Apply palette colors if present
	if variant.Palette != nil {
		p := variant.Palette
		theme.Primary = lipgloss.Color(p.Primary)
		theme.Error = lipgloss.Color(p.Error)
		theme.Warning = lipgloss.Color(p.Warning)
		theme.Success = lipgloss.Color(p.Success)
		theme.Info = lipgloss.Color(p.Info)

		if p.Accent != nil {
			theme.Accent = lipgloss.Color(*p.Accent)
		}
	}

	// Apply overrides
	for key, color := range variant.Overrides {
		switch key {
		case "text":
			theme.Text = lipgloss.Color(color)
		case "textMuted":
			theme.TextMuted = lipgloss.Color(color)
		case "background":
			theme.Background = lipgloss.Color(color)
		case "border":
			theme.Border = lipgloss.Color(color)
		}
	}

	return theme
}

// ===========================================
// Theme Helpers
// ===========================================

// CurrentTheme returns the current theme from app state
func CurrentTheme(state *AppState) Theme {
	registry := NewThemeRegistry()
	theme, ok := registry.Get(state.KV.Theme)
	if !ok {
		theme, _ = registry.Get("default")
	}
	return theme
}

// ApplyTheme returns lipgloss styles for a given theme
func ApplyTheme(theme Theme) ThemeStyles {
	return ThemeStyles{
		Title: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1),

		Prompt: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),

		UserMessage: lipgloss.NewStyle().
			Foreground(theme.Text).
			Padding(0, 1, 0, 2),

		AssistantMessage: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Padding(0, 1, 0, 2),

		SystemMessage: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Padding(0, 1),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(theme.Success),

		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning),

		Status: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Background(theme.Background).
			Padding(0, 1),

		Text: lipgloss.NewStyle().
			Foreground(theme.Text),

		TextMuted: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		Bold: lipgloss.NewStyle().
			Bold(true),

		Border: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border),

		BorderActive: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.BorderActive),

		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border).
			Padding(1, 2).
			Background(theme.PanelBg),

		ToolUse: lipgloss.NewStyle().
			Foreground(theme.Info).
			Bold(true).
			Padding(0, 1),

		ToolResult: lipgloss.NewStyle().
			Foreground(theme.Success).
			Padding(0, 1),

		Thinking: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Padding(0, 1),

		Heading: lipgloss.NewStyle().
			Foreground(theme.Heading).
			Bold(true),

		Code: lipgloss.NewStyle().
			Foreground(theme.Code).
			Background(theme.CodeBg),

		Link: lipgloss.NewStyle().
			Foreground(theme.Link),

		Added: lipgloss.NewStyle().
			Foreground(theme.Added).
			Background(theme.AddedBg),

		Removed: lipgloss.NewStyle().
			Foreground(theme.Removed).
			Background(theme.RemovedBg),
	}
}

// ===========================================
// Global Theme Registry Instance
// ===========================================

// globalThemeRegistry is the default theme registry
var globalThemeRegistry = NewThemeRegistry()

// GetTheme retrieves a theme by ID from the global registry
func GetTheme(id string) Theme {
	theme, ok := globalThemeRegistry.Get(id)
	if !ok {
		theme, _ = globalThemeRegistry.Get("default")
	}
	return theme
}

// GetAllThemes returns all available themes from the global registry
func GetAllThemes() map[string]Theme {
	return globalThemeRegistry.themes
}

// ListThemes returns all themes as a slice
func (r *ThemeRegistry) ListThemes() []Theme {
	themes := make([]Theme, 0, len(r.themes))
	for _, theme := range r.themes {
		themes = append(themes, theme)
	}
	return themes
}

// ThemeNames returns a list of theme names
func ThemeNames() []string {
	return globalThemeRegistry.List()
}
