package build

import (
	"os"

	"github.com/magefile/mage/sh"
)

func CopyFile(srcFile, dstFile string) error {
	return sh.Run("cp", srcFile, dstFile)
}

func MakeExecutable(file string) error {
	return os.Chmod(file, 0755)
}
