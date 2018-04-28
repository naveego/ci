package build

import "github.com/magefile/mage/sh"

func GinkgoIntegration() error {
	return sh.Run(
		"ginkgo",
		"--tags",
		"integration",
		"-r",
		"--progress",
		"--randomizeAllSpecs",
		"--randomizeSuites",
		"--cover",
		"--trace",
		"--race",
	)
}
