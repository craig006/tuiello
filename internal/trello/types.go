// Package trello provides types and a client for the Trello REST API.
package trello

// Board represents a Trello board with its lists.
type Board struct {
	ID    string
	Name  string
	URL   string
	Lists []List
}

// List represents a column on a Trello board.
type List struct {
	ID    string
	Name  string
	Pos   float64
	Cards []Card
}

// Card represents a card on a Trello board.
type Card struct {
	ID          string
	Name        string
	Description string
	Pos         float64
	URL         string
	Labels      []Label
	MemberIDs   []string
	ListID      string
}

// Label represents a Trello label.
type Label struct {
	ID    string
	Name  string
	Color string
}
