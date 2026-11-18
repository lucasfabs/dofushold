package repl

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas/dofushold/internal/ui"
)

func (m appModel) openTravel() (tea.Model, tea.Cmd) {
	r := m.g.Region()
	if r == nil || len(r.Exits) == 0 {
		m.ingest(ui.Bad("aucune sortie depuis ici") + "\n")
		return m, nil
	}
	if m.g.Encounter != nil {
		m.ingest(ui.Bad("un ennemi bloque le chemin") + "\n")
		return m, nil
	}

	labels := make([]string, len(r.Exits))
	ids := make([]string, len(r.Exits))
	for i, id := range r.Exits {
		name := id
		if dest, ok := m.g.World.Get(id); ok {
			name = dest.Name
		}
		labels[i] = fmt.Sprintf("%s (%s)", name, id)
		ids[i] = id
	}
	m.mode = modeTravel
	m.pickLabels = labels
	m.pickIDs = ids
	m.cursor = 0
	return m, nil
}

func (m appModel) startBestiary() (tea.Model, tea.Cmd) {
	m.busy = true
	return m, func() tea.Msg {
		err := showBestiaryPage(m.g, "")
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return bestiaryReadyMsg{}
	}
}

// bestiaryReadyMsg = page bestiaire chargee, ouvrir le sous-menu.
type bestiaryReadyMsg struct{}

func (m appModel) openBestiaryMenu() (tea.Model, tea.Cmd) {
	parts := strings.SplitN(m.g.BestiaryURL, "\n", 2)
	next, prev := parts[0], ""
	if len(parts) == 2 {
		prev = parts[1]
	}

	labels := []string{}
	ids := []string{}
	if next != "" {
		labels = append(labels, "Page suivante")
		ids = append(ids, "next:"+next)
	}
	if prev != "" {
		labels = append(labels, "Page precedente")
		ids = append(ids, "prev:"+prev)
	}
	if len(m.g.LastBestiary) > 0 {
		labels = append(labels, "Inspecter un monstre")
		ids = append(ids, "inspect")
	}
	labels = append(labels, "Retour au jeu")
	ids = append(ids, "back")

	m.busy = false
	m.mode = modeBestiary
	m.pickLabels = labels
	m.pickIDs = ids
	m.cursor = 0
	return m, nil
}

func (m appModel) openInspect() (tea.Model, tea.Cmd) {
	if len(m.g.LastBestiary) == 0 {
		m.ingest(ui.Bad("ouvre d'abord le bestiaire") + "\n")
		return m.openBestiaryMenu()
	}
	labels := make([]string, len(m.g.LastBestiary))
	ids := make([]string, len(m.g.LastBestiary))
	for i, mon := range m.g.LastBestiary {
		labels[i] = fmt.Sprintf("%s (%s)", mon.Name, mon.Slug)
		ids[i] = mon.Slug
	}
	m.mode = modeInspect
	m.pickLabels = labels
	m.pickIDs = ids
	m.cursor = 0
	return m, nil
}

func (m appModel) confirmPick() (tea.Model, tea.Cmd) {
	if len(m.pickLabels) == 0 {
		m.refreshMainMenu()
		return m, nil
	}
	id := m.pickIDs[m.cursor]

	switch m.mode {
	case modeTravel:
		m.busy = true
		dest := id
		return m, func() tea.Msg {
			return actionDoneMsg{err: m.g.Travel(dest)}
		}

	case modeInspect:
		m.busy = true
		slug := id
		return m, func() tea.Msg {
			mon, err := m.g.API.GetMonster(slug)
			if err != nil {
				return actionDoneMsg{err: err}
			}
			printMonster(mon)
			return inspectDoneMsg{}
		}

	case modeBestiary:
		switch {
		case id == "back":
			m.refreshMainMenu()
			return m, nil
		case id == "inspect":
			return m.openInspect()
		case strings.HasPrefix(id, "next:"):
			url := strings.TrimPrefix(id, "next:")
			m.busy = true
			return m, func() tea.Msg {
				if err := showBestiaryPage(m.g, url); err != nil {
					return actionDoneMsg{err: err}
				}
				return bestiaryReadyMsg{}
			}
		case strings.HasPrefix(id, "prev:"):
			url := strings.TrimPrefix(id, "prev:")
			m.busy = true
			return m, func() tea.Msg {
				if err := showBestiaryPage(m.g, url); err != nil {
					return actionDoneMsg{err: err}
				}
				return bestiaryReadyMsg{}
			}
		}
	}
	return m, nil
}

// inspectDoneMsg = fiche monstre affichee, revenir au menu bestiaire.
type inspectDoneMsg struct{}
