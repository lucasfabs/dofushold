package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/lucas/dofushold/internal/open5e"
	"github.com/lucas/dofushold/internal/ui"
	"github.com/lucas/dofushold/internal/world"
)

// Character = fiche du joueur.
type Character struct {
	HP          int    `json:"hp"`
	MaxHP       int    `json:"max_hp"`
	Level       int    `json:"level"`
	XP          int    `json:"xp"`
	AttackBonus int    `json:"attack_bonus"`
	Weapon      string `json:"weapon"`
	DamageDice  string `json:"damage_dice"` // ex: "1d8"
	DamageBonus int    `json:"damage_bonus"`
}

type Game struct {
	World    *world.World
	API      *open5e.Client
	SavePath string

	Location string
	Hero     Character
	Gold     int
	Slain    map[string]int

	// RestOK = true tant qu'on n'a pas encore repose dans cette region.
	RestOK bool

	// Encounter = monstre en cours apres explore (nil sinon).
	Encounter *open5e.Monster
	FoeHP     int

	// Pagination bestiaire Open5e + derniere page affichee (pour inspect).
	BestiaryURL  string
	LastBestiary []open5e.Monster
}

type saveFile struct {
	Location string         `json:"location"`
	Hero     Character      `json:"hero"`
	Gold     int            `json:"gold"`
	Slain    map[string]int `json:"slain"`
	RestOK   bool           `json:"rest_ok"`
}

func New(w *world.World, api *open5e.Client, savePath string) *Game {
	return &Game{
		World:    w,
		API:      api,
		SavePath: savePath,
		Location: w.Start,
		Hero: Character{
			HP: 20, MaxHP: 20, Level: 1, XP: 0,
			AttackBonus: 3, Weapon: "Epee courte",
			DamageDice: "1d8", DamageBonus: 2,
		},
		Gold:   0,
		Slain:  make(map[string]int),
		RestOK: true,
	}
}

func (g *Game) Region() *world.Region {
	r, _ := g.World.Get(g.Location)
	return r
}

func (g *Game) Look() string {
	r := g.Region()
	if r == nil {
		return "Tu es nulle part."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", ui.Banner(r.Name))
	fmt.Fprintf(&b, "%s\n", r.Description)
	fmt.Fprintf(&b, "%s\n", ui.Danger(r.Danger))
	fmt.Fprintf(&b, "Sorties: %s\n", ui.Accent(strings.Join(r.Exits, ", ")))
	return b.String()
}

// labelMap affiche un nom + danger colore ; ★ si tu es ici.
func (g *Game) labelMap(id string) string {
	r, ok := g.World.Get(id)
	if !ok {
		return id
	}
	s := fmt.Sprintf("%s (%d)", r.Name, r.Danger)
	switch {
	case r.Danger >= 4:
		s = ui.Bad(s)
	case r.Danger == 3:
		s = ui.Warn(s)
	default:
		s = ui.Ok(s)
	}
	if g.Location == id {
		return ui.Accent("★ " + s + " ★")
	}
	return s
}

// Map dessine la carte ASCII (meme mise en page que lore.md).
func (g *Game) Map() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", ui.Banner("Carte"))
	fmt.Fprintf(&b, "              %s\n", g.labelMap("frigost"))
	fmt.Fprintf(&b, "                 |\n")
	fmt.Fprintf(&b, "    %s--%s--%s\n",
		g.labelMap("astrub"), g.labelMap("crossroads"), g.labelMap("amakna"))
	fmt.Fprintf(&b, "                 |\n")
	fmt.Fprintf(&b, "              %s\n", g.labelMap("bonta"))
	fmt.Fprintf(&b, "                 |\n")
	fmt.Fprintf(&b, "             %s\n", g.labelMap("brakmar"))
	fmt.Fprintf(&b, "\n%s\n", ui.Accent("★ ... ★ = tu es ici"))
	return b.String()
}

func (g *Game) Travel(to string) error {
	to = strings.TrimSpace(strings.ToLower(to))
	if g.Encounter != nil {
		return fmt.Errorf("un ennemi bloque le chemin — combats ou fuis d'abord")
	}
	if !g.World.CanTravel(g.Location, to) {
		return fmt.Errorf("impossible d'aller a %q depuis ici", to)
	}
	if _, ok := g.World.Get(to); !ok {
		return fmt.Errorf("region inconnue: %s", to)
	}
	g.Location = to
	g.RestOK = true // nouveau sejour = un repos possible
	r := g.Region()
	ui.TypePrintln(ui.Ok(ui.Icon("🧭", "Tu arrives a "+r.Name+".")))
	ui.TypePrintln(r.Description)
	return nil
}

func (g *Game) CharacterSheet() string {
	h := g.Hero
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", ui.Banner("Personnage"))
	fmt.Fprintf(&b, "%s | %s | %s | %s\n",
		ui.Icon("⭐", "Niveau "+ui.Accent(fmt.Sprintf("%d", h.Level))),
		ui.Icon("❤️", "PV")+" "+ui.HPBar(h.HP, h.MaxHP),
		ui.Icon("✨", fmt.Sprintf("XP %d (prochain: %d)", h.XP, h.Level*h.Level*50)),
		ui.Icon("⚔️", fmt.Sprintf("Attaque +%d", h.AttackBonus)))
	fmt.Fprintf(&b, "%s | %s\n",
		ui.Icon("🗡️", fmt.Sprintf("Arme: %s (%s+%d)", h.Weapon, h.DamageDice, h.DamageBonus)),
		ui.Icon("🪙", "Or: "+ui.Yellow(fmt.Sprintf("%d", g.Gold))))
	return b.String()
}

func (g *Game) Rest() error {
	if !g.RestOK {
		return fmt.Errorf("tu as deja repose ici — voyage ailleurs pour te reposer a nouveau")
	}
	g.Hero.HP = g.Hero.MaxHP
	g.RestOK = false
	ui.TypePrintln(ui.Ok(ui.Icon("😴", "Tu te reposes. PV restaures.")))
	return nil
}

// Explore tire un ennemi (plusieurs jets si danger eleve, on garde le pire CR).
func (g *Game) Explore() error {
	if g.Encounter != nil {
		return fmt.Errorf("tu combats deja %s", g.Encounter.Name)
	}
	r := g.Region()
	if r == nil || len(r.Foes) == 0 {
		return fmt.Errorf("rien a explorer ici")
	}

	rolls := r.Danger
	if rolls < 1 {
		rolls = 1
	}

	var best *open5e.Monster
	for i := 0; i < rolls; i++ {
		slug := r.Foes[rand.Intn(len(r.Foes))]
		m, err := g.API.GetMonster(slug)
		if err != nil {
			return fmt.Errorf("chargement ennemi: %w", err)
		}
		if best == nil || open5e.CRFloat(m.ChallengeRating) > open5e.CRFloat(best.ChallengeRating) {
			best = m
		}
	}

	g.Encounter = best
	g.FoeHP = best.HitPoints
	ui.TypePrintln(fmt.Sprintf("%s %s (PV %s, CA %d, CR %s)",
		ui.Bad(ui.Icon("👹", "Une rencontre !")), ui.Bold(best.Name),
		ui.HPBar(best.HitPoints, best.HitPoints), best.ArmorClass, best.ChallengeRating))
	ui.TypePrintln(ui.Warn("Choisis Combattre ou Fuir dans le menu."))
	return nil
}

func (g *Game) addXP(amount int) {
	g.Hero.XP += amount
	for g.Hero.XP >= g.Hero.Level*g.Hero.Level*50 {
		g.Hero.XP -= g.Hero.Level * g.Hero.Level * 50
		g.Hero.Level++
		g.Hero.MaxHP += 5
		g.Hero.HP = g.Hero.MaxHP
		g.Hero.AttackBonus++
		ui.TypePrintln(ui.Ok(ui.Icon("⭐", fmt.Sprintf("Niveau superieur ! Tu es niveau %d.", g.Hero.Level))))
	}
}

func (g *Game) dieAndRespawn() {
	ui.TypePrintln(ui.Bad(ui.Icon("💀", "Tu tombes... Tu te reveilles au Carrefour, blesse (1 PV).")))
	g.Encounter = nil
	g.FoeHP = 0
	g.Location = g.World.Start
	g.Hero.HP = 1
	g.RestOK = true
}

func (g *Game) Save() error {
	data := saveFile{
		Location: g.Location,
		Hero:     g.Hero,
		Gold:     g.Gold,
		Slain:    g.Slain,
		RestOK:   g.RestOK,
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode save: %w", err)
	}
	if err := os.WriteFile(g.SavePath, b, 0o644); err != nil {
		return fmt.Errorf("ecriture save: %w", err)
	}
	ui.TypePrintln(ui.Ok(ui.Icon("💾", "Partie sauvegardee.")))
	return nil
}

func (g *Game) Load() error {
	b, err := os.ReadFile(g.SavePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("aucune sauvegarde trouvee")
		}
		return fmt.Errorf("lecture save: %w", err)
	}
	var data saveFile
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("decode save: %w", err)
	}
	if _, ok := g.World.Get(data.Location); !ok {
		return fmt.Errorf("sauvegarde invalide: region %s", data.Location)
	}
	g.Location = data.Location
	g.Hero = data.Hero
	g.Gold = data.Gold
	g.Slain = data.Slain
	if g.Slain == nil {
		g.Slain = make(map[string]int)
	}
	g.RestOK = data.RestOK
	g.Encounter = nil
	g.FoeHP = 0
	return nil
}

func (g *Game) SlainList() string {
	if len(g.Slain) == 0 {
		return "Aucun monstre vaincu pour l'instant."
	}
	var b strings.Builder
	for name, n := range g.Slain {
		fmt.Fprintf(&b, "- %s x%d\n", name, n)
	}
	return b.String()
}
