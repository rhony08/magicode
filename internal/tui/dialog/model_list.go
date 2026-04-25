// Package dialog provides modal dialog components for the TUI.
// This file implements the model selection dialog.
package dialog

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// ModelListDialog shows a searchable list of models
type ModelListDialog struct {
	BaseDialog

	// Available models
	models []ModelItem

	// Filtered models based on search
	filtered []ModelItem

	// Current selection (providerID:modelID)
	currentModel types.ModelKey

	// State reference
	state *types.AppState
}

// ModelItem represents a model in the list
type ModelItem struct {
	ProviderID   string
	ProviderName string
	ModelID      string
	ModelName    string
	Variant      string
	IsCurrent    bool
}

// NewModelListDialog creates a new model list dialog
func NewModelListDialog(theme types.Theme, state *types.AppState) *ModelListDialog {
	d := &ModelListDialog{
		BaseDialog: BaseDialog{
			theme:    theme,
			title:    "Select Model",
			action:   "",
			search:   "",
			selected: 0,
		},
		state:        state,
		currentModel: state.Local.CurrentModel,
	}

	// Build model list from providers
	d.buildModelList()
	d.filtered = d.models
	d.SetItemCount(len(d.filtered))

	return d
}

// buildModelList builds the list of available models from providers
func (d *ModelListDialog) buildModelList() {
	d.models = []ModelItem{}

	for _, provider := range d.state.Sync.Providers {
		for modelID, model := range provider.Models {
			isCurrent := provider.ID == d.currentModel.ProviderID && modelID == d.currentModel.ModelID

			d.models = append(d.models, ModelItem{
				ProviderID:   provider.ID,
				ProviderName: provider.Name,
				ModelID:      modelID,
				ModelName:    model.Name,
				Variant:      "",
				IsCurrent:    isCurrent,
			})

			// Add variants if available
			for variantID, variantName := range model.Variants {
				if variantName != "" && variantID != "" {
					d.models = append(d.models, ModelItem{
						ProviderID:   provider.ID,
						ProviderName: provider.Name,
						ModelID:      modelID,
						ModelName:    model.Name,
						Variant:      variantID,
						IsCurrent:    isCurrent && variantID == d.currentModel.Variant,
					})
				}
			}
		}
	}

	// Sort by provider name then model name
	sort.Slice(d.models, func(i, j int) bool {
		if d.models[i].ProviderName != d.models[j].ProviderName {
			return d.models[i].ProviderName < d.models[j].ProviderName
		}
		if d.models[i].ModelName != d.models[j].ModelName {
			return d.models[i].ModelName < d.models[j].ModelName
		}
		return d.models[i].Variant < d.models[j].Variant
	})

	// Move current model to top
	for i, m := range d.models {
		if m.IsCurrent {
			// Swap to beginning
			d.models[0], d.models[i] = d.models[i], d.models[0]
			break
		}
	}
}

// ID returns the dialog type
func (d *ModelListDialog) ID() types.DialogType {
	return types.DialogModelList
}

// Init initializes the dialog
func (d *ModelListDialog) Init() tea.Cmd {
	return nil
}

// Update handles events
func (d *ModelListDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		handled, cmd := d.HandleKey(msg)
		if handled {
			d.filterModels()
			return d, cmd
		}

		if msg.String() == "enter" && len(d.filtered) > 0 {
			selected := d.filtered[d.Selected()]
			return d, tea.Batch(
				CloseCmd(),
				func() tea.Msg {
					return SelectMsg{
						Type:  types.DialogModelList,
						Index: d.Selected(),
						Data: types.ModelKey{
							ProviderID: selected.ProviderID,
							ModelID:    selected.ModelID,
							Variant:    selected.Variant,
						},
					}
				},
			)
		}

	case tea.WindowSizeMsg:
		d.SetDimensions(msg.Width, msg.Height)
	}

	return d, nil
}

// filterModels filters the model list based on search text
func (d *ModelListDialog) filterModels() {
	if d.search == "" {
		d.filtered = d.models
	} else {
		d.filtered = []ModelItem{}
		searchLower := strings.ToLower(d.search)

		for _, m := range d.models {
			// Search in provider name, model name, model ID
			if strings.Contains(strings.ToLower(m.ProviderName), searchLower) ||
				strings.Contains(strings.ToLower(m.ModelName), searchLower) ||
				strings.Contains(strings.ToLower(m.ModelID), searchLower) ||
				strings.Contains(strings.ToLower(m.Variant), searchLower) {
				d.filtered = append(d.filtered, m)
			}
		}
	}

	d.SetItemCount(len(d.filtered))
	if d.selected >= len(d.filtered) {
		d.selected = 0
	}
}

// View renders the dialog
func (d *ModelListDialog) View() string {
	width := min(d.width-4, 60)
	height := min(d.height-4, 20)

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(d.theme.Primary).
		Bold(true).
		Padding(0, 1)

	searchStyle := lipgloss.NewStyle().
		Foreground(d.theme.Text).
		Background(d.theme.PanelBg).
		Padding(0, 1).
		Width(width - 4)

	itemStyle := lipgloss.NewStyle().
		Foreground(d.theme.Text).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(d.theme.Primary).
		Bold(true).
		Background(d.theme.PanelBg).
		Padding(0, 1)

	mutedStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted)

	currentStyle := lipgloss.NewStyle().
		Foreground(d.theme.Success).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.Border).
		Padding(1, 1)

	// Build content
	var lines []string

	// Title
	lines = append(lines, titleStyle.Render(d.title))

	// Search
	searchText := "🔍 " + d.search
	if d.search == "" {
		searchText = "🔍 Search models..."
	}
	lines = append(lines, searchStyle.Render(searchText))
	lines = append(lines, "")

	// Model list
	visibleHeight := height - 6
	startIdx := d.selected
	if startIdx > len(d.filtered)-visibleHeight {
		startIdx = max(0, len(d.filtered)-visibleHeight)
	}

	// Track current provider for grouping
	currentProvider := ""

	for i := startIdx; i < len(d.filtered) && i < startIdx+visibleHeight; i++ {
		m := d.filtered[i]

		// Show provider header when provider changes
		if m.ProviderName != currentProvider {
			currentProvider = m.ProviderName
			if i > startIdx {
				lines = append(lines, mutedStyle.Render(""))
			}
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("── %s ──", m.ProviderName)))
		}

		// Format model line
		modelName := m.ModelName
		if m.Variant != "" {
			modelName = fmt.Sprintf("%s [%s]", m.ModelName, m.Variant)
		}

		// Truncate if too long
		maxLen := width - 12
		if len(modelName) > maxLen {
			modelName = modelName[:maxLen-3] + "..."
		}

		// Add current indicator
		var line string
		if m.IsCurrent {
			line = fmt.Sprintf("✓ %s", modelName)
		} else {
			line = fmt.Sprintf("  %s", modelName)
		}

		if i == d.selected {
			lines = append(lines, selectedStyle.Render(line))
		} else if m.IsCurrent {
			lines = append(lines, currentStyle.Render(line))
		} else {
			lines = append(lines, itemStyle.Render(line))
		}
	}

	// Empty state
	if len(d.filtered) == 0 {
		emptyText := "No models found"
		if d.search != "" {
			emptyText = fmt.Sprintf("No models matching \"%s\"", d.search)
		}
		lines = append(lines, mutedStyle.Render(emptyText))
	}

	// Footer
	hints := "↑/↓ Navigate  Enter Select  Esc Close"
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render(hints))

	content := strings.Join(lines, "\n")
	return borderStyle.Width(width).Height(height).Render(content)
}
