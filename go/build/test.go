package build

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/sh"
)

// Ginkgo runs tests recursively
func Ginkgo(tags []string) error {
	// make sure we have ginkgo
	err := sh.Run("go", "get", "github.com/onsi/ginkgo/ginkgo")
	if err != nil {
		return fmt.Errorf("Could not install ginkgo, %v", err)
	}
	err := sh.Run("go", "get", "github.com/onsi/gomega/...")
	if err != nil {
		return fmt.Errorf("Could not install gomega, %v", err)
	}

	args := []string{
		"-r",
		"--progress",
		"--randomizeAllSpecs",
		"--randomizeSuites",
		"--cover",
		"--trace",
		"--race",
	}

	if len(tags) > 0 {
		args = append(args, "--tags")
		args = append(args, strings.Join(tags, " "))
	}

	return sh.Run("ginkgo", args)
}

// GinkgoIntegration runs test recusively, but uses integration tag
func GinkgoIntegration() error {
	return Ginkgo([]string{"integration"})
}
