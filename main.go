package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/urfave/cli/v3"
)

const (
	BASE_CONFIG_DIR = ".config"
	CONFIG_DIR      = "/.config/recon/"
	GIT_DIRS        = ".config/recon/git-dirs/"
	DB_FILE         = "/.config/recon/.data"
)

func main() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:   "pull",
				Usage:  "pull config from remote.",
				Action: pullFunc,
			},
			{
				Name: "add",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "repo",
						Usage: "the repository of which the configuration file lives in, either in the root or in a sub directory.",
					},
					&cli.StringFlag{
						Name:  "path",
						Usage: "the path of the config file within the remote repo. by default it uses the root of the repository as the config path.",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "the path of the config file to replace. by default it is ~/.config/<arg_1>/...",
					},
					&cli.StringFlag{
						Name:  "branch",
						Usage: "the branch of the repository to use as default.",
					},
				},
				Usage:  "add new config.",
				Action: addFunc,
			},
			{
				Name:   "init",
				Usage:  "initialize config path for all reconfigured tools.",
				Action: initFunc,
			},
		},
	}
	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func addFunc(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 {
		fmt.Println("please input valid config.")
		return errors.New("incorrect config")
	}
	conf := cmd.Args().Get(0)
	if len(conf) == 0 {
		fmt.Println("please input valid config.")
		return errors.New("incorrect config")
	}

	if len(cmd.String("repo")) == 0 {
		fmt.Println("please input repo using --repo.")
		return errors.New("missing required flag")
	}
	repo := cmd.String("repo")

	homeDir := os.Getenv("HOME")
	gitDir := path.Join(homeDir, GIT_DIRS)

	fmt.Printf("creating directory: %s;\n", gitDir)
	command := exec.Command("mkdir", "-p", gitDir)
	command.Stdout = os.Stdout
	err := command.Run()
	if err != nil {
		fmt.Printf("error creating directory for new cofig; err: %+v;\n", err)
		return err
	}
	cloneDir := path.Join(gitDir, conf)

	sshPath := fmt.Sprintf("%s/.ssh/id_rsa", homeDir)
	auth, err := ssh.NewPublicKeysFromFile("git", sshPath, "")
	if err != nil {
		fmt.Printf("error setting up ssh pub key; err: %+v;\n", err)
		return err
	}

	fmt.Printf("cloning git repo: %s;\n", repo)
	gitRepo, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:      repo,
		Auth:     auth,
		Progress: os.Stdout,
	})
	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		fmt.Println("repository already exist. skipping...")
	} else if err != nil {
		fmt.Printf("error cloning git repository; err: %+v;\n", err)
		return err
	}

	// clean up and switch branch
	if len(cmd.String("branch")) != 0 && gitRepo != nil {
		fmt.Printf("found branch option, switching to branch: %s;\n", cmd.String("branch"))

		workTree, err := gitRepo.Worktree()
		if err != nil {
			fmt.Printf("error getting git work tree. err: %s", err)
			return err
		}
		if workTree == nil {
			fmt.Print("error getting git work tree. nil work tree")
			return errors.New("nil work tree")
		}
		err = gitRepo.Fetch(&git.FetchOptions{
			Prune: true,
		})
		if err != nil {
			fmt.Printf("error fetching from git repository. err: %+v\n", err)
			return err
		}

		err = workTree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName(cmd.String("branch")),
		})
		if err != nil {
			fmt.Printf("error checking out branch. err: %s", err)
			return err
		}
		fmt.Printf("successfully, switched to branch: %s;\n", cmd.String("branch"))
	}

	configPath := path.Join(homeDir, BASE_CONFIG_DIR, conf)
	_, err = os.Stat(configPath)
	// if config exist, we move it to another place.
	if !errors.Is(err, os.ErrNotExist) {
		hideConf := fmt.Sprintf("%s-old", conf)
		hideConfigPath := path.Join(homeDir, BASE_CONFIG_DIR, hideConf)
		fmt.Printf("config already exists at: %s\n", configPath)
		fmt.Printf("do you with to hide the old config to: %s?\n", hideConfigPath)
		fmt.Print("(y/n):")

	input_loop:
		for {
			var i string
			fmt.Scan(&i)
			i = strings.ToLower(i)
			i = strings.TrimSpace(i)

			switch i {
			case "y":
				fmt.Println("moving old config")
				// gopls doesn't like it if I don't tag the loop like WHAT???
				break input_loop

			case "n":
				fmt.Println("old config is conflicting with new config, aborting...")
				return nil

			default:
				fmt.Print("invalid input, please pick (y/n):")
			}
		}

		err := os.Rename(configPath, hideConfigPath)
		if err != nil {
			fmt.Printf("error moving config %s to %s. err: %+v", configPath, hideConfigPath, err)
			return err
		}
	}

	fmt.Println("creating sym link.")
	err = os.Symlink(cloneDir, configPath)
	if err != nil {
		fmt.Printf("error creating symlink from %s to %s. err: %+v", cloneDir, configPath, err)
		return err
	}

	return nil
}

func pullFunc(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		fmt.Println("please input valid config.")
		return errors.New("incorrect config")
	}
	conf := cmd.Args().Get(0)
	if len(conf) == 0 {
		fmt.Println("please input valid config.")
		return errors.New("incorrect config")
	}

	fmt.Printf("config: %s;", conf)
	return nil
}

func initFunc(ctx context.Context, cmd *cli.Command) error {
	homeDir := os.Getenv("HOME")
	dataPath := path.Join(homeDir, DB_FILE)
	err := putAndWriteFile(dataPath, []byte{})
	if err != nil {
		fmt.Printf("error creating data file. err: %+v", err)
		return err
	}

	data, err := os.ReadFile(dataPath)
	if err != nil {
		fmt.Printf("error reading file. err: %+v", err)
		return err
	}

	fmt.Printf("%s\n", data)

	return nil
}

func putAndWriteFile(filePath string, data []byte) error {
	_, err := os.Stat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		dirPath := path.Dir(filePath)
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			fmt.Println("error handling data file.")
			return err
		}
	} else if err != nil {
		fmt.Printf("error handling data file. err: %+v", err)
		return err
	}

	err = os.WriteFile(filePath, data, os.ModePerm)
	if err != nil {
		fmt.Println("error writing data file.")
		return err
	}

	return nil
}
