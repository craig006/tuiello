// cmd/root.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	boardFlag   string
	boardIDFlag string
)

var rootCmd = &cobra.Command{
	Use:   "tuillo",
	Short: "TUI client for Trello boards",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("tuillo — TUI Trello client")
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
