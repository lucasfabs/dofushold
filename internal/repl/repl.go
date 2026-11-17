package repl

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lucas/dofushold/internal/game"
	"github.com/lucas/dofushold/internal/open5e"
	"github.com/lucas/dofushold/internal/ui"
)

// cliCommand = une commande (le sujet exige cette map ; help est genere dessus).
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
		"help":      {name: "help", description: "Affiche cette aide", callback: cmdHelp},
		"map":       {name: "map", description: "Carte ASCII des regions", callback: func(g *game.Game) error { fmt.Print(g.Map()); return nil }},
		"look":      {name: "look", description: "Decrit la region actuelle", callback: func(g *game.Game) error { fmt.Print(g.Look()); return nil }},
		"travel":    {name: "travel", description: "Choisir une sortie", callback: cmdTravel},
		"explore":   {name: "explore", description: "Explore (risque une rencontre)", callback: func(g *game.Game) error { return g.Explore() }},
		"fight":     {name: "fight", description: "Combat l'ennemi en cours", callback: func(g *game.Game) error { return g.Fight() }},
		"flee":      {name: "flee", description: "Tente de fuir le combat", callback: func(g *game.Game) error { return g.Flee() }},
		"character": {name: "character", description: "Fiche du personnage", callback: func(g *game.Game) error { fmt.Print(g.CharacterSheet()); return nil }},
		"rest":      {name: "rest", description: "Repos (1 fois par region)", callback: func(g *game.Game) error { return g.Rest() }},
		"bestiary":  {name: "bestiary", description: "Catalogue Open5e (menus)", callback: cmdBestiary},
		"inspect":   {name: "inspect", description: "Fiche d'un monstre (depuis le bestiaire)", callback: cmdInspect},
		"slain":     {name: "slain", description: "Monstres vaincus", callback: func(g *game.Game) error { fmt.Print(g.SlainList()); return nil }},
		"save":      {name: "save", description: "Sauvegarde la partie", callback: func(g *game.Game) error { return g.Save() }},
		"load": {name: "load", description: "Charge la sauvegarde", callback: func(g *game.Game) error {
			if err := g.Load(); err != nil {
				return err
			}
			fmt.Println(ui.Ok("Partie chargee."))
			fmt.Print(g.Look())
			return nil
		}},
		"quit": {name: "quit", description: "Quitte le jeu", callback: func(_ *game.Game) error {
			fmt.Println(ui.Accent("A bientot, aventurier."))
			os.Exit(0)
			return nil
		}},
	}
}

func cmdHelp(_ *game.Game) error {
	cmds := commands()
	keys := make([]string, 0, len(cmds))
	for k := range cmds {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println(ui.Banner("Aide"))
	for _, k := range keys {
		c := cmds[k]
		fmt.Printf("  %s  %s\n", ui.Accent(fmt.Sprintf("%-12s", c.name)), c.description)
	}
	return nil
}

func cmdTravel(g *game.Game) error {
	r := g.Region()
	if r == nil || len(r.Exits) == 0 {
		return fmt.Errorf("aucune sortie depuis ici")
	}
	if g.Encounter != nil {
		return fmt.Errorf("un ennemi bloque le chemin — combats ou fuis d'abord")
	}

	labels := make([]string, len(r.Exits))
	for i, id := range r.Exits {
		name := id
		if dest, ok := g.World.Get(id); ok {
			name = dest.Name
		}
		labels[i] = fmt.Sprintf("%s (%s)", name, id)
	}

	i, ok := selectOne("Ou veux-tu aller ?", labels)
	if !ok {
		fmt.Println(ui.Warn("Voyage annule."))
		return nil
	}
	return g.Travel(r.Exits[i])
}

func showBestiaryPage(g *game.Game, pageURL string) error {
	monsters, next, prev, err := g.API.ListPage(pageURL)
	if err != nil {
		return err
	}
	g.BestiaryURL = next + "\n" + prev
	g.LastBestiary = monsters

	fmt.Println(ui.Banner("Bestiaire"))
	for _, m := range monsters {
		fmt.Printf("  %s (%s) CR %s\n", ui.Bold(m.Name), m.Slug, m.ChallengeRating)
	}
	return nil
}

func cmdBestiary(g *game.Game) error {
	if err := showBestiaryPage(g, ""); err != nil {
		return err
	}

	for {
		parts := strings.SplitN(g.BestiaryURL, "\n", 2)
		next, prev := parts[0], ""
		if len(parts) == 2 {
			prev = parts[1]
		}

		labels := []string{}
		actions := []func() error{}

		if next != "" {
			labels = append(labels, "Page suivante")
			url := next
			actions = append(actions, func() error { return showBestiaryPage(g, url) })
		}
		if prev != "" {
			labels = append(labels, "Page precedente")
			url := prev
			actions = append(actions, func() error { return showBestiaryPage(g, url) })
		}
		if len(g.LastBestiary) > 0 {
			labels = append(labels, "Inspecter un monstre")
			actions = append(actions, func() error { return cmdInspect(g) })
		}
		labels = append(labels, "Retour au jeu")
		actions = append(actions, func() error { return errDone })

		i, ok := selectOne("Bestiaire — suite ?", labels)
		if !ok {
			return nil
		}
		if err := actions[i](); err != nil {
			if errors.Is(err, errDone) {
				return nil
			}
			return err
		}
	}
}

var errDone = errors.New("done")

func cmdInspect(g *game.Game) error {
	if len(g.LastBestiary) == 0 {
		return fmt.Errorf("ouvre d'abord le bestiaire pour charger une page")
	}

	labels := make([]string, len(g.LastBestiary))
	for i, m := range g.LastBestiary {
		labels[i] = fmt.Sprintf("%s (%s)", m.Name, m.Slug)
	}

	i, ok := selectOne("Quel monstre inspecter ?", labels)
	if !ok {
		return nil
	}

	m, err := g.API.GetMonster(g.LastBestiary[i].Slug)
	if err != nil {
		return err
	}
	printMonster(m)
	return nil
}

func printMonster(m *open5e.Monster) {
	fmt.Println(ui.Banner(m.Name))
	fmt.Printf("Slug: %s | PV: %s | CA: %d | CR: %s\n",
		m.Slug, ui.HPBar(m.HitPoints, m.HitPoints), m.ArmorClass, m.ChallengeRating)
	if len(m.Actions) == 0 {
		return
	}
	fmt.Println(ui.Accent("Actions:"))
	for _, a := range m.Actions {
		fmt.Printf("- %s: %s\n", ui.Bold(a.Name), a.Desc)
	}
}

func menuPrincipal(g *game.Game) []menuItem {
	cmds := commands()
	if g.Encounter != nil {
		return []menuItem{
			{label: "Combattre", run: cmds["fight"].callback},
			{label: "Fuir", run: cmds["flee"].callback},
			{label: "Personnage", run: cmds["character"].callback},
			{label: "Aide", run: cmds["help"].callback},
			{label: "Quitter", run: cmds["quit"].callback},
		}
	}
	return []menuItem{
		{label: "Regarder", run: cmds["look"].callback},
		{label: "Carte", run: cmds["map"].callback},
		{label: "Voyager", run: cmds["travel"].callback},
		{label: "Explorer", run: cmds["explore"].callback},
		{label: "Personnage", run: cmds["character"].callback},
		{label: "Se reposer", run: cmds["rest"].callback},
		{label: "Bestiaire", run: cmds["bestiary"].callback},
		{label: "Monstres vaincus", run: cmds["slain"].callback},
		{label: "Sauvegarder", run: cmds["save"].callback},
		{label: "Charger", run: cmds["load"].callback},
		{label: "Aide", run: cmds["help"].callback},
		{label: "Quitter", run: cmds["quit"].callback},
	}
}

// Run boucle de menus Bubble Tea (horizontal pour le menu principal).
func Run(g *game.Game) {
	fmt.Println(ui.Banner("Dofushold"))
	fmt.Println(ui.Accent("Menu: ← → Entrée. Sous-menus: ↑ ↓ Entrée. Esc pour annuler."))
	fmt.Println(ui.Separator())
	fmt.Print(g.Look())

	for {
		foeName, foeHP, foeMax := "", 0, 0
		if g.Encounter != nil {
			foeName = g.Encounter.Name
			foeHP, foeMax = g.FoeHP, g.Encounter.HitPoints
		}
		status := ui.StatusLine(g.Hero.HP, g.Hero.MaxHP, foeName, foeHP, foeMax)

		items := menuPrincipal(g)
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.label
		}

		title := "Que veux-tu faire ?"
		if g.Encounter != nil {
			title = fmt.Sprintf("Rencontre: %s", g.Encounter.Name)
		}

		i, ok := selectHorizontal(title, labels, status)
		if !ok {
			fmt.Println(ui.Warn("Menu annule — choisis une action ou Quitter."))
			continue
		}
		if err := items[i].run(g); err != nil {
			fmt.Println(ui.Bad("Erreur: " + err.Error()))
		}
		fmt.Println(ui.Separator())
	}
}
