// internal/commands/custom.go
package commands

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

type CardContext struct {
	ID          string
	Name        string
	URL         string
	Description string
	Labels      string
	Members     string
}

type ListContext struct {
	ID   string
	Name string
}

type BoardContext struct {
	ID   string
	Name string
}

type TemplateContext struct {
	Card   CardContext
	List   ListContext
	Board  BoardContext
	Prompt map[string]string
}

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

var funcMap = template.FuncMap{
	"kebab": func(s string) string {
		return strings.Trim(nonAlphaNum.ReplaceAllString(strings.ToLower(s), "-"), "-")
	},
	"snake": func(s string) string {
		return strings.Trim(nonAlphaNum.ReplaceAllString(strings.ToLower(s), "_"), "_")
	},
	"camel": func(s string) string {
		parts := nonAlphaNum.Split(strings.ToLower(s), -1)
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) > 0 {
				parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
		return strings.Join(parts, "")
	},
	"lower":   strings.ToLower,
	"upper":   strings.ToUpper,
	"trim":    strings.TrimSpace,
	"replace": func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
}

func RenderTemplate(tmplStr string, ctx TemplateContext) (string, error) {
	tmpl, err := template.New("cmd").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("invalid template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

func FilterByContext(cmds []config.CustomCommandConfig, context string) []config.CustomCommandConfig {
	var result []config.CustomCommandConfig
	for _, cmd := range cmds {
		if cmd.Context == context {
			result = append(result, cmd)
		}
	}
	return result
}

func BuildContext(card trello.Card, list trello.List, board trello.Board) TemplateContext {
	var labelNames []string
	for _, l := range card.Labels {
		labelNames = append(labelNames, l.Name)
	}

	return TemplateContext{
		Card: CardContext{
			ID:          card.ID,
			Name:        card.Name,
			URL:         card.URL,
			Description: card.Description,
			Labels:      strings.Join(labelNames, ","),
			Members:     strings.Join(card.MemberIDs, ","),
		},
		List: ListContext{
			ID:   list.ID,
			Name: list.Name,
		},
		Board: BoardContext{
			ID:   board.ID,
			Name: board.Name,
		},
		Prompt: make(map[string]string),
	}
}

// ExecuteSilent runs a command and returns stdout+stderr.
func ExecuteSilent(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ExecuteTerminal returns the exec.Cmd so the caller can run it with full terminal access.
func ExecuteTerminal(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
