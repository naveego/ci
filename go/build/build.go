package build

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/coreos/go-semver/semver"

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
	TargetLocal        = PackageTarget{"", ""}

	DefaultPackageTargets = []PackageTarget{
		TargetLinux386,
		TargetLinuxAmd64,
		TargetWindows386,
		TargetWindowsAmd64,
	}

	ReleaserTemplate *template.Template

	releaserConfig = `
# .goreleaser.yml
builds:
  - main: {{.Main}}
    binary: {{.Name}}
    goos:
      - windows
      - linux
      - darwin
    goarch:
      - amd64
    ldflags: '-s -w -X "{{.PackagePath}}/version.VersionBuild=Build.{{ "{{" }}.Env.BUILD_NUMBER{{ "}}" }}"'
    env:
      - CGO_ENABLED=0

dockers:
  - goos: linux
    goarch: amd64
    binary: {{.Name}}
    image: {{.DockerRepo}}/{{.Name}}
    tag_templates:
    - "{{ "{{" }} .Tag {{ "}}" }}"

# Archive customization
archive:
  format: tar.gz
  format_overrides:
    - goos: windows
      format: zip
  wrap_in_directory: true
  replacements:
    darwin: macOS
`
)

func init() {
	ReleaserTemplate = template.Must(template.New("releaser").Parse(releaserConfig))
}

// Package provides information for building a binary image
type Package struct {
	Name        string
	Version     semver.Version
	CommitHash  string
	PackagePath string
	OutDir      string
	DockerRepo  string
	Main        string // The path to main.go or build dir
	BuildArgs   []string
}

// NewPackage creates a new package with default values configured.
// The default values set by this operation are:
//		PackagePath: "github.com/naveegoinc/{name}"
//		OutDir: 	 "./bin"
//		DockerRepo:	 "docker.naveego.com:4333"
//		Main: 		 "main.go"
// Given the variables name="helloworld" and version="v1.0.0", the
// return package would have the following values:
//		Name:		 "helloworld"
//		Version:	 "v1.0.0"
//		PackagePath: "github.com/naveegoinc/helloworld"
//		OutDir:		 "./bin"
//		DockerRepo:	 "docker.naveego.com:4333"
//		Main:		 "main.go"
func NewPackage(name string, version semver.Version) Package {
	return Package{
		Name:        name,
		Version:     version,
		PackagePath: "github.com/naveegoinc/" + name,
		OutDir:      "./bin",
		DockerRepo:  "docker.naveego.com:4333",
		Main:        "main.go",
	}
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
	SetTeamCityParameter("env.VERSION_NUMBER", pkg.Version.String())
	var outDir string

	env := map[string]string{
		"CGO_ENABLED": "0",
	}

	if t.OS != "" {
		env["GOOS"] = t.OS
	}

	if t.Arch != "" {
		env["GOARCH"] = t.Arch
	}

	if pkg.OutDir != "" {
		outDir = pkg.OutDir
	} else {
		outDir = DefaultOutDir
	}

	var pkgName string
	if t.OS == "" && t.Arch == "" {
		pkgName = pkg.Name
	} else {
		pkgName = fmt.Sprintf("%s_%s_%s_%s", pkg.Name, pkg.Version.String(), t.OS, t.Arch)
	}

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

	buildArgs = append(buildArgs, pkg.Main)

	log.Printf("Building Package %s ...\n", pkgName)
	err := sh.RunWith(env, "go", buildArgs...)
	return outFile, err
}

// Release executes a gorelease operation
func Release(pkg Package) error {
	if !RunningOnTeamCity() {
		return fmt.Errorf("this operation should only be performed in our CI environment")
	}

	configFile := "./.goreleaser.yml"

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Println("no goreleaser config found, auto-generating .goreleaser.yml")
		tmpDir, err := ioutil.TempDir("", pkg.Name)
		if err != nil {
			return fmt.Errorf("could not create temp directory for goreleaser config, %v", err)
		}
		defer os.RemoveAll(tmpDir)

		err = writeConfigFile(tmpDir, pkg)
		if err != nil {
			return fmt.Errorf("could not write .goreleaser.yml, %v", err)
		}

		configFile = filepath.Join(tmpDir, ".goreleaser.yml")
	}

	// make sure we have goreleaser
	err := sh.Run("go", "get", "github.com/goreleaser/goreleaser")
	if err != nil {
		return fmt.Errorf("could not install goreleaser, %v", err)
	}

	log.Printf("executing goreleaser with config file '%s'\n", configFile)
	return sh.Run("goreleaser", "--config", configFile, "--rm-dist")
}

func writeConfigFile(tmpDir string, pkg Package) error {
	cfgPath := filepath.Join(tmpDir, ".goreleaser.yml")
	configFile, err := os.Create(cfgPath)
	if err != nil {
		return fmt.Errorf("could not create .goreleaser.yml, %v", err)
	}
	defer configFile.Close()

	return ReleaserTemplate.Execute(configFile, pkg)
}
