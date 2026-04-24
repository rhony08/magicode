package main

import (
	"fmt"
	"os"

	"github.com/rhony08/magicode/cmd/magicode/commands"
)

var (
	// Version is set at build time
	Version = "dev"
)

func main() {
	rootCmd := commands.NewRootCommand(Version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}