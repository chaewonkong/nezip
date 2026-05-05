package main

import (
	"log"

	"github.com/chaewonkong/nezip/internal/cmd"
	_ "modernc.org/sqlite"
)

var version = "dev"

func main() {
	rootCmd := cmd.New(
		version,
		cmd.NewAnalyzeCmd(),
		cmd.NewSearchCmd(),
		cmd.NewMCPCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
