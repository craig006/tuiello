// internal/commands/custom_test.go
package commands

import (
	"testing"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

func TestRenderTemplate(t *testing.T) {
	ctx := TemplateContext{
		Card: CardContext{
			ID:   "card1",
			Name: "Fix login bug",
			URL:  "https://trello.com/c/card1",
		},
		List: ListContext{
			ID:   "list1",
			Name: "In Progress",
		},
		Board: BoardContext{
			ID:   "board1",
			Name: "My Board",
		},
	}

	result, err := RenderTemplate("open {{.Card.URL}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "open https://trello.com/c/card1" {
		t.Errorf("expected 'open https://trello.com/c/card1', got %q", result)
	}
}

func TestRenderTemplateKebab(t *testing.T) {
	ctx := TemplateContext{
		Card: CardContext{Name: "Fix Login Bug"},
	}

	result, err := RenderTemplate("git checkout -b {{.Card.Name | kebab}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "git checkout -b fix-login-bug" {
		t.Errorf("expected 'git checkout -b fix-login-bug', got %q", result)
	}
}

func TestRenderTemplateSnake(t *testing.T) {
	ctx := TemplateContext{
		Card: CardContext{Name: "Fix Login Bug"},
	}

	result, err := RenderTemplate("{{.Card.Name | snake}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "fix_login_bug" {
		t.Errorf("expected 'fix_login_bug', got %q", result)
	}
}

func TestRenderTemplateWithPrompt(t *testing.T) {
	ctx := TemplateContext{
		Card:   CardContext{ID: "card1"},
		Prompt: map[string]string{"Note": "This is a note"},
	}

	result, err := RenderTemplate("echo {{index .Prompt \"Note\"}} >> notes/{{.Card.ID}}.md", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "echo This is a note >> notes/card1.md" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestFilterCommandsByContext(t *testing.T) {
	cmds := []config.CustomCommandConfig{
		{Key: "g", Context: "card", Description: "Open in browser"},
		{Key: "n", Context: "board", Description: "New list"},
		{Key: "d", Context: "card", Description: "Delete card"},
	}

	filtered := FilterByContext(cmds, "card")
	if len(filtered) != 2 {
		t.Errorf("expected 2 card commands, got %d", len(filtered))
	}
}

func TestBuildContextFromCard(t *testing.T) {
	card := trello.Card{
		ID:   "c1",
		Name: "Test Card",
		URL:  "https://trello.com/c/c1",
		Labels: []trello.Label{
			{Name: "bug", Color: "red"},
			{Name: "urgent", Color: "orange"},
		},
		MemberIDs: []string{"m1", "m2"},
	}
	list := trello.List{ID: "l1", Name: "Todo"}
	board := trello.Board{ID: "b1", Name: "Project"}

	ctx := BuildContext(card, list, board)
	if ctx.Card.Labels != "bug,urgent" {
		t.Errorf("expected 'bug,urgent', got %q", ctx.Card.Labels)
	}
}
