package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type Config struct {
	SourceDir             string `env:"BITRISE_SOURCE_DIR,required"`
	SSHPrivateKeyPath     string `env:"ssh_key_save_path,required"`
	CloneUrl              string `env:"git_repo_url,required"`
	VersionCodeFile       string `env:"version_code_file,required"`
	ReleaseBranchTemplate string `env:"release_branch_template,required"`
	VersionCodeTemplate   string `env:"version_code_template,required"`
	VersionCodeRegex      string `env:"version_code_regex,required"`
	TagFile               string `env:"tag_file,required"`
}

func fail(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func updateBuildNo(repo *git.Repository, cfg *Config) error {
	path := cfg.SourceDir + cfg.VersionCodeFile
	file, _ := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	reader := bufio.NewScanner(file)
	writer := bufio.NewWriter(file)

	var lines []string

	buildVersionRe, err := regexp.Compile(cfg.VersionCodeRegex)
	if err != nil {
		return err
	}

	replaced := false
	for reader.Scan() {
		line := reader.Text()

		if buildVersionRe.MatchString(line) {
			replaced = true
			verCodeRe := regexp.MustCompile(`\d+`)
			match := verCodeRe.FindString(line)
			verCode, err := strconv.Atoi(match)
			if err != nil {
				panic("Unable to parse versionCode")
			}

			var out bytes.Buffer
			funcMap := template.FuncMap{
				"add":
				func(i int, what int) int {
					return i + what
				},
			}
			t1, _ := template.New("verCode").Funcs(funcMap).Parse(cfg.VersionCodeTemplate)
			_ = t1.Execute(&out, verCode)
			verCodeNew, _ := strconv.Atoi(out.String())
			line = strings.Replace(line, match, strconv.Itoa(verCodeNew), 1)
		}
		lines = append(lines, line)
	}

	if !replaced {
		fail("failed")
	}

	_, _ = file.Seek(0, 0)
	for _, line := range lines {
		_, _ = writer.WriteString(line)
		_ = writer.WriteByte(10)
	}
	err = writer.Flush()
	if err != nil {
		return err
	}

	_ = gitAddAll(repo)
	_ = gitCommit(repo, "[skip ci] Update Version Code")

	return nil
}

func forkNewReleaseBranch(repo *git.Repository, cfg *Config) (*string, error) {
	now := time.Now()
	funcMap := template.FuncMap{
		"Week":
		func(t time.Time) int {
			_, week := t.ISOWeek()
			return week
		},
	}

	var out bytes.Buffer
	t1, _ := template.New("mutate").Funcs(funcMap).Parse(cfg.ReleaseBranchTemplate)
	_ = t1.Execute(&out, now)
	branchName := out.String()
	_, _ = fmt.Fprintf(os.Stdout, "Attempting to create branch: %s", branchName)
	newBranch := gitRefName(branchName)

	wt, _ := repo.Worktree()
	head, _ := repo.Head()

	err := wt.Checkout(&git.CheckoutOptions{
		Hash:   head.Hash(),
		Branch: newBranch,
		Create: true,
	})

	if err != nil {
		return nil, errors.New("unable to checkout release branch\n")
	}

	_, err = wt.Commit("diverge from master", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Bitrise",
			Email: "bitrise@bitrise.io",
			When:  now,
		},
	})

	if err != nil {
		return nil, errors.New("unable to create diverge commit\n")
	}

	return &branchName, nil
}

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		fail("Error parsing config: %s\n", err)
	}
	stepconf.Print(cfg)

	pk, err := getPublicKey(&cfg)
	if err != nil {
		fail("%v\n", err)
	}
	repo, err := gitCloneMaster(cfg.CloneUrl, cfg.SourceDir, pk)
	if err != nil {
		fail("%v\n", err)
	}
	_ = updateBuildNo(repo, &cfg)

	if err := gitPushBranch(repo, pk, "master"); err != nil {
		fail("%v\n", err)
	}

	branchName, _ := forkNewReleaseBranch(repo, &cfg)
	if err := gitPushBranch(repo, pk, *branchName); err != nil {
		fail("%v\n", err)
	}

	if err := processTagFile(repo, pk, cfg.TagFile); err != nil {
		fail("%v", err)
	}
}
