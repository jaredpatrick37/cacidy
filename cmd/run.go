package main

import (
	"context"

	"cacidy/pkg/runner"

	"github.com/urfave/cli/v3"
)

var runCommand = &cli.Command{
	Name:  "run",
	Usage: "run a ci function",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "source",
			Usage:    "target application source code",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "module",
			Usage: "ci pipeline module",
		},
		&cli.StringFlag{
			Name:  "function",
			Usage: "ci pipeline module function",
		},
	},
	Action: func(_ context.Context, c *cli.Command) error {
		return runner.RunPipeline(
			c.String("source"),
			c.String("module"),
			c.String("function"),
		)
	},
}
