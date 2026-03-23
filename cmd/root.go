// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/craig006/tuillo/internal/config"
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
		fmt.Printf("Board ID: %q, Board Name: %q\n", appConfig.Board.ID, appConfig.Board.Name)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&boardFlag, "board", "", "Trello board name")
	rootCmd.PersistentFlags().StringVar(&boardIDFlag, "board-id", "", "Trello board ID")
}

func Execute() error {
	return rootCmd.Execute()
}
