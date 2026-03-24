// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
	"github.com/craig006/tuiello/internal/tui"
	"github.com/spf13/cobra"
)

var (
	boardFlag   string
	boardIDFlag string
	appConfig   config.Config
)

var rootCmd = &cobra.Command{
	Use:   "tuiello",
	Short: "TUI client for Trello boards",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		globalDir, err := os.UserConfigDir()
		if err != nil {
			globalDir = ""
		} else {
			globalDir = globalDir + "/tuiello"
		}

		cwd, _ := os.Getwd()
		appConfig, err = config.Load(globalDir, cwd)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// CLI flags override config
		if boardIDFlag != "" {
			appConfig.Board.ID = boardIDFlag
		}
		if boardFlag != "" {
			appConfig.Board.Name = boardFlag
		}

		// Env vars override config auth
		if envKey := os.Getenv("TRELLO_API_KEY"); envKey != "" {
			appConfig.Auth.APIKey = envKey
		}
		if envToken := os.Getenv("TRELLO_TOKEN"); envToken != "" {
			appConfig.Auth.Token = envToken
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if appConfig.Auth.APIKey == "" || appConfig.Auth.Token == "" {
			return fmt.Errorf("missing Trello credentials.\n\n" +
				"Set credentials in ~/.config/tuiello/auth.yml:\n" +
				"  auth:\n" +
				"    apiKey: <your-api-key>\n" +
				"    token: <your-token>\n\n" +
				"Or set environment variables:\n" +
				"  export TRELLO_API_KEY=<your-api-key>\n" +
				"  export TRELLO_TOKEN=<your-token>\n\n" +
				"Get your API key at: https://trello.com/power-ups/admin")
		}

		client := trello.NewClient(appConfig.Auth.APIKey, appConfig.Auth.Token)

		if err := client.ValidateCredentials(); err != nil {
			return fmt.Errorf("invalid credentials: %w", err)
		}

		app := tui.NewApp(client, appConfig)
		p := tea.NewProgram(app)
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&boardFlag, "board", "", "Trello board name")
	rootCmd.PersistentFlags().StringVar(&boardIDFlag, "board-id", "", "Trello board ID")
}

func Execute() error {
	return rootCmd.Execute()
}
