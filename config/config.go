package config

import (
	"os"
	"path"

	"github.com/pelletier/go-toml/v2"
)

const RECON_CONFIG_FILE = ".config/recon/recon.toml"

// All the config options for recon.
type ReconConfig struct {
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

func GetConfigFromFile() (*ReconConfig, error) {
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

	return &config, nil
}
