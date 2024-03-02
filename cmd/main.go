package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".local", "cacidy")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return "", err
		}
	}
	return path, nil
}

var rootCmd = &cli.Command{
	Name: "cacidy",
	Commands: []*cli.Command{
		runCommand,
		listenCommand,
	},
}

func main() {
	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		fmt.Printf("%s %s\n", errorStyle.Render("ERRO"), err)
	}
}
