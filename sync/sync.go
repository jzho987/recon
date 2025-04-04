package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"

	reconConfig "github.com/jzho987/recon/config"
	"github.com/jzho987/recon/util"
)

const BASE_CONFIG_DIR = ".config"

func NewSyncCommand() cli.Command {
	return cli.Command{
		Name: "sync",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "the specific config to sync",
			},
			&cli.BoolFlag{
				Name:  "clean",
				Usage: "whether or not to delete the unused repositories",
			},
		},
		Usage:  "sync configured remote configs with current config.",
		Action: syncFunc,
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
	gitDir := path.Join(homeDir, rcfg.CloneDir)
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
	cloneRepos := make(map[string]reconConfig.RepoConfig, 0)
	labeledDirNameSet := make(map[string]bool, 0)
	for _, repoConfig := range repoConfigs {
		labeledDirName, err := util.GetLabeledDirName(repoConfig)
		labeledDirNameSet[labeledDirName] = true
		if err != nil {
			fmt.Printf("error getting clone destination directory name. err: %s", err)
			return err
		}
		if existingDirSet[labeledDirName] {
			existingDirs = append(existingDirs, repoConfig)
			continue
		}

		cloneRepos[labeledDirName] = repoConfig
		missingDirs = append(missingDirs, repoConfig)
	}

	if cmd.Bool("clean") {
		unusedExistingDirs := make([]string, 0)
		for _, entry := range entries {
			if labeledDirNameSet[entry.Name()] {
				continue
			}

			fullPath := path.Join(gitDir, entry.Name())
			unusedExistingDirs = append(unusedExistingDirs, fullPath)
		}

		fmt.Printf("found %d unused repos.\n", len(unusedExistingDirs))

		if len(unusedExistingDirs) != 0 {
			fmt.Print("delete unused? (y/n): ")
			var i string
			fmt.Scan(&i)
			switch strings.ToLower(i) {
			case "y":
				for _, unusedExistingDir := range unusedExistingDirs {
					err := os.RemoveAll(unusedExistingDir)
					if err != nil {
						fmt.Printf("error deleting unused repo: %s. err: %+v \n", unusedExistingDir, err)
						continue
					}
				}

			case "n":
				fallthrough
			default:
				fmt.Println("ignoring for now...")
			}
		}
	}

	fmt.Printf("found %d missing repos.\n", len(cloneRepos))

	// clone dirs that don't exist
	sshPath := fmt.Sprintf("%s/.ssh/id_rsa", homeDir)
	auth, err := ssh.NewPublicKeysFromFile("git", sshPath, "")
	var eg errgroup.Group
	for _, repoConfig := range cloneRepos {
		eg.Go(func() error {
			labeledDirName, err := util.GetLabeledDirName(repoConfig)
			if err != nil {
				fmt.Printf("error getting clone destination directory name. err: %s", err)
				return err
			}

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
			_, err = git.PlainClone(cloneDir, false, &cloneOps)
			if errors.Is(err, git.ErrRepositoryAlreadyExists) {
				fmt.Println("repository already exist. skipping...")
			} else if err != nil {
				fmt.Printf("error cloning git repository; err: %+v;\n", err)
				return err
			}

			fmt.Printf("finished pulling\t: [%s]\n", labeledDirName)
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		return err
	}

	// TODO: update dirs that do exist

	// setup sym links
	symLinksCreatedMetric := 0
	for _, repoConfig := range repoConfigs {
		labeledDirName, err := util.GetLabeledDirName(repoConfig)
		if err != nil {
			fmt.Printf("error getting clone destination directory name. err: %s", err)
			return err
		}

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
				oldConfigReplaceName := fmt.Sprintf(".%s-old", repoConfig.Name)
				oldConfigReplacePath := path.Join(homeDir, BASE_CONFIG_DIR, oldConfigReplaceName)
				fmt.Printf("existing config found at: %s\n", configPath)
				fmt.Println("would you like to replace the config file?")
				fmt.Printf("(old config will be moved to: %s)\n", oldConfigReplacePath)
				fmt.Print("(y/n): ")

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

				case "n":
					fallthrough
				default:
					fmt.Printf("skipping %s for now...", repoConfig.Name)
					continue
				}
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
		symLinksCreatedMetric += 1
	}

	fmt.Printf("resolved %d symlink.", symLinksCreatedMetric)

	return nil
}
