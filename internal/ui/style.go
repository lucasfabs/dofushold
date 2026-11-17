package ui

import (
	"fmt"
	"os"
	"strings"
)

// Codes ANSI simples (bonus couleurs du sujet).
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
)

func enabled() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return !ok
}

func wrap(color, s string) string {
	if !enabled() {
		return s
	}
	return color + s + reset
}

func Cyan(s string) string   { return wrap(cyan, s) }
func Green(s string) string  { return wrap(green, s) }
func Yellow(s string) string { return wrap(yellow, s) }
func Red(s string) string    { return wrap(red, s) }
func Bold(s string) string   { return wrap(bold, s) }

func Ok(s string) string     { return Green(s) }
func Warn(s string) string   { return Yellow(s) }
func Bad(s string) string    { return Red(s) }
func Accent(s string) string { return Cyan(s) }

// Icon ignore l'emoji (regle projet) et renvoie le texte.
func Icon(_ string, text string) string { return text }

// Banner cadre d'accueil.
func Banner(title string) string {
	line := strings.Repeat("─", len(title)+4)
	top := "┌" + line + "┐"
	mid := "│  " + title + "  │"
	bot := "└" + line + "┘"
	return Accent(top) + "\n" + Accent(Bold(mid)) + "\n" + Accent(bot)
}

func Separator() string {
	return Accent("────────")
}

// FullWidthLine tiret horizontal sur toute la largeur du terminal.
// width <= 0 utilise une largeur par defaut.
func FullWidthLine(width int) string {
	if width < 8 {
		width = 40
	}
	return Accent(strings.Repeat("─", width))
}

// Danger colore le niveau 1-5.
func Danger(level int) string {
	label := fmt.Sprintf("Danger %d/5", level)
	switch {
	case level >= 4:
		return Bad(label)
	case level == 3:
		return Warn(label)
	default:
		return Ok(label)
	}
}

// HPBar barre [####----] coloree selon le ratio.
func HPBar(hp, max int) string {
	if max < 1 {
		max = 1
	}
	if hp < 0 {
		hp = 0
	}
	width := 10
	filled := (hp * width) / max
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	label := fmt.Sprintf("[%s] %d/%d", bar, hp, max)
	ratio := float64(hp) / float64(max)
	switch {
	case ratio > 0.6:
		return Ok(label)
	case ratio > 0.3:
		return Warn(label)
	default:
		return Bad(label)
	}
}

// StatusLine affiche les PV du heros, et ceux de l'ennemi si foeName n'est pas vide.
func StatusLine(heroHP, heroMax int, foeName string, foeHP, foeMax int) string {
	line := "PV " + HPBar(heroHP, heroMax)
	if foeName != "" {
		line += " | " + Bold(foeName) + " " + HPBar(foeHP, foeMax)
	}
	return line
}
