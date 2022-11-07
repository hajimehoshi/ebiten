// Copyright 2019 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// ebitenmobile is a wrapper of gomobile for Ebitengine.
//
// For the usage, see https://ebitengine.org/en/documents/mobile.html.
//
// gomobile's version is fixed by ebitenmobile.
// You can specify gomobile's version by EBITENMOBILE_GOMOBILE environment variable.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	exec "golang.org/x/sys/execabs"
	"golang.org/x/tools/go/packages"
)

const (
	ebitenmobileCommand = "ebitenmobile"
)

//go:embed _files/EbitenViewController.h
var objcH string

func init() {
	flag.Usage = func() {
		// This message is copied from `gomobile bind -h`
		fmt.Fprintf(os.Stderr, "%s bind [-target android|ios] [-bootclasspath <path>] [-classpath <path>] [-o output] [build flags] [package]\n", ebitenmobileCommand)
		os.Exit(2)
	}
	flag.Parse()
}

func goEnv(name string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	val, err := exec.Command("go", "env", name).Output()
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(val))
}

const (
	// Copied from gomobile.
	minAndroidAPI = 15
)

var (
	buildA          bool   // -a
	buildI          bool   // -i
	buildN          bool   // -n
	buildV          bool   // -v
	buildX          bool   // -x
	buildO          string // -o
	buildGcflags    string // -gcflags
	buildLdflags    string // -ldflags
	buildTarget     string // -target
	buildTrimpath   bool   // -trimpath
	buildWork       bool   // -work
	buildBundleID   string // -bundleid
	buildIOSVersion string // -iosversion
	buildAndroidAPI int    // -androidapi
	buildTags       string // -tags

	bindPrefix        string // -prefix
	bindJavaPkg       string // -javapkg
	bindClasspath     string // -classpath
	bindBootClasspath string // -bootclasspath
)

func main() {
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
	}

	if args[0] != "bind" {
		flag.Usage()
	}

	var flagset flag.FlagSet
	flagset.StringVar(&buildO, "o", "", "")
	flagset.StringVar(&buildGcflags, "gcflags", "", "")
	flagset.StringVar(&buildLdflags, "ldflags", "", "")
	flagset.StringVar(&buildTarget, "target", "android", "")
	flagset.StringVar(&buildBundleID, "bundleid", "", "")
	flagset.StringVar(&buildIOSVersion, "iosversion", "7.0", "")
	flagset.StringVar(&buildTags, "tags", "", "")
	flagset.IntVar(&buildAndroidAPI, "androidapi", minAndroidAPI, "")
	flagset.BoolVar(&buildA, "a", false, "")
	flagset.BoolVar(&buildI, "i", false, "")
	flagset.BoolVar(&buildN, "n", false, "")
	flagset.BoolVar(&buildV, "v", false, "")
	flagset.BoolVar(&buildX, "x", false, "")
	flagset.BoolVar(&buildTrimpath, "trimpath", false, "")
	flagset.BoolVar(&buildWork, "work", false, "")
	flagset.StringVar(&bindJavaPkg, "javapkg", "", "")
	flagset.StringVar(&bindPrefix, "prefix", "", "")
	flagset.StringVar(&bindClasspath, "classpath", "", "")
	flagset.StringVar(&bindBootClasspath, "bootclasspath", "", "")

	_ = flagset.Parse(args[1:])

	buildTarget, err := osFromBuildTarget(buildTarget)
	if err != nil {
		log.Fatal(err)
	}

	// Add ldflags to suppress linker errors (#932).
	// See https://github.com/golang/go/issues/17807
	if buildTarget == "android" {
		if buildLdflags != "" {
			buildLdflags += " "
		}
		buildLdflags += "-extldflags=-Wl,-soname,libgojni.so"
	}

	dir, err := prepareGomobileCommands()
	defer func() {
		if dir != "" && !buildWork {
			_ = removeAll(dir)
		}
	}()
	if err != nil {
		log.Fatal(err)
	}

	if err := doBind(args, &flagset, buildTarget); err != nil {
		log.Fatal(err)
	}
}

func osFromBuildTarget(buildTarget string) (string, error) {
	var os string
	for i, pair := range strings.Split(buildTarget, ",") {
		osarch := strings.SplitN(pair, "/", 2)
		if i == 0 {
			os = osarch[0]
		}
		if os != osarch[0] {
			return "", fmt.Errorf("ebitenmobile: cannot target different OSes")
		}
	}
	if os == "ios" {
		os = "darwin"
	}
	return os, nil
}

func doBind(args []string, flagset *flag.FlagSet, buildOS string) error {
	tags := buildTags
	cfg := &packages.Config{}
	cfg.Env = append(os.Environ(), "GOOS="+buildOS)
	if buildOS == "darwin" {
		if tags != "" {
			tags += " "
		}
		tags += "ios"
	}
	cfg.BuildFlags = []string{"-tags", tags}

	flagsetArgs := flagset.Args()
	if len(flagsetArgs) == 0 {
		flagsetArgs = []string{"."}
	}
	pkgs, err := packages.Load(cfg, flagsetArgs[0])
	if err != nil {
		return err
	}
	prefixLower := bindPrefix + pkgs[0].Name
	prefixUpper := strings.Title(bindPrefix) + strings.Title(pkgs[0].Name)

	args = append(args, "github.com/hajimehoshi/ebiten/v2/mobile/ebitenmobileview")

	if buildO == "" {
		fmt.Fprintln(os.Stderr, "-o must be specified.")
		os.Exit(2)
		return nil
	}

	if buildN {
		fmt.Print("gomobile")
		for _, arg := range args {
			fmt.Print(" ", arg)
		}
		fmt.Println()
		return nil
	}

	cmd := exec.Command("gomobile", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(err.(*exec.ExitError).ExitCode())
		return nil
	}

	replacePrefixes := func(content string) string {
		content = strings.ReplaceAll(content, "{{.PrefixUpper}}", prefixUpper)
		content = strings.ReplaceAll(content, "{{.PrefixLower}}", prefixLower)
		return content
	}

	if buildOS == "darwin" {
		// TODO: Use os.ReadDir after Ebitengine stops supporting Go 1.15.
		f, err := os.Open(buildO)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()

		names, err := f.Readdirnames(-1)
		if err != nil {
			return err
		}

		for _, name := range names {
			if name == "Info.plist" {
				continue
			}
			frameworkName := filepath.Base(buildO)
			frameworkNameBase := frameworkName[:len(frameworkName)-len(".xcframework")]
			// The first character must be an upper case (#2192).
			// TODO: strings.Title is used here for the consistency with gomobile (see cmd/gomobile/bind_iosapp.go).
			// As strings.Title is deprecated, golang.org/x/text/cases should be used.
			frameworkNameBase = strings.Title(frameworkNameBase)
			dir := filepath.Join(buildO, name, frameworkNameBase+".framework", "Versions", "A")

			if err := os.WriteFile(filepath.Join(dir, "Headers", prefixUpper+"EbitenViewController.h"), []byte(replacePrefixes(objcH)), 0644); err != nil {
				return err
			}
			// TODO: Remove 'Ebitenmobileview.objc.h' here. Now it is hard since there is a header file importing
			// that header file.

			fs, err := os.ReadDir(filepath.Join(dir, "Headers"))
			if err != nil {
				return err
			}
			var headerFiles []string
			for _, f := range fs {
				if strings.HasSuffix(f.Name(), ".h") {
					headerFiles = append(headerFiles, f.Name())
				}
			}

			w, err := os.OpenFile(filepath.Join(dir, "Modules", "module.modulemap"), os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer func() {
				_ = w.Close()
			}()
			var mmVals = struct {
				Module  string
				Headers []string
			}{
				Module:  prefixUpper,
				Headers: headerFiles,
			}
			if err := iosModuleMapTmpl.Execute(w, mmVals); err != nil {
				return err
			}

			// TODO: Remove Ebitenmobileview.objc.h?
		}
	}

	return nil
}

var iosModuleMapTmpl = template.Must(template.New("iosmmap").Parse(`framework module "{{.Module}}" {
{{range .Headers}}    header "{{.}}"
{{end}}
    export *
}`))
