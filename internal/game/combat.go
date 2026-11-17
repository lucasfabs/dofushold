package game

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/lucas/dofushold/internal/open5e"
	"github.com/lucas/dofushold/internal/ui"
)

// rollDice parse "1d8" ou "2d6" et tire le total.
func rollDice(expr string) int {
	expr = strings.TrimSpace(strings.ToLower(expr))
	if expr == "" {
		return 1
	}
	parts := strings.Split(expr, "d")
	if len(parts) != 2 {
		n, _ := strconv.Atoi(expr)
		if n < 1 {
			return 1
		}
		return n
	}
	n, _ := strconv.Atoi(parts[0])
	sides, _ := strconv.Atoi(parts[1])
	if n < 1 {
		n = 1
	}
	if sides < 1 {
		sides = 4
	}
	total := 0
	for i := 0; i < n; i++ {
		total += rand.Intn(sides) + 1
	}
	return total
}

func (g *Game) playerAttack(foeAC int) (hit bool, dmg int) {
	roll := rand.Intn(20) + 1
	total := roll + g.Hero.AttackBonus
	ui.TypePrintln(fmt.Sprintf("  Tu attaques: d20=%d +%d = %d vs CA %d", roll, g.Hero.AttackBonus, total, foeAC))
	if total >= foeAC {
		dmg = rollDice(g.Hero.DamageDice) + g.Hero.DamageBonus
		if dmg < 1 {
			dmg = 1
		}
		return true, dmg
	}
	return false, 0
}

func (g *Game) foeAttack(m *open5e.Monster) (hit bool, dmg int) {
	bonus := 2
	dice := "1d6"
	dmgBonus := 0
	if len(m.Actions) > 0 {
		a := m.Actions[0]
		if a.AttackBonus != 0 {
			bonus = a.AttackBonus
		}
		if a.DamageDice != "" {
			dice = a.DamageDice
		}
		dmgBonus = a.DamageBonus
	}

	roll := rand.Intn(20) + 1
	total := roll + bonus
	ui.TypePrintln(fmt.Sprintf("  %s attaque: d20=%d +%d = %d vs CA 12", m.Name, roll, bonus, total))
	if total >= 12 {
		dmg = rollDice(dice) + dmgBonus
		if dmg < 1 {
			dmg = 1
		}
		return true, dmg
	}
	return false, 0
}

// Fight deroule tout le combat jusqu'a victoire, defaite, ou plus d'ennemi.
func (g *Game) Fight() error {
	if g.Encounter == nil {
		return fmt.Errorf("aucun ennemi — explore d'abord")
	}
	m := g.Encounter
	ui.TypePrintln(ui.Banner("Combat: " + m.Name))

	for g.FoeHP > 0 && g.Hero.HP > 0 {
		ui.TypePrintln(fmt.Sprintf("--- Tour (toi %s | %s %s) ---",
			ui.HPBar(g.Hero.HP, g.Hero.MaxHP), m.Name, ui.HPBar(g.FoeHP, m.HitPoints)))

		if hit, dmg := g.playerAttack(m.ArmorClass); hit {
			g.FoeHP -= dmg
			ui.TypePrintln(ui.Ok("  " + ui.Icon("💥", fmt.Sprintf("Touche ! %d degats.", dmg))))
		} else {
			ui.TypePrintln(ui.Warn("  " + ui.Icon("💨", "Rate.")))
		}
		if g.FoeHP <= 0 {
			break
		}

		if hit, dmg := g.foeAttack(m); hit {
			g.Hero.HP -= dmg
			ui.TypePrintln(ui.Bad("  " + ui.Icon("💢", fmt.Sprintf("Tu prends %d degats.", dmg))))
		} else {
			ui.TypePrintln(ui.Accent("  " + ui.Icon("🛡️", "L'ennemi rate.")))
		}
	}

	if g.Hero.HP <= 0 {
		g.dieAndRespawn()
		return nil
	}

	cr := open5e.CRFloat(m.ChallengeRating)
	xp := int(cr*100) + 10
	gold := rand.Intn(10) + 1
	g.Gold += gold
	g.Slain[m.Name]++
	g.addXP(xp)
	ui.TypePrintln(ui.Ok(ui.Icon("🏆", fmt.Sprintf("Victoire ! +%d XP, +%d or.", xp, gold))))
	g.Encounter = nil
	g.FoeHP = 0
	return nil
}

// Flee tente de fuir l'ennemi (jet style DEX).
func (g *Game) Flee() error {
	if g.Encounter == nil {
		return fmt.Errorf("rien a fuir")
	}
	m := g.Encounter
	roll := rand.Intn(20) + 1
	ui.TypePrintln(ui.Icon("🏃", fmt.Sprintf("Tentative de fuite: d20 = %d (il faut >= 12)", roll)))
	if roll >= 12 {
		ui.TypePrintln(ui.Ok(ui.Icon("🏃", "Tu t'echappes !")))
		g.Encounter = nil
		g.FoeHP = 0
		return nil
	}

	ui.TypePrintln(ui.Bad(ui.Icon("😵", "Fuite ratee — l'ennemi frappe gratuitement !")))
	if hit, dmg := g.foeAttack(m); hit {
		g.Hero.HP -= dmg
		ui.TypePrintln(ui.Bad(ui.Icon("💢", fmt.Sprintf("Tu prends %d degats.", dmg))))
	} else {
		ui.TypePrintln(ui.Accent(ui.Icon("🛡️", "Mais il rate.")))
	}
	if g.Hero.HP <= 0 {
		g.dieAndRespawn()
	}
	return nil
}
