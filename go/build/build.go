package build

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

const (
	// DefaultOutDir defines the default location for build packages
	DefaultOutDir = "./bin"
)

var (
	TargetLinux386     = PackageTarget{"linux", "386"}
	TargetLinuxAmd64   = PackageTarget{"linux", "amd64"}
	TargetWindows386   = PackageTarget{"windows", "386"}
	TargetWindowsAmd64 = PackageTarget{"windows", "amd64"}
	TargetDarwinAmd64  = PackageTarget{"darwin", "amd64"}

	DefaultPackageTargets = []PackageTarget{
		TargetLinux386,
		TargetLinuxAmd64,
		TargetWindows386,
		TargetWindowsAmd64,
	}
)

// Package provides information for building a binary image
type Package struct {
	Name       string
	Version    string
	CommitHash string
	OutDir     string
	Path       string // The path to .go files to build
	BuildArgs  []string
}

// PackageTarget defines
type PackageTarget struct {
	OS   string
	Arch string
}

// BuildPackages performs a go build on the supplied package.
func BuildPackages(pkg Package, targets ...PackageTarget) error {
	var buildTargets []PackageTarget

	if len(targets) > 0 {
		buildTargets = targets
	} else {
		buildTargets = DefaultPackageTargets
	}

	for _, t := range buildTargets {
		BuildPackage(pkg, t)
	}

	return nil
}

// BuildPackage builds a package and returns the path to the output.
func BuildPackage(pkg Package, t PackageTarget) (string, error) {
	var outDir string

	env := map[string]string{
		"GOOS":        t.OS,
		"GOARCH":      t.Arch,
		"CGO_ENABLED": "0",
	}

	if pkg.OutDir != "" {
		outDir = pkg.OutDir
	} else {
		outDir = DefaultOutDir
	}

	pkgName := fmt.Sprintf("%s_%s_%s_%s", pkg.Name, pkg.Version, t.OS, t.Arch)

	if t.OS == "windows" {
		pkgName = pkgName + ".exe"
	}

	outFile := filepath.Join(outDir, pkgName)

	buildArgs := []string{
		"build",
		"-o",
		outFile,
	}

	for _, a := range pkg.BuildArgs {
		buildArgs = append(buildArgs, a)
	}

	buildArgs = append(buildArgs, pkg.Path)

	log.Printf("Building Package %s ...\n", pkgName)
	err := sh.RunWith(env, "go", buildArgs...)
	return outFile, err
}
