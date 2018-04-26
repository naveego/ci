package build

import (
	"strings"

	"github.com/magefile/mage/sh"
)

func MustGetGit() (branch, commit, shortCommit string) {
	var err error

	branch, err = GitBranch()
	if err != nil {
		panic(err)
	}

	commit, err = GitHash()
	if err != nil {
		panic(err)
	}

	shortCommit, err = GitShortHash()
	if err != nil {
		panic(err)
	}

	return
}

func GitHash() (string, error) {
	outBytes, err := sh.Output("git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(outBytes)), nil
}

func GitShortHash() (string, error) {
	outBytes, err := sh.Output("git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(outBytes)), nil
}

func GitBranch() (string, error) {
	outBytes, err := sh.Output("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(outBytes)), nil
}

func OnReleaseBranch() bool {
	branch, err := GitBranch()
	if err != nil {
		return false
	}
	return strings.HasPrefix(branch, "release")
}

func OnMasterBranch() bool {
	branch, err := GitBranch()
	if err != nil {
		return false
	}
	return branch == "master"
}

func GitTag(tag, msg string) error {
	return sh.Run("git", "tag", tag, "-m", msg)
}

func GitPushToRemote(remote, target string) error {
	return sh.Run("git", "push", remote, target)
}

func GetPush(target string) error {
	return GitPushToRemote("origin", target)
}
