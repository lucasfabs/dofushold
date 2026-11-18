package repl

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas/dofushold/internal/game"
	"github.com/lucas/dofushold/internal/ui"
)

// Modes d'affichage du menu du bas.
const (
	modeMain = iota
	modeTravel
	modeBestiary
	modeInspect
)

// writeMsg = morceau de texte envoye par ui.TypePrint vers la TUI.
type writeMsg string

// actionDoneMsg = une action (combat, look, ...) est terminee.
type actionDoneMsg struct{ err error }

// appModel = ecran unique : zone logs + status + menu fixe.
type appModel struct {
	g      *game.Game
	width  int
	height int

	lines   []string // lignes terminees de la zone "tele"
	pending string   // ligne en cours de frappe

	mode   int
	busy   bool
	cursor int
	items  []menuItem

	// Sous-menus : labels + ids/actions associes
	pickLabels []string
	pickIDs    []string // travel: id region ; inspect: slug
}

func newApp(g *game.Game) appModel {
	m := appModel{g: g, width: 80, height: 24}
	m.refreshMainMenu()
	m.ingest(ui.Banner("Dofushold") + "\n")
	m.ingest(ui.Accent("Menu fixe en bas. Zone haute = actions.") + "\n")
	m.ingest(ui.Separator() + "\n")
	m.ingest(g.Look())
	return m
}

func (m appModel) Init() tea.Cmd {
	return nil
}

func (m *appModel) ingest(s string) {
	for _, r := range s {
		if r == '\n' {
			m.lines = append(m.lines, m.pending)
			m.pending = ""
			continue
		}
		m.pending += string(r)
	}
	m.lines = trimLines(m.lines, maxLogKeep)
}

func (m *appModel) refreshMainMenu() {
	m.mode = modeMain
	m.items = menuPrincipal(m.g)
	m.cursor = 0
	if m.cursor >= len(m.items) {
		m.cursor = 0
	}
}

func (m appModel) statusText() string {
	foeName, foeHP, foeMax := "", 0, 0
	if m.g.Encounter != nil {
		foeName = m.g.Encounter.Name
		foeHP, foeMax = m.g.FoeHP, m.g.Encounter.HitPoints
	}
	return ui.StatusLine(m.g.Hero.HP, m.g.Hero.MaxHP, foeName, foeHP, foeMax)
}

func (m appModel) menuTitle() string {
	switch m.mode {
	case modeTravel:
		return "Ou veux-tu aller ?"
	case modeBestiary:
		return "Bestiaire — suite ?"
	case modeInspect:
		return "Quel monstre inspecter ?"
	default:
		if m.g.Encounter != nil {
			return fmt.Sprintf("Rencontre: %s", m.g.Encounter.Name)
		}
		return "Que veux-tu faire ?"
	}
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case writeMsg:
		m.ingest(string(msg))
		return m, nil

	case actionDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.ingest(ui.Bad("Erreur: "+msg.err.Error()) + "\n")
		}
		m.ingest(ui.Separator() + "\n")
		m.refreshMainMenu()
		return m, nil

	case bestiaryReadyMsg:
		return m.openBestiaryMenu()

	case inspectDoneMsg:
		m.busy = false
		m.ingest(ui.Separator() + "\n")
		return m.openBestiaryMenu()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m appModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.busy {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.mode != modeMain {
			m.refreshMainMenu()
			m.ingest(ui.Warn("Annule.") + "\n")
		}
		return m, nil
	case "left", "h":
		if m.mode == modeMain {
			m.moveCursor(-1)
		}
	case "right", "l":
		if m.mode == modeMain {
			m.moveCursor(1)
		}
	case "up", "k":
		if m.mode != modeMain {
			m.moveCursor(-1)
		}
	case "down", "j":
		if m.mode != modeMain {
			m.moveCursor(1)
		}
	case "enter":
		return m.confirm()
	}
	return m, nil
}

func (m *appModel) moveCursor(delta int) {
	n := m.choiceCount()
	if n == 0 {
		return
	}
	m.cursor = (m.cursor + delta + n) % n
}

func (m appModel) choiceCount() int {
	if m.mode == modeMain {
		return len(m.items)
	}
	return len(m.pickLabels)
}

func (m appModel) choiceLabel(i int) string {
	if m.mode == modeMain {
		return m.items[i].label
	}
	return m.pickLabels[i]
}

func (m appModel) confirm() (tea.Model, tea.Cmd) {
	if m.mode != modeMain {
		return m.confirmPick()
	}
	if len(m.items) == 0 {
		return m, nil
	}
	item := m.items[m.cursor]
	switch item.label {
	case "Quitter":
		m.ingest(ui.Accent("A bientot, aventurier.") + "\n")
		return m, tea.Quit
	case "Voyager":
		return m.openTravel()
	case "Bestiaire":
		return m.startBestiary()
	default:
		m.busy = true
		cb := item.run
		return m, func() tea.Msg {
			return actionDoneMsg{err: cb(m.g)}
		}
	}
}

// Run lance la TUI plein ecran (menu fixe en bas).
func Run(g *game.Game) {
	m := newApp(g)
	p := tea.NewProgram(m, tea.WithAltScreen())
	ui.SetSink(func(s string) { p.Send(writeMsg(s)) })
	defer ui.SetSink(nil)

	if _, err := p.Run(); err != nil {
		fmt.Println(ui.Bad(err.Error()))
	}
}

// trimLines garde les dernieres lignes pour ne pas exploser la memoire.
func trimLines(lines []string, max int) []string {
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}
