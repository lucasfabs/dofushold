package ui

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	charDelay = 15 * time.Millisecond
	linePause = 300 * time.Millisecond
)

// typewriterOn = false si pas un TTY ou si NO_TYPEWRITER est defini.
func typewriterOn() bool {
	if _, ok := os.LookupEnv("NO_TYPEWRITER"); ok {
		return false
	}
	// Avec sink TUI, on garde l'effet meme si on ne teste que stdout.
	if sink != nil {
		return true
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// TypePrint ecrit s rune par rune (sequences ANSI d'un coup).
func TypePrint(s string) {
	if !typewriterOn() {
		Write(s)
		return
	}

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		// Sequence ANSI complete : \033[...m
		if runes[i] == '\033' {
			end := i + 1
			for end < len(runes) && runes[end-1] != 'm' {
				end++
			}
			Write(string(runes[i:end]))
			i = end - 1
			continue
		}
		Write(string(runes[i]))
		if runes[i] != '\n' {
			time.Sleep(charDelay)
		}
	}
}

// TypePrintln = TypePrint + saut de ligne + pause narrative.
func TypePrintln(s string) {
	TypePrint(s)
	Write("\n")
	if typewriterOn() {
		time.Sleep(linePause)
	}
}

// TypePrintf formate puis TypePrint (sans pause de fin de ligne).
func TypePrintf(format string, args ...any) {
	TypePrint(fmt.Sprintf(format, args...))
}
