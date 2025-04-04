package config

import (
	"context"
	"os"
	"path"

	"github.com/davecgh/go-spew/spew"
	"github.com/pelletier/go-toml/v2"
	"github.com/urfave/cli"
)

const (
	// main
	RECON_CONFIG_FILE = ".config/recon/recon.toml"

	// defaults
	DEFAULT_CLONE_DIR = ".config/recon/git-dirs/"
)

// All the config options for recon.
type ReconConfig struct {
	// Location to clone git repositories
	CloneDir string `toml:"clone_dir"`

	// List of repositories to sync with
	Repos []RepoConfig `toml:"repos"`
}

type RepoConfig struct {
	// Required: name of the config. e.g. "tmux".
	Name string `toml:"name"`

	// Required: url of the repository for this config.
	Remote string `toml:"remote"`

	// Optional: tagged version to use. default to latest.
	Version *string `toml:"version"`

	// Optional: branch to use. default to main.
	// this option takes priority over "Version" when configuring a config.
	Branch *string `toml:"branch"`

	// Optional: alternative path to use for config.
	// by default, the config path used to create the symlink if: $HOME/.config/<config-name>
	Path *string `toml:"path"`
}

func NewConfigCommand() cli.Command {
	return cli.Command{
		Name:  "config",
		Usage: "debug configurations.",
		Commands: []*cli.Command{
			{
				Name: "get",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rcfg, err := GetConfigFromFile()
					spew.Dump(rcfg)

					return err
				},
			},
		},
	}
}

func GetConfigFromFile() (*ReconConfig, error) {
	// get config
	homeDir := os.Getenv("HOME")
	configFilePath := path.Join(homeDir, RECON_CONFIG_FILE)
	configFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	var config ReconConfig
	err = toml.Unmarshal([]byte(configFile), &config)
	if err != nil {
		return nil, err
	}

	// set defaults
	if len(config.CloneDir) == 0 {
		config.CloneDir = DEFAULT_CLONE_DIR
	}

	return &config, nil
}
