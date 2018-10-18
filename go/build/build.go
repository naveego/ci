package build

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// Alternative to setting Version, to avoid vendoring annoyances
	VersionString string
	PackagePath string
	OutDir      string
	// If present, will be compiled into a template and passed a Build to construct the name of the compiled binary.
	OutTemplate string
	DockerRepo  string
	Shrink      bool
	Main        string // The path to main.go or build dir
	BuildArgs   []string
	CGOEnabled bool
}

// NewPackage creates a new package with default values configured.
// The default values set by this operation are:
// 		PackagePath: "github.com/naveegoinc/{name}"
// 		OutDir: 	 "./bin"
// 		DockerRepo:	 "docker.naveego.com:4333"
// 		Main: 		 "main.go"
// Given the variables name="helloworld" and version="v1.0.0", the
// return package would have the following values:
// 		Name:		 "helloworld"
// 		Version:	 "v1.0.0"
// 		PackagePath: "github.com/naveegoinc/helloworld"
// 		OutDir:		 "./bin"
// 		DockerRepo:	 "docker.naveego.com:4333"
// 		Main:		 "main.go"
func NewPackage(name string, version semver.Version) Package {
	return Package{
		Name:        name,
		Version:     version,
		VersionString: version.String(),
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

// Build combines a Package and a PackageTarget
type Build struct {
	Package Package
	PackageTarget PackageTarget
}

func (t PackageTarget) String() string {
	return fmt.Sprintf("%s_%s", t.OS, t.Arch)
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

	var err error
	if pkg.VersionString == "" {
		pkg.VersionString = pkg.Version.String()
	}

	SetTeamCityParameter("env.VERSION_NUMBER", "v"+pkg.VersionString)
	var outDir string

	env := map[string]string{}

	if pkg.CGOEnabled {
		env["CGO_ENABLED"] = "1"
	} else {
		env["CGO_ENABLED"] = "0"
	}

	if t.OS != "" {
		env["GOOS"] = t.OS
	}

	if t.Arch != "" {
		env["GOARCH"] = t.Arch
	}

	var outFile string

	if pkg.OutTemplate != "" {
		outTemplate := template.Must(template.New("out").Parse(pkg.OutTemplate))
		b := new(strings.Builder)
		err = outTemplate.Execute(b, Build{pkg,t})
		if err != nil {
			return "", fmt.Errorf("executing pkg.OutTemplate %q: %s", pkg.OutTemplate, err)
		}
		outFile = b.String()
	} else {
		if pkg.OutDir != "" {
			outDir = pkg.OutDir
		} else {
			outDir = DefaultOutDir
		}

		var pkgName string
		if t.OS == "" && t.Arch == "" {
			pkgName = pkg.Name
		} else {
			pkgName = fmt.Sprintf("%s_%s_%s_%s", pkg.Name, pkg.VersionString, t.OS, t.Arch)
		}

		if t.OS == "windows" {
			pkgName = pkgName + ".exe"
		}

		outFile = filepath.Join(outDir, pkgName)
	}


	buildArgs := []string{
		"build",
		"-o",
		outFile,
	}

	for _, a := range pkg.BuildArgs {
		buildArgs = append(buildArgs, a)
	}

	buildArgs = append(buildArgs, pkg.Main)

	log.Printf("Building %s to %s ...\n", pkg.PackagePath, outFile)
	err = sh.RunWith(env, "go", buildArgs...)

	if err != nil {
		if pkg.Shrink {
			tryShrink(pkg, t, outFile)
		}
	}

	return outFile, err
}

func tryShrink(pkg Package, t PackageTarget, binaryPath string) {
	strip, err := exec.LookPath("strip")
	if strip != "" && err == nil {
		switch t.OS {
		case "linux":
			if err = sh.Run("strip", binaryPath); err != nil {
				log.Printf("running strip on %q returned error: %s", binaryPath, err)
			}

		}
	}

	upx, err := exec.LookPath("upx")
	if upx != "" && err == nil {
		if err = sh.Run("upx", binaryPath); err != nil {
			log.Printf("running UPX on %q returned error: %s", binaryPath, err)
		}
	}
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

type PluginConfig struct {
	Package Package
	Targets []PackageTarget
	Files   []string
}

func BuildPlugin(cfg PluginConfig) error {

	manifestBytes, err := ioutil.ReadFile("manifest.json")
	if err != nil {
		return err
	}
	var manifest map[string]interface{}
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		return err
	}

	manifest["version"] = cfg.Package.VersionString
	pkg := cfg.Package
	if iconFile, ok := manifest["iconFile"].(string); ok {
		iconBytes, err := ioutil.ReadFile(iconFile)
		if err == nil {
			iconBytes64 := base64.StdEncoding.EncodeToString(iconBytes)
			ext := filepath.Ext(iconFile)
			icon64 := fmt.Sprintf("data:image/%s;base64,%s", ext, iconBytes64)
			manifest["icon"] = icon64
		}
	}

	if pkg.OutTemplate == "" {
		pkg.OutTemplate = fmt.Sprintf("build/outputs/{{.PackageTarget.OS}}/{{.PackageTarget.Arch}}/%s/{{.Package.VersionString}}/%s{{if eq .PackageTarget.OS `windows`}}.exe{{end}}", cfg.Package.Name, cfg.Package.Name)
	}

	for _, target := range cfg.Targets {

		outBinary, err := BuildPackage(pkg, target)
		outDir := filepath.Dir(outBinary)

		if err != nil {
			return fmt.Errorf("error building target %s", target)
		}

		manifest["os"] = target.OS
		manifest["arch"] = target.Arch
		manifest["executable"] = filepath.Base(outBinary)

		outManifest := filepath.Join(outDir, "manifest.json")

		manifestBytes, _ = json.Marshal(manifest)

		ioutil.WriteFile(outManifest, manifestBytes, 0777)

		include := []string{
			outBinary,
			outManifest,
		}
		for _, file := range cfg.Files {
			dst := filepath.Join(outDir, file)
			os.Link(file, dst)
			include = append(include, dst)
		}

		zipPath := filepath.Join(outDir, "package.zip")

		err = ZipFiles(zipPath, include)
		if err != nil {
			return fmt.Errorf("error zipping files %v into %q: %s", include, zipPath, err)
		}

		uploadEnv := os.Getenv("UPLOAD")

		fmt.Println("UPLOAD: ", uploadEnv)

		if uploadEnv != "" {

			err = ensureGoBetweenInstalled()
			if err != nil {
				return err
			}

			err = sh.Run("between", "dev", "upload-plugin", zipPath, "--env", uploadEnv)
			if err != nil {
				return err
			}
		}
	}

	return err
}

func ensureGoBetweenInstalled() error {

	exe, err := exec.LookPath("between")
	if err != nil {
		return err
	}
	if exe == "" {
		return errors.New("you need to run go install github.com/naveegoinc/go-between/cmd/between")
	}

	return nil
}