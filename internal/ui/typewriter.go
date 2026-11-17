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
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// TypePrint ecrit s rune par rune (sequences ANSI d'un coup).
func TypePrint(s string) {
	if !typewriterOn() {
		fmt.Print(s)
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
			fmt.Print(string(runes[i:end]))
			i = end - 1
			continue
		}
		fmt.Print(string(runes[i]))
		_ = os.Stdout.Sync()
		if runes[i] != '\n' {
			time.Sleep(charDelay)
		}
	}
}

// TypePrintln = TypePrint + saut de ligne + pause narrative.
func TypePrintln(s string) {
	TypePrint(s)
	fmt.Println()
	if typewriterOn() {
		time.Sleep(linePause)
	}
}

// TypePrintf formate puis TypePrint (sans pause de fin de ligne).
func TypePrintf(format string, args ...any) {
	TypePrint(fmt.Sprintf(format, args...))
}
