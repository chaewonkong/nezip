package main

import (
	"log"

	"github.com/chaewonkong/nezip/internal/cmd"
	_ "modernc.org/sqlite"
)

func main() {
	rootCmd := cmd.New(
		cmd.NewAnalyzeCmd(),
		cmd.NewSearchCmd(),
		cmd.NewMCPCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
