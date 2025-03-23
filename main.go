package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "pull",
				Usage: "pull config from remote.",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("pulled")

					return nil
				},
			},
			{
				Name: "add",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "repo",
						Usage: "the repository of which the configuration file lives in, either in the root or in a sub directory.",
					},
				},
				Usage: "add new config.",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					conf := cmd.Args().Get(0)
					if len(conf) == 0 {
						fmt.Println(" qac")
					}

					// conf := cmd.Args().Get(0)

					return nil
				},
			},
		},
	}
	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
