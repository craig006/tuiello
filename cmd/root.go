// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
	"github.com/craig006/tuillo/internal/tui"
	"github.com/spf13/cobra"
)

var (
	boardFlag   string
	boardIDFlag string
	appConfig   config.Config
)

var rootCmd = &cobra.Command{
	Use:   "tuillo",
	Short: "TUI client for Trello boards",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		globalDir, err := os.UserConfigDir()
		if err != nil {
			globalDir = ""
		} else {
			globalDir = globalDir + "/tuillo"
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

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := os.Getenv("TRELLO_API_KEY")
		token := os.Getenv("TRELLO_TOKEN")

		if apiKey == "" || token == "" {
			return fmt.Errorf("missing Trello credentials.\n\n" +
				"Set these environment variables:\n" +
				"  export TRELLO_API_KEY=<your-api-key>\n" +
				"  export TRELLO_TOKEN=<your-token>\n\n" +
				"Get your API key at: https://trello.com/power-ups/admin\n" +
				"Then authorize a token at:\n" +
				"  https://trello.com/1/authorize?expiration=never&scope=read,write&response_type=token&key=<YOUR_KEY>")
		}

		client := trello.NewClient(apiKey, token)

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
