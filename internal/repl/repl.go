package repl

import (
	"fmt"

	"github.com/lucas/dofushold/internal/game"
	"github.com/lucas/dofushold/internal/open5e"
	"github.com/lucas/dofushold/internal/ui"
)

// cliCommand = une commande (le sujet exige cette map).
type cliCommand struct {
	name        string
	description string
	callback    func(g *game.Game) error
}

type menuItem struct {
	label string
	run   func(g *game.Game) error
}

func commands() map[string]cliCommand {
	return map[string]cliCommand{
		"map":       {name: "map", description: "Carte ASCII des regions", callback: func(g *game.Game) error { ui.Write(g.Map()); return nil }},
		"look":      {name: "look", description: "Decrit la region actuelle", callback: func(g *game.Game) error { ui.TypePrint(g.Look()); return nil }},
		"travel":    {name: "travel", description: "Choisir une sortie", callback: nil}, // gere par la TUI
		"explore":   {name: "explore", description: "Explore (risque une rencontre)", callback: func(g *game.Game) error { return g.Explore() }},
		"fight":     {name: "fight", description: "Combat l'ennemi en cours", callback: func(g *game.Game) error { return g.Fight() }},
		"flee":      {name: "flee", description: "Tente de fuir le combat", callback: func(g *game.Game) error { return g.Flee() }},
		"character": {name: "character", description: "Fiche du personnage", callback: func(g *game.Game) error { ui.TypePrint(g.CharacterSheet()); return nil }},
		"rest":      {name: "rest", description: "Repos (1 fois par region)", callback: func(g *game.Game) error { return g.Rest() }},
		"bestiary":  {name: "bestiary", description: "Catalogue Open5e (menus)", callback: nil}, // gere par la TUI
		"inspect":   {name: "inspect", description: "Fiche d'un monstre (depuis le bestiaire)", callback: nil},
		"slain":     {name: "slain", description: "Monstres vaincus", callback: func(g *game.Game) error { ui.Write(g.SlainList()); return nil }},
		"save":      {name: "save", description: "Sauvegarde la partie", callback: func(g *game.Game) error { return g.Save() }},
		"load": {name: "load", description: "Charge la sauvegarde", callback: func(g *game.Game) error {
			if err := g.Load(); err != nil {
				return err
			}
			ui.TypePrintln(ui.Ok("Partie chargee."))
			ui.TypePrint(g.Look())
			return nil
		}},
		"quit": {name: "quit", description: "Quitte le jeu", callback: nil}, // gere par la TUI
	}
}

func showBestiaryPage(g *game.Game, pageURL string) error {
	monsters, next, prev, err := g.API.ListPage(pageURL)
	if err != nil {
		return err
	}
	g.BestiaryURL = next + "\n" + prev
	g.LastBestiary = monsters

	ui.Writeln(ui.Banner("Bestiaire"))
	for _, mon := range monsters {
		ui.Writeln(fmt.Sprintf("  %s (%s) CR %s", ui.Bold(mon.Name), mon.Slug, mon.ChallengeRating))
	}
	return nil
}

func printMonster(m *open5e.Monster) {
	ui.Writeln(ui.Banner(m.Name))
	ui.Writeln(fmt.Sprintf("Slug: %s | PV: %s | CA: %d | CR: %s",
		m.Slug, ui.HPBar(m.HitPoints, m.HitPoints), m.ArmorClass, m.ChallengeRating))
	if len(m.Actions) == 0 {
		return
	}
	ui.Writeln(ui.Accent("Actions:"))
	for _, a := range m.Actions {
		ui.Writeln(fmt.Sprintf("- %s: %s", ui.Bold(a.Name), a.Desc))
	}
}

func menuPrincipal(g *game.Game) []menuItem {
	cmds := commands()
	// En rencontre : un choix par tour (Combattre = 1 tour, ou Fuir).
	if g.Encounter != nil {
		return []menuItem{
			{label: "Combattre", run: cmds["fight"].callback},
			{label: "Fuir", run: cmds["flee"].callback},
			{label: "Personnage", run: cmds["character"].callback},
			{label: "Quitter", run: nil},
		}
	}
	return []menuItem{
		{label: "Regarder", run: cmds["look"].callback},
		{label: "Carte", run: cmds["map"].callback},
		{label: "Voyager", run: nil},
		{label: "Explorer", run: cmds["explore"].callback},
		{label: "Personnage", run: cmds["character"].callback},
		{label: "Se reposer", run: cmds["rest"].callback},
		{label: "Bestiaire", run: nil},
		{label: "Monstres vaincus", run: cmds["slain"].callback},
		{label: "Sauvegarder", run: cmds["save"].callback},
		{label: "Charger", run: cmds["load"].callback},
		{label: "Quitter", run: nil},
	}
}
