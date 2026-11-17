package world

import (
	"encoding/json"
	"fmt"
	"os"
)

// Region = une zone du monde (id, dangers, sorties, ennemis).
type Region struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Danger      int      `json:"danger"`
	Foes        []string `json:"foes"`
	Exits       []string `json:"exits"`
}

type World struct {
	Start   string   `json:"start"`
	Regions []Region `json:"regions"`
	byID    map[string]*Region
}

// Load lit world.json et indexe les regions par id.
func Load(path string) (*World, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture monde: %w", err)
	}

	var w World
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("json monde: %w", err)
	}

	w.byID = make(map[string]*Region, len(w.Regions))
	for i := range w.Regions {
		r := &w.Regions[i]
		w.byID[r.ID] = r
	}

	if _, ok := w.byID[w.Start]; !ok {
		return nil, fmt.Errorf("region de depart inconnue: %s", w.Start)
	}
	return &w, nil
}

func (w *World) Get(id string) (*Region, bool) {
	r, ok := w.byID[id]
	return r, ok
}

// CanTravel verifie que to est une sortie de from.
func (w *World) CanTravel(from, to string) bool {
	r, ok := w.byID[from]
	if !ok {
		return false
	}
	for _, e := range r.Exits {
		if e == to {
			return true
		}
	}
	return false
}
