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
// ebitenmobile uses github.com/ebitengine/gomobile for gomobile, not golang.org/x/mobile.
// gomobile's version is fixed by ebitenmobile.
// You can specify gomobile's version by EBITENMOBILE_GOMOBILE environment variable.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/tools/go/packages"
)

const (
	ebitenmobileCommand = "ebitenmobile"
)

//go:embed _files/EbitenViewController.h
var objcH string

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
	flag.Usage = func() {
		// This message is copied from `gomobile bind -h`
		fmt.Fprintf(os.Stderr, "%s bind [-target android|ios] [-bootclasspath <path>] [-classpath <path>] [-o output] [build flags] [package]\n", ebitenmobileCommand)
		os.Exit(2)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
	}

	if args[0] != "bind" {
		flag.Usage()
	}

	// minAndroidAPI specifies the minimum API version for Android.
	// Now Google Player v23.30.99+ drops API levels that are older than 21.
	// See https://apilevels.com/.
	const minAndroidAPI = 21

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

		if !isValidJavaPackageName(bindJavaPkg) {
			log.Fatalf("invalid Java package name: %s", bindJavaPkg)
		}
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

	// If args doesn't include '-androidapi', set it to args explicitly.
	// It's because ebitenmobile's default API level is different from gomobile's one.
	if buildTarget == "android" && buildAndroidAPI == minAndroidAPI {
		var found bool
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "androidapi" {
				found = true
			}
		})
		if !found {
			args = append([]string{args[0], "-androidapi", fmt.Sprintf("%d", minAndroidAPI)}, args[1:]...)
		}
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
			dir := filepath.Join(buildO, name, frameworkNameBase+".framework")

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

func isValidJavaPackageName(name string) bool {
	if name == "" {
		return false
	}
	// A Java package name consists of one or more Java identifiers separated by dots.
	for _, token := range strings.Split(name, ".") {
		if !isValidJavaIdentifier(token) {
			return false
		}
	}
	return true
}

// isValidJavaIdentifier reports whether the given strings is a valid Java identifier.
func isValidJavaIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// Java identifiers must not be a Java keyword or a boolean/null literal.
	// https://docs.oracle.com/javase/specs/jls/se21/html/jls-3.html#jls-Identifier
	switch name {
	case "_", "abstract", "assert", "boolean", "break", "byte", "case", "catch", "char", "class", "const", "continue", "default", "do", "double", "else", "enum", "extends", "final", "finally", "float", "for", "goto", "if", "implements", "import", "instanceof", "int", "interface", "long", "native", "new", "package", "private", "protected", "public", "return", "short", "static", "strictfp", "super", "switch", "synchronized", "this", "throw", "throws", "transient", "try", "void", "volatile", "while":
		return false
	}
	if name == "null" || name == "true" || name == "false" {
		return false
	}

	// References:
	// * https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/Character.html#isJavaIdentifierPart(int)
	// * https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/Character.html#isJavaIdentifierStart(int)

	isJavaLetter := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.Is(unicode.Pc, r) || unicode.Is(unicode.Sc, r)
	}
	isJavaDigit := unicode.IsDigit

	// A Java identifier is a Java letter or Java letter followed by Java letters or Java digits.
	// https://docs.oracle.com/javase/specs/jls/se21/html/jls-3.html#jls-Identifier
	for i, r := range name {
		if !isJavaLetter(r) && (i == 0 || !isJavaDigit(r)) {
			return false
		}
	}
	return true
}
