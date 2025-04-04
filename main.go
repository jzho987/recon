package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/urfave/cli/v3"

	reconConfig "github.com/jzho987/recon/config"
	"github.com/jzho987/recon/util"
)

const (
	BASE_CONFIG_DIR = ".config"
	GIT_DIRS        = ".config/recon/git-dirs/"
	DB_FILE         = ".config/recon/.data"
)

func main() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "config",
				Commands: []*cli.Command{
					{
						Name: "get",
						Action: func(ctx context.Context, cmd *cli.Command) error {

							rcfg, err := reconConfig.GetConfigFromFile()
							spew.Dump(rcfg)

							return err
						},
					},
				},
			},
			{
				Name: "sync",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Usage: "the specific config to sync",
					},
				},
				Usage:  "sync configured remote configs with current config.",
				Action: syncFunc,
			},
		},
	}
	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func syncFunc(ctx context.Context, cmd *cli.Command) error {
	rcfg, err := reconConfig.GetConfigFromFile()
	if err != nil {
		fmt.Printf("errors getting config from file. err: %s", err)
		return err
	}
	repoConfigs := rcfg.Repos

	if len(cmd.String("config")) != 0 {
		conf := cmd.String("config")
		var filtered *reconConfig.RepoConfig
		for _, repoConfig := range repoConfigs {
			if repoConfig.Name != conf {
				continue
			}

			filtered = &repoConfig
			break
		}

		if filtered == nil {
			fmt.Printf("config %s not found in recon.toml.\n", conf)
			return errors.New("missing config")
		}

		repoConfigs = []reconConfig.RepoConfig{*filtered}
	}

	// get existing git dirs
	homeDir := os.Getenv("HOME")
	gitDir := path.Join(homeDir, GIT_DIRS)
	entries, err := os.ReadDir(gitDir)
	if err != nil {
		return err
	}

	// dedupe desired dirs
	existingDirSet := make(map[string]bool, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		existingDirSet[entry.Name()] = true
	}

	missingDirs := make([]reconConfig.RepoConfig, 0)
	existingDirs := make([]reconConfig.RepoConfig, 0)
	for _, repoConfig := range repoConfigs {
		labeledDirName := util.GetLabeledDirName(repoConfig)
		if existingDirSet[labeledDirName] {
			existingDirs = append(existingDirs, repoConfig)
			continue
		}
		missingDirs = append(missingDirs, repoConfig)
	}

	fmt.Printf("found %d missing repos.\n", len(missingDirs))

	// clone dirs that don't exist
	sshPath := fmt.Sprintf("%s/.ssh/id_rsa", homeDir)
	auth, err := ssh.NewPublicKeysFromFile("git", sshPath, "")
	for _, repoConfig := range missingDirs {
		labeledDirName := util.GetLabeledDirName(repoConfig)

		fmt.Printf("began pulling\t\t: [%s]\n", labeledDirName)

		cloneOps := git.CloneOptions{
			URL:  repoConfig.Remote,
			Auth: auth,
		}
		if repoConfig.Branch != nil {
			branchRef := fmt.Sprintf("refs/heads/%s", *repoConfig.Branch)
			refName := plumbing.ReferenceName(branchRef)
			cloneOps.ReferenceName = refName
			cloneOps.SingleBranch = true
		}

		cloneDir := path.Join(gitDir, labeledDirName)
		_, err := git.PlainClone(cloneDir, false, &cloneOps)
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			fmt.Println("repository already exist. skipping...")
		} else if err != nil {
			fmt.Printf("error cloning git repository; err: %+v;\n", err)
			return err
		}

		fmt.Printf("finished pulling\t: [%s]\n", labeledDirName)
	}

	// TODO: update dirs that do exist

	// setup sym links
	fmt.Println("resolving symlinks")
	for _, repoConfig := range repoConfigs {
		labeledDirName := util.GetLabeledDirName(repoConfig)

		internalConfigPath := ""
		if repoConfig.Path != nil {
			internalConfigPath = *repoConfig.Path
		}

		clonedConfigPath := path.Join(gitDir, labeledDirName, internalConfigPath)
		configPath := path.Join(homeDir, BASE_CONFIG_DIR, repoConfig.Name)

		info, err := os.Lstat(configPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Printf("error lstatting path: %s. err: %s\n", configPath, err)
			return err
		}
		if info != nil {
			if info.Mode().Type() != os.ModeSymlink.Type() {
				oldConfigReplaceName := fmt.Sprintf("%s-old", repoConfig.Name)
				oldConfigReplacePath := path.Join(homeDir, BASE_CONFIG_DIR, oldConfigReplaceName)
				fmt.Printf("existing config found at: %s\n", configPath)
				fmt.Println("would you like to replace the config file?")
				fmt.Printf("(old config will be moved to: %s)\n", oldConfigReplacePath)
				fmt.Print("(y/n):")

				var i string
				fmt.Scan(&i)

				switch strings.ToLower(i) {
				case "y":
					fmt.Println("replacing old config")
					err := os.Rename(configPath, oldConfigReplacePath)
					if err != nil {
						fmt.Printf("error renaming old config. err: %s", err)
						return err
					}

				case "default":
					fallthrough
				case "n":
					fmt.Printf("skipping %s for now...", repoConfig.Name)
					continue
				}
				continue
			} else {
				// existing symlink
				syml, err := os.Readlink(configPath)
				if err != nil {
					fmt.Printf("error resolving symlink: %s. err: %s", configPath, err)
					return err
				}

				if syml == clonedConfigPath {
					// already configured correctly
					continue
				}

				// clean up incorrect symlink
				os.Remove(configPath)
			}
		}

		fmt.Printf("linking: %s \t -> %s.\n", clonedConfigPath, configPath)
		err = os.Symlink(clonedConfigPath, configPath)
		if err != nil {
			fmt.Printf("error creating symlink from %s to %s. err: %+v", clonedConfigPath, configPath, err)
			return err
		}
	}

	return nil
}
