package main

import (
	"cacidy/pkg/runner"
	"context"

	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"
)

var listenCommand = &cli.Command{
	Name:  "listen",
	Usage: "listen for changes to a git repository and call a pipeline function",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "branch",
			Usage: "git repository branch to listen on",
			Value: "master",
		},
		&cli.StringFlag{
			Name:  "ssh-key-file",
			Usage: "git repository private key path",
		},
		&cli.StringFlag{
			Name:  "username",
			Usage: "git repository username",
		},
		&cli.StringFlag{
			Name:  "password",
			Usage: "git repository password",
		},
	},
	ArgsUsage: "[url]",
	Action: func(_ context.Context, c *cli.Command) error {
		dataDir, err := dataDir()
		if err != nil {
			return err
		}
		if err := runner.Listen(dataDir, &runner.Repository{
			URL:            c.Args().First(),
			Ref:            c.String("branch"),
			PrivateKeyFile: c.String("ssh-key-file"),
			Username:       c.String("username"),
			Password:       c.String("password"),
		}); err != nil {
			log.Fatal(err)
		}
		return nil
	},
}
