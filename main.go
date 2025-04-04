package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	reconConfig "github.com/jzho987/recon/config"
	reconSync "github.com/jzho987/recon/sync"
)

func main() {
	syncComamnd := reconSync.NewSyncCommand()
	configComamnd := reconConfig.NewConfigCommand()

	cmd := &cli.Command{
		Usage: "Simple but deadly dotfile manager.",
		Commands: []*cli.Command{
			&syncComamnd,
			&configComamnd,
		},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
