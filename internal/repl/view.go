package repl

import (
	"strings"

	"github.com/lucas/dofushold/internal/ui"
)

const (
	footerMain   = 6  // status + titre + 1 ligne choix + bord
	footerSubMax = 12 // sous-menu vertical : plus de place en bas
	maxLogKeep   = 500
)

func (m appModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}
	h := m.height
	if h < 12 {
		h = 12
	}

	footer := footerMain
	if m.mode != modeMain {
		n := m.choiceCount()
		footer = 4 + n // titre + status + bord + n lignes
		if footer > footerSubMax {
			footer = footerSubMax
		}
	}
	tvHeight := h - footer - 2 // cadre haut/bas
	if tvHeight < 3 {
		tvHeight = 3
	}

	innerW := w - 2
	if innerW < 10 {
		innerW = 10
	}

	var b strings.Builder
	top := "┌" + strings.Repeat("─", innerW) + "┐"
	bot := "└" + strings.Repeat("─", innerW) + "┘"
	b.WriteString(ui.Accent(top))
	b.WriteString("\n")

	visible := m.visibleLogLines(tvHeight)
	for _, line := range visible {
		b.WriteString("│")
		b.WriteString(padClip(line, innerW))
		b.WriteString("│\n")
	}

	b.WriteString(ui.Accent(bot))
	b.WriteString("\n")
	b.WriteString(padClip(m.statusText(), w))
	b.WriteString("\n")

	hint := "(← → Entrée)"
	if m.mode != modeMain {
		hint = "(↑ ↓ Entrée, Esc annule)"
	}
	if m.busy {
		hint = "(action en cours...)"
	}
	b.WriteString(ui.Accent(m.menuTitle() + "  " + hint))
	b.WriteString("\n")
	b.WriteString(m.renderChoices())
	b.WriteString("\n")
	b.WriteString(ui.FullWidthLine(w))
	return b.String()
}

// visibleLogLines = dernieres lignes + ligne en cours, cadre rempli.
func (m appModel) visibleLogLines(height int) []string {
	all := append([]string{}, m.lines...)
	if m.pending != "" || len(all) == 0 {
		all = append(all, m.pending)
	}
	if len(all) > height {
		all = all[len(all)-height:]
	}
	// Remplir le haut avec des lignes vides pour coller le texte en bas du cadre.
	for len(all) < height {
		all = append([]string{""}, all...)
	}
	return all
}

func (m appModel) renderChoices() string {
	n := m.choiceCount()
	if n == 0 {
		return ""
	}
	// Menu principal : une ligne horizontale. Sous-menus : liste verticale.
	if m.mode == modeMain {
		parts := make([]string, n)
		for i := 0; i < n; i++ {
			label := m.choiceLabel(i)
			if i == m.cursor && !m.busy {
				parts[i] = ui.Accent(ui.Bold("▸ " + label))
			} else {
				parts[i] = "  " + label
			}
		}
		return strings.Join(parts, " | ")
	}

	var b strings.Builder
	maxShow := footerSubMax - 4
	start := 0
	if n > maxShow {
		start = m.cursor - maxShow/2
		if start < 0 {
			start = 0
		}
		if start > n-maxShow {
			start = n - maxShow
		}
	}
	end := start + maxShow
	if end > n {
		end = n
	}
	for i := start; i < end; i++ {
		label := m.choiceLabel(i)
		if i == m.cursor && !m.busy {
			b.WriteString(ui.Accent(ui.Bold("▸ " + label)))
		} else {
			b.WriteString("  " + label)
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// padClip aligne une ligne a width colonnes (ANSI ignore pour le padding simple).
func padClip(s string, width int) string {
	plain := stripANSI(s)
	runes := []rune(plain)
	if len(runes) > width {
		return string(runes[:width])
	}
	// Si s a des couleurs, on pad a la fin sans tout recalculer parfaitement.
	pad := width - len(runes)
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
