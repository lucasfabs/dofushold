package open5e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://api.open5e.com/v1/monsters/"

// Action = une attaque du monstre (champs utiles pour le combat).
type Action struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
	DamageDice  string `json:"damage_dice"`
	DamageBonus int    `json:"damage_bonus"`
}

type Monster struct {
	Slug            string   `json:"slug"`
	Name            string   `json:"name"`
	HitPoints       int      `json:"hit_points"`
	ArmorClass      int      `json:"armor_class"`
	ChallengeRating string   `json:"challenge_rating"`
	Actions         []Action `json:"actions"`
}

type pageResponse struct {
	Count    int       `json:"count"`
	Next     *string   `json:"next"`
	Previous *string   `json:"previous"`
	Results  []Monster `json:"results"`
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// Client appelle Open5e avec un cache TTL protege par mutex.
type Client struct {
	http       *http.Client
	mu         sync.Mutex
	cache      map[string]cacheEntry
	ttl        time.Duration
	stopReaper chan struct{}
}

func New(ttl time.Duration) *Client {
	c := &Client{
		http:       &http.Client{Timeout: 15 * time.Second},
		cache:      make(map[string]cacheEntry),
		ttl:        ttl,
		stopReaper: make(chan struct{}),
	}
	// Goroutine qui nettoie les entrees expirees toutes les minutes.
	go c.reap()
	return c
}

func (c *Client) Close() {
	close(c.stopReaper)
}

func (c *Client) reap() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-c.stopReaper:
			return
		case <-t.C:
			now := time.Now()
			c.mu.Lock()
			for k, e := range c.cache {
				if now.After(e.expiresAt) {
					delete(c.cache, k)
				}
			}
			c.mu.Unlock()
		}
	}
}

func (c *Client) getBytes(rawURL string) ([]byte, error) {
	c.mu.Lock()
	if e, ok := c.cache[rawURL]; ok && time.Now().Before(e.expiresAt) {
		data := e.data
		c.mu.Unlock()
		return data, nil
	}
	c.mu.Unlock()

	resp, err := c.http.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open5e status %d pour %s", resp.StatusCode, rawURL)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lecture body: %w", err)
	}

	c.mu.Lock()
	c.cache[rawURL] = cacheEntry{data: data, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
	return data, nil
}

// GetMonster charge un monstre par slug (ex: "goblin").
func (c *Client) GetMonster(slug string) (*Monster, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	rawURL := baseURL + url.PathEscape(slug) + "/"

	data, err := c.getBytes(rawURL)
	if err != nil {
		return nil, fmt.Errorf("monstre %s: %w", slug, err)
	}

	var m Monster
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode monstre %s: %w", slug, err)
	}
	return &m, nil
}

// ListPage liste une page de monstres (url vide = premiere page).
func (c *Client) ListPage(pageURL string) (monsters []Monster, next, prev string, err error) {
	if pageURL == "" {
		pageURL = baseURL + "?limit=10"
	}
	data, err := c.getBytes(pageURL)
	if err != nil {
		return nil, "", "", err
	}

	var page pageResponse
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, "", "", fmt.Errorf("decode page: %w", err)
	}

	next, prev = "", ""
	if page.Next != nil {
		next = *page.Next
	}
	if page.Previous != nil {
		prev = *page.Previous
	}
	return page.Results, next, prev, nil
}

// CRFloat convertit "1/4", "2", "0" en float pour comparer / XP.
func CRFloat(cr string) float64 {
	cr = strings.TrimSpace(cr)
	if cr == "" {
		return 0
	}
	if strings.Contains(cr, "/") {
		var a, b float64
		fmt.Sscanf(cr, "%f/%f", &a, &b)
		if b != 0 {
			return a / b
		}
		return 0
	}
	var v float64
	fmt.Sscanf(cr, "%f", &v)
	return v
}
