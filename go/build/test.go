package build

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/sh"
)

// RunUnitTests runs unit tests recursively
func RunUnitTests(tags []string) error {
	// make sure we have ginkgo
	err := sh.Run("go", "get", "github.com/onsi/ginkgo/ginkgo")
	if err != nil {
		return fmt.Errorf("Could not install ginkgo, %v", err)
	}
	err = sh.Run("go", "get", "github.com/onsi/gomega/...")
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
		"-keepGoing",
	}

	if len(tags) > 0 {
		args = append(args, "--tags")
		args = append(args, strings.Join(tags, " "))
	}

	return sh.Run("ginkgo", args...)
}

// RunIntegrationTestsInDocker executes integration tests using docker-compose.
func RunIntegrationTestsInDocker(name, dockerComposePath string) error {
	defer cleanUpAfterIntegrationTests(name, dockerComposePath)

	fmt.Println("Running Integration Tests...")

	fmt.Println("Building Docker Compose Images...")
	err := sh.Run("docker-compose", "-p", name, "-f", dockerComposePath, "build")
	if err != nil {
		return err
	}

	fmt.Println("Running Docker Compose Images...")
	err = sh.Run("docker-compose", "-p", name, "-f", dockerComposePath, "up", "-d")
	if err != nil {
		return err
	}

	runCmd := exec.Command("docker-compose", "-p", name, "-f", dockerComposePath, "run", "sut", "ginkgo", "-tags", "integration", "-r", "--progress", "--randomizeAllSpecs", "--randomizeSuites", "--cover", "--trace", "--race", "-keepGoing")
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	err = runCmd.Run()
	if err != nil {
		return fmt.Errorf("Tests failed: %v", err)
	}

	return nil

}

func cleanUpAfterIntegrationTests(name, dockerComposePath string) {
	sh.Run("docker-compose", "-p", name, "-f", dockerComposePath, "kill")
	sh.Run("docker-compose", "-p", name, "-f", dockerComposePath, "rm", "-f")
}
