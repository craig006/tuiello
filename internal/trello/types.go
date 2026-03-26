// Package trello provides types and a client for the Trello REST API.
package trello

import "time"

// Board represents a Trello board with its lists.
type Board struct {
	ID           string
	Name         string
	URL          string
	Lists        []List
	Members      []Member
	CustomFields []CustomFieldDef
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
	ID           string
	Name         string
	Description  string
	Pos          float64
	URL          string
	Labels       []Label
	MemberIDs    []string
	ListID       string
	CommentCount      int
	CheckItemCount    int
	CheckItemsChecked int
	Members           []Member
	CustomFields      []CustomFieldValue
}

// Label represents a Trello label.
type Label struct {
	ID    string
	Name  string
	Color string
}

// Member represents a Trello board member.
type Member struct {
	ID       string
	FullName string
	Initials string
	Username string
}

// CustomFieldDef represents a custom field definition on a board.
type CustomFieldDef struct {
	ID      string
	Name    string
	Type    string // "text", "number", "date", "checkbox", "list"
	Options []CustomFieldOption
}

// CustomFieldOption represents an option for a list-type custom field.
type CustomFieldOption struct {
	ID    string
	Text  string
	Color string
}

// CustomFieldValue represents a custom field value on a card.
type CustomFieldValue struct {
	FieldName string
	Value     string
	Color     string // set for list-type fields with a color
}

// Comment represents a comment on a Trello card.
type Comment struct {
	ID       string
	Author   Member
	Body     string
	Date     time.Time
	Editable bool // Can user edit/delete this comment?
}

// Checklist represents a checklist on a Trello card.
type Checklist struct {
	ID    string
	Name  string
	Items []CheckItem
}

// CheckItem represents a single item in a checklist.
type CheckItem struct {
	ID       string
	Name     string
	Complete bool
}
