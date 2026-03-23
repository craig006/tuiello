// main.go
package main

import (
	"os"

	"github.com/craig006/tuillo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
