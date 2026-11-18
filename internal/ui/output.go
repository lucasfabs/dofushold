package ui

import "fmt"

// sink recoit le texte des actions (branche vers la TUI si defini).
var sink func(string)

// SetSink redirige TypePrint / Write vers la TUI. Passer nil pour stdout.
func SetSink(fn func(string)) {
	sink = fn
}

// Write envoie s vers la TUI ou stdout.
func Write(s string) {
	if sink != nil {
		sink(s)
		return
	}
	fmt.Print(s)
}

// Writeln = Write + saut de ligne.
func Writeln(s string) {
	Write(s + "\n")
}
