package tui

import (
	"strings"
	"testing"

	"github.com/craig006/tuillo/internal/trello"
)

func TestParseFilterTextOnly(t *testing.T) {
	f := ParseFilter("fix door", "")
	if f.Text != "fix door" {
		t.Errorf("expected text 'fix door', got %q", f.Text)
	}
	if len(f.Members) != 0 {
		t.Errorf("expected no members, got %v", f.Members)
	}
	if len(f.Labels) != 0 {
		t.Errorf("expected no labels, got %v", f.Labels)
	}
}

func TestParseFilterMemberToken(t *testing.T) {
	f := ParseFilter("member:craig fix", "")
	if f.Text != "fix" {
		t.Errorf("expected text 'fix', got %q", f.Text)
	}
	if len(f.Members) != 1 || f.Members[0] != "craig" {
		t.Errorf("expected members [craig], got %v", f.Members)
	}
}

func TestParseFilterLabelToken(t *testing.T) {
	f := ParseFilter("label:Bug label:Design", "")
	if len(f.Labels) != 2 {
		t.Errorf("expected 2 labels, got %v", f.Labels)
	}
}

func TestParseFilterQuotedValue(t *testing.T) {
	f := ParseFilter(`member:"Craig Smith" fix`, "")
	if len(f.Members) != 1 || f.Members[0] != "Craig Smith" {
		t.Errorf("expected members [Craig Smith], got %v", f.Members)
	}
	if f.Text != "fix" {
		t.Errorf("expected text 'fix', got %q", f.Text)
	}
}

func TestParseFilterEmpty(t *testing.T) {
	f := ParseFilter("", "")
	if !f.IsEmpty() {
		t.Error("expected empty filter")
	}
}

func TestMatchesCardTextMatch(t *testing.T) {
	card := trello.Card{Name: "Fix the back door"}
	f := Filter{Text: "back"}
	if !f.MatchesCard(card) {
		t.Error("expected card to match text filter")
	}
}

func TestMatchesCardTextNoMatch(t *testing.T) {
	card := trello.Card{Name: "Fix the back door"}
	f := Filter{Text: "window"}
	if f.MatchesCard(card) {
		t.Error("expected card not to match text filter")
	}
}

func TestMatchesCardMemberMatch(t *testing.T) {
	card := trello.Card{
		Members: []trello.Member{{Username: "craig", FullName: "Craig Smith"}},
	}
	f := Filter{Members: []string{"craig"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match member filter")
	}
}

func TestMatchesCardMemberByFullName(t *testing.T) {
	card := trello.Card{
		Members: []trello.Member{{Username: "craig006", FullName: "Craig Smith"}},
	}
	f := Filter{Members: []string{"Craig Smith"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match member by full name")
	}
}

func TestMatchesCardLabelMatch(t *testing.T) {
	card := trello.Card{
		Labels: []trello.Label{{Name: "Bug"}},
	}
	f := Filter{Labels: []string{"bug"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match label filter (case-insensitive)")
	}
}

func TestMatchesCardAndLogic(t *testing.T) {
	card := trello.Card{
		Name:    "Fix login bug",
		Members: []trello.Member{{Username: "craig"}},
		Labels:  []trello.Label{{Name: "Bug"}},
	}
	f := Filter{Text: "login", Members: []string{"craig"}, Labels: []string{"Bug"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match all filters")
	}
}

func TestMatchesCardAndLogicFail(t *testing.T) {
	card := trello.Card{
		Name:    "Fix login bug",
		Members: []trello.Member{{Username: "craig"}},
	}
	// Card has no labels, so label filter should fail
	f := Filter{Text: "login", Members: []string{"craig"}, Labels: []string{"Bug"}}
	if f.MatchesCard(card) {
		t.Error("expected card not to match — missing label")
	}
}

func TestMatchesCardEmptyFilter(t *testing.T) {
	card := trello.Card{Name: "Anything"}
	f := Filter{}
	if !f.MatchesCard(card) {
		t.Error("empty filter should match all cards")
	}
}

func TestBuildFilterText(t *testing.T) {
	f := Filter{Text: "fix", Members: []string{"craig"}, Labels: []string{"Bug"}}
	result := BuildFilterText(f)
	if !strings.Contains(result, "member:craig") {
		t.Errorf("expected member token in %q", result)
	}
	if !strings.Contains(result, "label:Bug") {
		t.Errorf("expected label token in %q", result)
	}
	if !strings.Contains(result, "fix") {
		t.Errorf("expected text in %q", result)
	}
}

func TestBuildFilterTextQuotesSpaces(t *testing.T) {
	f := Filter{Members: []string{"Craig Smith"}}
	result := BuildFilterText(f)
	if !strings.Contains(result, `member:"Craig Smith"`) {
		t.Errorf("expected quoted member in %q", result)
	}
}

func TestParseFilterAtMe(t *testing.T) {
	f := ParseFilter("member:@me fix", "craig")
	if len(f.Members) != 1 || f.Members[0] != "craig" {
		t.Errorf("expected @me resolved to 'craig', got %v", f.Members)
	}
	if f.Text != "fix" {
		t.Errorf("expected text 'fix', got %q", f.Text)
	}
}

func TestParseFilterAtMeEmpty(t *testing.T) {
	f := ParseFilter("member:@me", "")
	if len(f.Members) != 1 || f.Members[0] != "@me" {
		t.Errorf("expected literal '@me' when no user, got %v", f.Members)
	}
}
