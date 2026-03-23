package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client is an HTTP client for the Trello REST API.
type Client struct {
	BaseURL    string
	apiKey     string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Trello API client with the given credentials.
func NewClient(apiKey, token string) *Client {
	return &Client{
		BaseURL:    "https://api.trello.com",
		apiKey:     apiKey,
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *Client) get(path string, target interface{}) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("key", c.apiKey)
	q.Set("token", c.token)
	u.RawQuery = q.Encode()
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: check your TRELLO_API_KEY and TRELLO_TOKEN")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited by Trello API, please try again shortly")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) put(path string, form url.Values) error {
	form.Set("key", c.apiKey)
	form.Set("token", c.token)
	req, err := http.NewRequest(http.MethodPut, c.BaseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: check your TRELLO_API_KEY and TRELLO_TOKEN")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// ValidateCredentials checks that the API key and token are valid.
func (c *Client) ValidateCredentials() error {
	var result map[string]interface{}
	return c.get("/1/members/me", &result)
}

type apiBoard struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	URL   string    `json:"url"`
	Lists []apiList `json:"lists"`
}

type apiList struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Pos   float64   `json:"pos"`
	Cards []apiCard `json:"cards"`
}

type apiCard struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Desc      string     `json:"desc"`
	Pos       float64    `json:"pos"`
	URL       string     `json:"url"`
	IDList    string     `json:"idList"`
	IDMembers []string   `json:"idMembers"`
	IDLabels  []string   `json:"idLabels"`
	Labels    []apiLabel `json:"labels"`
}

type apiLabel struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// FetchBoard retrieves a board with its open lists and cards.
func (c *Client) FetchBoard(boardID string) (*Board, error) {
	var ab apiBoard
	path := fmt.Sprintf("/1/boards/%s?lists=open&cards=open&card_fields=name,desc,labels,idMembers,url,pos,idList&list_fields=name,pos", boardID)
	if err := c.get(path, &ab); err != nil {
		return nil, err
	}
	board := &Board{ID: ab.ID, Name: ab.Name, URL: ab.URL}
	for _, al := range ab.Lists {
		list := List{ID: al.ID, Name: al.Name, Pos: al.Pos}
		for _, ac := range al.Cards {
			card := Card{
				ID:          ac.ID,
				Name:        ac.Name,
				Description: ac.Desc,
				Pos:         ac.Pos,
				URL:         ac.URL,
				MemberIDs:   ac.IDMembers,
				ListID:      ac.IDList,
			}
			for _, lbl := range ac.Labels {
				card.Labels = append(card.Labels, Label{ID: lbl.ID, Name: lbl.Name, Color: lbl.Color})
			}
			list.Cards = append(list.Cards, card)
		}
		board.Lists = append(board.Lists, list)
	}
	return board, nil
}

// ResolveBoard looks up a board ID by name (case-insensitive). Errors if 0 or multiple matches.
func (c *Client) ResolveBoard(name string) (string, error) {
	var boards []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.get("/1/members/me/boards", &boards); err != nil {
		return "", err
	}
	var matches []string
	for _, b := range boards {
		if strings.EqualFold(b.Name, name) {
			matches = append(matches, b.ID)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no board found with name %q", name)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple boards match %q (IDs: %s) — use --board-id to specify", name, strings.Join(matches, ", "))
	}
}

// MoveCardToList moves a card to a different list at the given position.
func (c *Client) MoveCardToList(cardID, listID, pos string) error {
	form := url.Values{}
	form.Set("idList", listID)
	form.Set("pos", pos)
	return c.put(fmt.Sprintf("/1/cards/%s", cardID), form)
}

// ReorderCard updates the position of a card within its current list.
func (c *Client) ReorderCard(cardID string, pos float64) error {
	form := url.Values{}
	form.Set("pos", fmt.Sprintf("%f", pos))
	return c.put(fmt.Sprintf("/1/cards/%s", cardID), form)
}
