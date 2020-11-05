package main

import (
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"os"
)

type Config struct {
	RepoPath              string `env:"BITRISE_SOURCE_DIR,required"`
	CloneUrl              string `env:"git_repo_url,required"`
	VersionCodeFile       string `env:"version_code_file,required"`
	ReleaseBranchTemplate string `env:"release_branch_template,required"`
	VersionCodeTemplate   string `env:"version_code_template,required"`
	VersionCodeRegex      string `env:"version_code_regex,required"`
	TagsToPush            string `env:"tags_to_push,required"`
}

func fail(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		fail("Error parsing config: %s", err)
	}
	stepconf.Print(cfg)
}
