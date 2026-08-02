package rgprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"rg-logs/internal/wow"
)

const (
	defaultBaseURL = "https://db.rising-gods.de"
	userAgent      = "rg-logs/1.0 (+character-gearscore)"
)

var (
	profileIDRe = regexp.MustCompile(`initialize\(\s*'[^']*'\s*,\s*\{\s*id\s*:\s*(\d+)\s*\}`)
	itemAddRe   = regexp.MustCompile(`g_items\.add\((\d+)\s*,`)
	classIDRe   = regexp.MustCompile(`"classs"\s*:\s*(\d+)`)
)

// Client fetches Rising Gods (AoWow) character profiles.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// ProfileURL builds the public armory URL for a character name.
func ProfileURL(name string) string {
	return defaultBaseURL + "/?profile=eu.rising-gods." + NormalizeName(name)
}

// NormalizeName strips a realm suffix and lowercases for AoWow URLs.
func NormalizeName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.IndexByte(name, '-'); i > 0 {
		name = name[:i]
	}
	return strings.ToLower(name)
}

// Character holds parsed gear needed for GearScoreLite.
type Character struct {
	Name      string
	Class     wow.Class
	ClassID   int
	Inventory map[int]wow.EquippedItem // inv slot → item
	GearScore int
}

// Fetch loads profile gear and computes GearScoreLite.
// Returns (nil, nil) when the profile is missing.
func (c *Client) Fetch(ctx context.Context, characterName string) (*Character, error) {
	name := NormalizeName(characterName)
	if name == "" {
		return nil, nil
	}

	id, err := c.resolveProfileID(ctx, name)
	if err != nil {
		return nil, err
	}
	if id == 0 {
		return nil, nil
	}

	body, err := c.get(ctx, fmt.Sprintf("?profile=load&id=%d&%d", id, time.Now().UnixMilli()))
	if err != nil {
		return nil, err
	}

	ch, err := parseLoadPayload(body)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, nil
	}
	ch.Name = name
	ch.GearScore = wow.GearScore(ch.Inventory, ch.Class)
	return ch, nil
}

func (c *Client) resolveProfileID(ctx context.Context, name string) (int64, error) {
	body, err := c.get(ctx, "?profile=eu.rising-gods."+name)
	if err != nil {
		return 0, err
	}
	m := profileIDRe.FindSubmatch(body)
	if m == nil {
		return 0, nil
	}
	return strconv.ParseInt(string(m[1]), 10, 64)
}

func (c *Client) get(ctx context.Context, pathQuery string) ([]byte, error) {
	base := c.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	url := strings.TrimRight(base, "/") + "/"
	if strings.HasPrefix(pathQuery, "?") {
		url = strings.TrimRight(base, "/") + "/" + pathQuery
	} else {
		url += strings.TrimLeft(pathQuery, "/")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/javascript,*/*")

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rising-gods HTTP %d for %s", res.StatusCode, pathQuery)
	}
	return body, nil
}

type itemMeta struct {
	Quality   int `json:"quality"`
	JSONEquip struct {
		Level   int `json:"level"`
		SlotBak int `json:"slotbak"`
		Slot    int `json:"slot"`
	} `json:"jsonequip"`
}

func parseLoadPayload(body []byte) (*Character, error) {
	items := make(map[int]itemMeta)
	for _, m := range itemAddRe.FindAllSubmatchIndex(body, -1) {
		// m[2]:m[3] = item id; object starts after the match end
		id, err := strconv.Atoi(string(body[m[2]:m[3]]))
		if err != nil {
			continue
		}
		start := m[1]
		for start < len(body) && (body[start] == ' ' || body[start] == '\t' || body[start] == '\n') {
			start++
		}
		if start >= len(body) || body[start] != '{' {
			continue
		}
		end, err := matchBraces(body, start)
		if err != nil {
			continue
		}
		var meta itemMeta
		if err := json.Unmarshal(body[start:end+1], &meta); err != nil {
			continue
		}
		items[id] = meta
	}

	invRaw, classID, ok := extractInventoryAndClass(body)
	if !ok {
		return nil, nil
	}

	var invMap map[string][]int
	if err := json.Unmarshal(invRaw, &invMap); err != nil {
		return nil, fmt.Errorf("parse inventory: %w", err)
	}

	ch := &Character{
		ClassID:   classID,
		Class:     wow.ClassFromWoWClassID(classID),
		Inventory: make(map[int]wow.EquippedItem, len(invMap)),
	}

	for slotStr, arr := range invMap {
		if len(arr) < 1 || arr[0] == 0 {
			continue
		}
		slot, err := strconv.Atoi(slotStr)
		if err != nil {
			continue
		}
		meta, found := items[arr[0]]
		if !found {
			continue
		}
		slotBak := meta.JSONEquip.SlotBak
		if slotBak == 0 {
			slotBak = meta.JSONEquip.Slot
		}
		ilvl := meta.JSONEquip.Level
		if ilvl <= 0 {
			continue
		}
		ch.Inventory[slot] = wow.EquippedItem{
			ItemLevel: ilvl,
			Quality:   meta.Quality,
			SlotBak:   slotBak,
		}
	}
	return ch, nil
}

func extractInventoryAndClass(body []byte) (invJSON []byte, classID int, ok bool) {
	// Prefer the registerProfile payload — item blobs also contain "classs" (armor class).
	profileStart := strings.Index(string(body), "registerProfile(")
	search := body
	if profileStart >= 0 {
		search = body[profileStart:]
	}

	const invKey = `"inventory":`
	idx := strings.Index(string(search), invKey)
	if idx < 0 {
		return nil, 0, false
	}
	absInv := 0
	if profileStart >= 0 {
		absInv = profileStart + idx
	} else {
		absInv = idx
	}
	start := absInv + len(invKey)
	for start < len(body) && body[start] != '{' {
		start++
	}
	if start >= len(body) {
		return nil, 0, false
	}
	end, err := matchBraces(body, start)
	if err != nil {
		return nil, 0, false
	}
	invJSON = body[start : end+1]

	// Character classs sits on the profile object before inventory.
	prefix := search[:idx]
	if m := classIDRe.FindSubmatch(prefix); m != nil {
		classID, _ = strconv.Atoi(string(m[1]))
	}
	return invJSON, classID, true
}

func matchBraces(b []byte, start int) (int, error) {
	if start >= len(b) || b[start] != '{' {
		return 0, fmt.Errorf("not an object")
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(b); i++ {
		c := b[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unbalanced braces")
}
