package build

import (
	"fmt"

	// mg contains helpful utility functions, like Deps
	"github.com/magefile/mage/sh"
)

// TagAndPushDockerImages uses the environment parameters
// IMAGE_NAME, BUILD_NUMBER, MAJOR_VERSION, and MINOR_VERSION
// to create images with the following tags:
// IMAGE_NAME:MAJOR_VERSION.MINOR_VERSION-BUILD_NUMBER
// IMAGE_NAME:MAJOR_VERSION.MINOR_VERSION
// IMAGE_NAME:MAJOR_VERSION
// IMAGE_NAME:latest
// If IMAGE_TAG_PREFIX is set, it will be inserted at the beginning of the tag.
// This task expects there to be an existing image named IMAGE_NAME:git-commit-hash
// This task returns a slice containing the deployed images, in order from
// most specific to least specific.
func TagAndPushDockerImages(sourceImage, imageName, imageTagPrefix, buildNumber, majorVersion, minorVersion string) ([]string, error) {
	var (
		err error
	)

	images := []string{		
		fmt.Sprintf("%s:%s%s.%s-build.%s", imageName, imageTagPrefix, majorVersion, minorVersion, buildNumber),
		fmt.Sprintf("%s:%s%s.%s", imageName, imageTagPrefix, majorVersion, minorVersion),
		fmt.Sprintf("%s:%s%s", imageName, imageTagPrefix, majorVersion),
		fmt.Sprintf("%s:%slatest", imageName, imageTagPrefix),
	}

	for _, name := range images {
		err = sh.Run("docker", "tag", sourceImage, name)
		if err != nil {
			return nil, fmt.Errorf("error tagging image '%s' as '%s': %s", sourceImage, name, err)
		}

		err = sh.Run("docker", "push", name)
		if err != nil {
			return nil, fmt.Errorf("error pushing image '%s': %s", name, err)
		}
	}

	return images, nil
}
