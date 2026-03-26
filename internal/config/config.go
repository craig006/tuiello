// internal/config/config.go
package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Auth           AuthConfig            `mapstructure:"auth"`
	GUI            GUIConfig             `mapstructure:"gui"`
	Board          BoardConfig           `mapstructure:"board"`
	Keybinding     KeybindingConfig      `mapstructure:"keybinding"`
	CustomCommands []CustomCommandConfig `mapstructure:"customCommands"`
	Views          []ViewConfig          `mapstructure:"views"`
}

type GUIConfig struct {
	Theme           ThemeConfig `mapstructure:"theme"`
	ColumnWidth     int         `mapstructure:"columnWidth"`
	ShowCardLabels  bool        `mapstructure:"showCardLabels"`
	ShowDetailPanel bool        `mapstructure:"showDetailPanel"`
	Padding         int         `mapstructure:"padding"`
}

type ThemeConfig struct {
	ActiveBorderColor   []string `mapstructure:"activeBorderColor"`
	InactiveBorderColor []string `mapstructure:"inactiveBorderColor"`
	SelectedCardColor   []string `mapstructure:"selectedCardColor"`
	ColumnTitleColor    []string `mapstructure:"columnTitleColor"`
}

type BoardConfig struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

type AuthConfig struct {
	APIKey string `mapstructure:"apiKey"`
	Token  string `mapstructure:"token"`
}

type KeybindingConfig struct {
	Universal UniversalKeys `mapstructure:"universal"`
	Board     BoardKeys     `mapstructure:"board"`
	Detail    DetailKeys    `mapstructure:"detail"`
	Filter    FilterKeys    `mapstructure:"filter"`
	Views     ViewKeys      `mapstructure:"views"`
}

type FilterKeys struct {
	Focus   string `mapstructure:"focus"`
	Members string `mapstructure:"members"`
	Labels  string `mapstructure:"labels"`
}

type UniversalKeys struct {
	Quit    string `mapstructure:"quit"`
	Help    string `mapstructure:"help"`
	Refresh string `mapstructure:"refresh"`
}

type BoardKeys struct {
	MoveLeft      string `mapstructure:"moveLeft"`
	MoveRight     string `mapstructure:"moveRight"`
	MoveUp        string `mapstructure:"moveUp"`
	MoveDown      string `mapstructure:"moveDown"`
	MoveCardLeft  string `mapstructure:"moveCardLeft"`
	MoveCardRight string `mapstructure:"moveCardRight"`
	MoveCardUp    string `mapstructure:"moveCardUp"`
	MoveCardDown  string `mapstructure:"moveCardDown"`
	OpenCard      string `mapstructure:"openCard"`
	CopyCardURL   string `mapstructure:"copyCardUrl"`
	Enter         string `mapstructure:"enter"`
	CustomCommand string `mapstructure:"customCommand"`
}

type DetailKeys struct {
	Toggle      string `mapstructure:"toggle"`
	TabPrev     string `mapstructure:"tabPrev"`
	TabNext     string `mapstructure:"tabNext"`
	ScrollDown  string `mapstructure:"scrollDown"`
	ScrollUp    string `mapstructure:"scrollUp"`
	FocusDetail string `mapstructure:"focusDetail"`
	FocusBoard  string `mapstructure:"focusBoard"`
}

type CustomCommandConfig struct {
	Key         string         `mapstructure:"key"`
	Description string         `mapstructure:"description"`
	Command     string         `mapstructure:"command"`
	Context     string         `mapstructure:"context"`
	Output      string         `mapstructure:"output"`
	Prompts     []PromptConfig `mapstructure:"prompts"`
}

type PromptConfig struct {
	Type    string         `mapstructure:"type"`
	Title   string         `mapstructure:"title"`
	Key     string         `mapstructure:"key"`
	Options []OptionConfig `mapstructure:"options"`
}

type OptionConfig struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

type ViewConfig struct {
	Title           string   `mapstructure:"title"`
	Filter          string   `mapstructure:"filter"`
	Key             string   `mapstructure:"key"`
	HideColumns     []string `mapstructure:"hideColumns"`
	ShowDetailPanel *bool    `mapstructure:"showDetailPanel"`
	ColumnWidth     *int     `mapstructure:"columnWidth"`
	ShowCardLabels  *bool    `mapstructure:"showCardLabels"`
}

type ViewKeys struct {
	NextView string `mapstructure:"nextView"`
	PrevView string `mapstructure:"prevView"`
}

func DefaultConfig() Config {
	return Config{
		GUI: GUIConfig{
			Theme: ThemeConfig{
				ActiveBorderColor:   []string{"4", "bold"},
				InactiveBorderColor: []string{"8"},
				SelectedCardColor:   []string{"6"},
				ColumnTitleColor:    []string{"5", "bold"},
			},
			ColumnWidth:     30,
			ShowCardLabels:  true,
			ShowDetailPanel: true,
			Padding:         1,
		},
		Keybinding: KeybindingConfig{
			Universal: UniversalKeys{
				Quit:    "q",
				Help:    "?",
				Refresh: "r",
			},
			Board: BoardKeys{
				MoveLeft:      "h",
				MoveRight:     "l",
				MoveUp:        "k",
				MoveDown:      "j",
				MoveCardLeft:  "H",
				MoveCardRight: "L",
				MoveCardUp:    "K",
				MoveCardDown:  "J",
				OpenCard:      "o",
				CopyCardURL:   "u",
				Enter:         "enter",
				CustomCommand: "x",
			},
			Detail: DetailKeys{
				Toggle:      "d",
				TabPrev:     "[",
				TabNext:     "]",
				ScrollDown:  "ctrl+j",
				ScrollUp:    "ctrl+k",
				FocusDetail: "enter",
				FocusBoard:  "esc",
			},
			Filter: FilterKeys{
				Focus:   "/",
				Members: "ctrl+m",
				Labels:  "ctrl+l",
			},
			Views: ViewKeys{
				NextView: "v",
				PrevView: "V",
			},
		},
		Views: []ViewConfig{
			{Title: "My Cards", Filter: "member:@me", Key: "m"},
			{Title: "All Cards"},
		},
	}
}

// AssignViewKeys assigns shortcut keys to views. Views with a custom Key
// keep it (first occurrence wins for duplicates). Views without a Key get
// auto-assigned incrementing numbers, skipping already-used keys.
func AssignViewKeys(views []ViewConfig) []string {
	used := map[string]bool{}
	keys := make([]string, len(views))
	for i, v := range views {
		if v.Key != "" && !used[v.Key] {
			keys[i] = v.Key
			used[v.Key] = true
		}
	}
	next := 1
	for i := range views {
		if keys[i] == "" {
			for used[strconv.Itoa(next)] {
				next++
			}
			keys[i] = strconv.Itoa(next)
			used[strconv.Itoa(next)] = true
			next++
		}
	}
	return keys
}

// Load reads config with cascade: globalDir/config.yml → globalDir/auth.yml →
// projectDir/.tuiello/config.yml → projectDir/.tuiello/auth.yml.
// Either path can be empty to skip that layer.
func Load(globalDir, projectDir string) (Config, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetConfigType("yaml")

	var files []string
	if globalDir != "" {
		files = append(files,
			filepath.Join(globalDir, "config.yml"),
			filepath.Join(globalDir, "auth.yml"),
		)
	}
	if projectDir != "" {
		files = append(files,
			filepath.Join(projectDir, ".tuiello", "config.yml"),
			filepath.Join(projectDir, ".tuiello", "auth.yml"),
		)
	}

	for _, f := range files {
		v.SetConfigFile(f)
		if err := v.MergeInConfig(); err != nil {
			if !os.IsNotExist(err) {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return cfg, err
				}
			}
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
