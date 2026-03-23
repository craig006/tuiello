package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	URL          string           `json:"url"`
	Lists        []apiList        `json:"lists"`
	Cards        []apiCard        `json:"cards"`
	Members      []apiMember      `json:"members"`
	CustomFields []apiCustomField `json:"customFields"`
}

type apiList struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Pos  float64 `json:"pos"`
}

type apiCard struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Desc             string              `json:"desc"`
	Pos              float64             `json:"pos"`
	URL              string              `json:"url"`
	IDList           string              `json:"idList"`
	IDMembers        []string            `json:"idMembers"`
	IDLabels         []string            `json:"idLabels"`
	Labels           []apiLabel          `json:"labels"`
	Badges           apiBadges           `json:"badges"`
	CustomFieldItems []apiCustomFieldItem `json:"customFieldItems"`
}

type apiBadges struct {
	Comments int `json:"comments"`
}

type apiLabel struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type apiMember struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Initials string `json:"initials"`
	Username string `json:"username"`
}

type apiCustomField struct {
	ID      string                `json:"id"`
	Name    string                `json:"name"`
	Type    string                `json:"type"`
	Options []apiCustomFieldOption `json:"options"`
}

type apiCustomFieldOption struct {
	ID    string                    `json:"id"`
	Value apiCustomFieldOptionValue `json:"value"`
	Color string                    `json:"color"`
}

type apiCustomFieldOptionValue struct {
	Text string `json:"text"`
}

type apiCustomFieldItem struct {
	ID            string                     `json:"id"`
	IDCustomField string                     `json:"idCustomField"`
	Value         map[string]string          `json:"value"`
	IDValue       string                     `json:"idValue"`
}

// FetchBoard retrieves a board with its open lists and cards.
func (c *Client) FetchBoard(boardID string) (*Board, error) {
	var ab apiBoard
	path := fmt.Sprintf("/1/boards/%s?lists=open&cards=open&card_fields=name,desc,labels,idMembers,url,pos,idList,badges&card_customFieldItems=true&list_fields=name,pos&members=all&member_fields=id,fullName,initials,username&customFields=true", boardID)
	if err := c.get(path, &ab); err != nil {
		return nil, err
	}

	// Build member lookup
	memberMap := make(map[string]Member)
	var boardMembers []Member
	for _, am := range ab.Members {
		m := Member{ID: am.ID, FullName: am.FullName, Initials: am.Initials, Username: am.Username}
		memberMap[am.ID] = m
		boardMembers = append(boardMembers, m)
	}

	// Build custom field definitions lookup
	var boardCFDefs []CustomFieldDef
	cfDefMap := make(map[string]CustomFieldDef)
	// option ID -> (text, color)
	cfOptionMap := make(map[string]CustomFieldOption)
	for _, acf := range ab.CustomFields {
		def := CustomFieldDef{ID: acf.ID, Name: acf.Name, Type: acf.Type}
		for _, opt := range acf.Options {
			o := CustomFieldOption{ID: opt.ID, Text: opt.Value.Text, Color: opt.Color}
			def.Options = append(def.Options, o)
			cfOptionMap[opt.ID] = o
		}
		cfDefMap[acf.ID] = def
		boardCFDefs = append(boardCFDefs, def)
	}

	// Group cards by list ID
	cardsByList := make(map[string][]Card)
	for _, ac := range ab.Cards {
		card := Card{
			ID:           ac.ID,
			Name:         ac.Name,
			Description:  ac.Desc,
			Pos:          ac.Pos,
			URL:          ac.URL,
			MemberIDs:    ac.IDMembers,
			ListID:       ac.IDList,
			CommentCount: ac.Badges.Comments,
		}
		for _, lbl := range ac.Labels {
			card.Labels = append(card.Labels, Label{ID: lbl.ID, Name: lbl.Name, Color: lbl.Color})
		}
		// Resolve members
		for _, mid := range ac.IDMembers {
			if m, ok := memberMap[mid]; ok {
				card.Members = append(card.Members, m)
			}
		}
		// Resolve custom field values
		for _, cfi := range ac.CustomFieldItems {
			def, ok := cfDefMap[cfi.IDCustomField]
			if !ok {
				continue
			}
			cfv := CustomFieldValue{FieldName: def.Name}
			if cfi.IDValue != "" {
				// List-type field: resolve option
				if opt, ok := cfOptionMap[cfi.IDValue]; ok {
					cfv.Value = opt.Text
					cfv.Color = opt.Color
				}
			} else if cfi.Value != nil {
				// Other types: grab first value
				for _, v := range cfi.Value {
					cfv.Value = v
					break
				}
			}
			if cfv.Value != "" {
				card.CustomFields = append(card.CustomFields, cfv)
			}
		}
		cardsByList[ac.IDList] = append(cardsByList[ac.IDList], card)
	}

	board := &Board{ID: ab.ID, Name: ab.Name, URL: ab.URL, Members: boardMembers, CustomFields: boardCFDefs}
	for _, al := range ab.Lists {
		list := List{ID: al.ID, Name: al.Name, Pos: al.Pos, Cards: cardsByList[al.ID]}
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

// API response types for card actions (comments)
type apiAction struct {
	ID            string        `json:"id"`
	Date          string        `json:"date"`
	Data          apiActionData `json:"data"`
	MemberCreator apiMember     `json:"memberCreator"`
}

type apiActionData struct {
	Text string `json:"text"`
}

// FetchCardComments retrieves comments on a card.
func (c *Client) FetchCardComments(cardID string) ([]Comment, error) {
	var actions []apiAction
	path := fmt.Sprintf("/1/cards/%s/actions?filter=commentCard&fields=data,date,idMemberCreator,memberCreator&memberCreator_fields=fullName,initials,username", cardID)
	if err := c.get(path, &actions); err != nil {
		return nil, err
	}

	comments := make([]Comment, 0, len(actions))
	for _, a := range actions {
		t, err := time.Parse(time.RFC3339, a.Date)
		if err != nil {
			t = time.Time{}
		}
		comments = append(comments, Comment{
			ID:   a.ID,
			Body: a.Data.Text,
			Date: t,
			Author: Member{
				ID:       a.MemberCreator.ID,
				FullName: a.MemberCreator.FullName,
				Initials: a.MemberCreator.Initials,
				Username: a.MemberCreator.Username,
			},
		})
	}
	return comments, nil
}

// API response types for checklists
type apiChecklist struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	CheckItems []apiCheckItem `json:"checkItems"`
}

type apiCheckItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// FetchCardChecklists retrieves checklists for a card.
func (c *Client) FetchCardChecklists(cardID string) ([]Checklist, error) {
	var raw []apiChecklist
	path := fmt.Sprintf("/1/cards/%s/checklists?fields=name&checkItem_fields=name,state", cardID)
	if err := c.get(path, &raw); err != nil {
		return nil, err
	}

	checklists := make([]Checklist, 0, len(raw))
	for _, cl := range raw {
		checklist := Checklist{ID: cl.ID, Name: cl.Name}
		for _, ci := range cl.CheckItems {
			checklist.Items = append(checklist.Items, CheckItem{
				ID:       ci.ID,
				Name:     ci.Name,
				Complete: ci.State == "complete",
			})
		}
		checklists = append(checklists, checklist)
	}
	return checklists, nil
}
