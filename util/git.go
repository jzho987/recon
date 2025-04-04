package util

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/jzho987/recon/config"
)

// State of the local repository
type RepoState struct {
	Hash   string
	Tag    *string
	Branch *string
}

func GetAllRepoState(dirPath string) (map[string]RepoState, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	repoStates := make(map[string]RepoState, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// open
		repoPath := path.Join(dirPath, entry.Name())
		gitRepo, err := git.PlainOpen(repoPath)
		if err != nil {
			return nil, err
		}
		head, err := gitRepo.Head()
		if err != nil {
			return nil, err
		}

		repoState := RepoState{
			Hash: head.Hash().String(),
		}

		// tag
		tag, err := gitRepo.TagObject(head.Hash())
		if err != nil && errors.Is(err, plumbing.ErrObjectNotFound) {
			return nil, err
		}
		if tag != nil {
			repoState.Tag = &tag.Name
		}

		// branch
		name := head.Name().String()
		if head.Name().IsBranch() {
			repoState.Branch = &name
		}

		repoStates[entry.Name()] = repoState
	}

	return repoStates, nil
}

func GetLabeledDirName(repoConfig config.RepoConfig) string {
	dirName := repoConfig.Name

	if repoConfig.Version != nil || repoConfig.Branch != nil {
		refMode := "version"
		refName := repoConfig.Version
		if repoConfig.Version == nil {
			refMode = "branch"
			refName = repoConfig.Branch
		}

		cleanedName := strings.Replace(dirName, "-", "_", -1)
		cleanedRefName := strings.Replace(*refName, "-", "_", -1)
		dirName = fmt.Sprintf("%s-%s:%s", cleanedName, refMode, cleanedRefName)
	}

	return dirName
}
