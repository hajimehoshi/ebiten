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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

const (
	ebitenmobileCommand = "ebitenmobile"
)

func init() {
	flag.Usage = func() {
		// This message is copied from `gomobile bind -h`
		fmt.Fprintf(os.Stderr, "%s bind [-target android|ios] [-bootclasspath <path>] [-classpath <path>] [-o output] [build flags] [package]", ebitenmobileCommand)
		os.Exit(2)
	}
	flag.Parse()
}

func goEnv(name string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	gocmd := filepath.Join(runtime.GOROOT(), "bin", "go")
	val, err := exec.Command(gocmd, "env", name).Output()
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
	flagset.BoolVar(&buildWork, "work", false, "")
	flagset.StringVar(&bindJavaPkg, "javapkg", "", "")
	flagset.StringVar(&bindPrefix, "prefix", "", "")
	flagset.StringVar(&bindClasspath, "classpath", "", "")
	flagset.StringVar(&bindBootClasspath, "bootclasspath", "", "")

	flagset.Parse(args[1:])

	// Add ldflags to suppress linker errors (#932).
	// See https://github.com/golang/go/issues/17807
	if buildLdflags == "" {
		buildLdflags += " "
	}
	buildLdflags += "-extldflags=-Wl,-soname,libgojni.so"

	if err := prepareGomobileCommands(); err != nil {
		log.Fatal(err)
	}

	switch args[0] {
	case "bind":
		if err := doBind(args, &flagset); err != nil {
			log.Fatal(err)
		}
	default:
		flag.Usage()
	}
}

func doBind(args []string, flagset *flag.FlagSet) error {
	tags := buildTags
	cfg := &packages.Config{}
	switch buildTarget {
	case "android":
		cfg.Env = append(os.Environ(), "GOOS=android")
	case "ios":
		cfg.Env = append(os.Environ(), "GOOS=darwin")
		if tags != "" {
			tags += " "
		}
		tags += "ios"
	}
	cfg.BuildFlags = []string{"-tags", tags}

	pkgs, err := packages.Load(cfg, flagset.Args()[0])
	if err != nil {
		return err
	}
	prefixLower := bindPrefix + pkgs[0].Name
	prefixUpper := strings.Title(bindPrefix) + strings.Title(pkgs[0].Name)

	args = append(args, "github.com/hajimehoshi/ebiten/mobile/ebitenmobileview")

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
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "GO111MODULE=off")
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

	switch buildTarget {
	case "android":
		// Do nothing.
	case "ios":
		dir := filepath.Join(buildO, "Versions", "A")

		if err := ioutil.WriteFile(filepath.Join(dir, "Headers", prefixUpper+"EbitenViewController.h"), []byte(replacePrefixes(objcH)), 0644); err != nil {
			return err
		}
		// TODO: Remove 'Ebitenmobileview.objc.h' here. Now it is hard since there is a header file importing
		// that header file.

		fs, err := ioutil.ReadDir(filepath.Join(dir, "Headers"))
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
		defer w.Close()
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

	return nil
}

const objcH = `// Code generated by ebitenmobile. DO NOT EDIT.

#import <UIKit/UIKit.h>

@interface {{.PrefixUpper}}EbitenViewController : UIViewController

// onErrorOnGameUpdate is called on the main thread when an error happens when updating a game.
// You can define your own error handler, e.g., using Crashlytics, by overwriting this method.
- (void)onErrorOnGameUpdate:(NSError*)err;

// suspendGame suspends the game.
// It is recommended to call this when the application is being suspended e.g.,
// UIApplicationDelegate's applicationWillResignActive is called.
- (void)suspendGame;

// resumeGame resumes the game.
// It is recommended to call this when the application is being resumed e.g.,
// UIApplicationDelegate's applicationDidBecomeActive is called.
- (void)resumeGame;

@end
`

var iosModuleMapTmpl = template.Must(template.New("iosmmap").Parse(`framework module "{{.Module}}" {
{{range .Headers}}    header "{{.}}"
{{end}}
    export *
}`))
