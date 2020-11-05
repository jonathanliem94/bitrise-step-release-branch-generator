package main

import (
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"os"
	"time"
)

func gitCloneMaster(url string, path string, auth transport.AuthMethod) (*git.Repository, error) {
	repo, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:           url,
		Auth:          auth,
		ReferenceName: "refs/heads/master",
		Progress:      os.Stdout,
		Tags:          git.NoTags,
	})
	return repo, err
}

func gitRefName(name string) plumbing.ReferenceName {
	return plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", name))
}

func gitCheckoutBranch(repo *git.Repository, branchName string) {
	branch := gitRefName(branchName)
	wt, _ := repo.Worktree()
	_ = wt.Checkout(&git.CheckoutOptions{
		Branch: branch,
	})
}

func gitAddAll(repo *git.Repository) error {
	wt, _ := repo.Worktree()
	err := wt.AddGlob(".")
	if err != nil {
		return err
	}
	return nil
}

func gitCommit(repo *git.Repository, commitMsg string) error {
	wt, _ := repo.Worktree()
	_, err := wt.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Bitrise",
			Email: "bitrise@bitrise.io",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func gitTag(repo *git.Repository, tagName string) error {
	head, _ := repo.Head()
	_, _ = fmt.Fprintf(os.Stdout, "Attempting to tag HEAD with: %s", tagName)
	_, err := repo.CreateTag(tagName, head.Hash(), nil)

	if err != nil {
		return errors.New(fmt.Sprintf("error creating tag: %v", err))
	}
	return nil
}

func gitPushTag(repo *git.Repository, auth transport.AuthMethod, tagName *string) error {
	refSpec := config.RefSpec("refs/tags/*:refs/tags/*")
	if tagName != nil {
		refSpec = config.RefSpec(fmt.Sprintf("refs/tags/%[1]s:refs/tags/%[1]s", *tagName))
	}
	opts := git.PushOptions{
		RefSpecs: []config.RefSpec{refSpec},
		Progress: os.Stdout,
		Auth: auth,
	}
	err := repo.Push(&opts)
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return err
	}
	return nil
}

func gitPushBranch(repo *git.Repository, auth transport.AuthMethod, branchName string) error {
	refSpec := config.RefSpec(fmt.Sprintf("refs/heads/%[1]s:refs/heads/%[1]s", branchName))
	opts := git.PushOptions{
		RefSpecs: []config.RefSpec{refSpec},
		Progress: os.Stdout,
		Auth:     auth,
	}
	err := repo.Push(&opts)
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return errors.New(fmt.Sprintf("unable to push branch: %v", err))
	}
	return nil
}

func getPublicKey(cfg *Config) (*ssh.PublicKeys, error) {
	sshPk, err := ssh.NewPublicKeysFromFile("git", cfg.SSHPrivateKeyPath, "")
	if err != nil {
		return nil, err
	}
	return sshPk, err
}
