package repl

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas/dofushold/internal/ui"
	"golang.org/x/term"
)

// selectModel = un menu Bubble Tea (horizontal ou vertical).
type selectModel struct {
	title      string
	status     string // ligne optionnelle au-dessus (ex: barres HP)
	items      []string
	cursor     int
	width      int // largeur terminal (pour la ligne pleine largeur)
	horizontal bool
	chosen     int
	cancelled  bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			m.chosen = m.cursor
			return m, tea.Quit

		case "left", "h":
			if m.horizontal {
				m.cursor = (m.cursor - 1 + len(m.items)) % len(m.items)
			}
		case "right", "l":
			if m.horizontal {
				m.cursor = (m.cursor + 1) % len(m.items)
			}

		case "up", "k":
			if !m.horizontal && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if !m.horizontal && m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	if len(m.items) == 0 {
		return ""
	}

	border := ui.FullWidthLine(m.width)
	var b strings.Builder
	b.WriteString(border)
	b.WriteString("\n")

	if m.status != "" {
		b.WriteString(m.status)
		b.WriteString("\n")
	}

	hint := "(↑↓ Entrée, Esc annule)"
	if m.horizontal {
		hint = "(← → Entrée, Esc annule)"
	}
	b.WriteString(ui.Accent(m.title + "  " + hint))
	b.WriteString("\n")

	if m.horizontal {
		b.WriteString(viewHorizontal(m.items, m.cursor))
	} else {
		b.WriteString(viewVertical(m.items, m.cursor))
	}
	b.WriteString("\n")
	b.WriteString(border)
	return b.String()
}

func viewHorizontal(items []string, cursor int) string {
	parts := make([]string, len(items))
	for i, label := range items {
		if i == cursor {
			parts[i] = ui.Accent(ui.Bold("▸ " + label))
		} else {
			parts[i] = "  " + label
		}
	}
	return strings.Join(parts, " | ")
}

func viewVertical(items []string, cursor int) string {
	var b strings.Builder
	for i, label := range items {
		if i == cursor {
			b.WriteString(ui.Accent(ui.Bold("▸ " + label)))
		} else {
			b.WriteString("  " + label)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// runSelect lance Bubble Tea et renvoie l'index choisi.
// ok=false si Escape / Ctrl+C / erreur.
func runSelect(title string, items []string, horizontal bool, status string) (int, bool) {
	if len(items) == 0 {
		return -1, false
	}
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	m := selectModel{
		title:      title,
		status:     status,
		items:      items,
		width:      width,
		horizontal: horizontal,
		chosen:     -1,
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Println(ui.Bad(err.Error()))
		return -1, false
	}
	res, ok := final.(selectModel)
	if !ok || res.cancelled || res.chosen < 0 {
		return -1, false
	}
	return res.chosen, true
}

// selectHorizontal : menu sur une ligne (menu principal).
func selectHorizontal(title string, items []string, status string) (int, bool) {
	return runSelect(title, items, true, status)
}

// selectOne : liste verticale (voyage, bestiaire, inspect).
func selectOne(title string, items []string) (int, bool) {
	return runSelect(title, items, false, "")
}
