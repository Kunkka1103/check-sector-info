package main

import (
	"log"
	"os"
	"rebuild/internal"
	"rebuild/pkg/errutil"
	"rebuild/pkg/version"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "rebuild-tool",
		Version: strings.TrimPrefix(version.GetVersion(), "v"),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "chain",
				Value:    "http://127.0.0.1:3453/",
				EnvVars:  []string{"CHAIN"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "token",
				Value:   "",
				EnvVars: []string{"TOKEN"},
			},
		},
		Commands: []*cli.Command{
			internal.RebuildInfoCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		errutil.HandleExitCoder(err)
		log.Fatal(err)
	}
}
