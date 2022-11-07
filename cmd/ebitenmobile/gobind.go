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

//go:build ebitenmobilegobind

// gobind is a wrapper of the original gobind. This command adds extra files like a view controller.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	exec "golang.org/x/sys/execabs"
	"golang.org/x/tools/go/packages"
)

//go:embed _files/EbitenViewController.m
var objcM string

//go:embed _files/EbitenView.java
var viewJava string

//go:embed _files/EbitenSurfaceView.java
var surfaceViewJava string

var (
	lang          = flag.String("lang", "", "")
	outdir        = flag.String("outdir", "", "")
	javaPkg       = flag.String("javapkg", "", "")
	prefix        = flag.String("prefix", "", "")
	bootclasspath = flag.String("bootclasspath", "", "")
	classpath     = flag.String("classpath", "", "")
	tags          = flag.String("tags", "", "")
)

var usage = `The Gobind tool generates Java language bindings for Go.

For usage details, see doc.go.`

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func invokeOriginalGobind(lang string) (pkgName string, err error) {
	cmd := exec.Command("gobind-original", os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	cfgtags := strings.Join(strings.Split(*tags, ","), " ")
	cfg := &packages.Config{}
	switch lang {
	case "java":
		cfg.Env = append(os.Environ(), "GOOS=android")
	case "objc":
		cfg.Env = append(os.Environ(), "GOOS=darwin")
		if cfgtags != "" {
			cfgtags += " "
		}
		cfgtags += "ios"
	}
	cfg.BuildFlags = []string{"-tags", cfgtags}
	pkgs, err := packages.Load(cfg, flag.Args()[0])
	if err != nil {
		return "", err
	}
	return pkgs[0].Name, nil
}

func run() error {
	writeFile := func(filename string, content string) error {
		if err := os.WriteFile(filepath.Join(*outdir, filename), []byte(content), 0644); err != nil {
			return err
		}
		return nil
	}

	// Add additional files.
	langs := strings.Split(*lang, ",")
	for _, lang := range langs {
		pkgName, err := invokeOriginalGobind(lang)
		if err != nil {
			return err
		}
		prefixLower := *prefix + pkgName
		prefixUpper := strings.Title(*prefix) + strings.Title(pkgName)
		replacePrefixes := func(content string) string {
			content = strings.ReplaceAll(content, "{{.PrefixUpper}}", prefixUpper)
			content = strings.ReplaceAll(content, "{{.PrefixLower}}", prefixLower)
			content = strings.ReplaceAll(content, "{{.JavaPkg}}", *javaPkg)
			return content
		}

		switch lang {
		case "objc":
			// iOS
			if err := writeFile(filepath.Join("src", "gobind", prefixLower+"ebitenviewcontroller_ios.m"), replacePrefixes(objcM)); err != nil {
				return err
			}
			if err := writeFile(filepath.Join("src", "gobind", prefixLower+"ebitenviewcontroller_ios.go"), `package main

// #cgo CFLAGS: -DGLES_SILENCE_DEPRECATION
import "C"`); err != nil {
				return err
			}
		case "java":
			// Android
			dir := filepath.Join(strings.Split(*javaPkg, ".")...)
			dir = filepath.Join(dir, prefixLower)
			if err := writeFile(filepath.Join("java", dir, "EbitenView.java"), replacePrefixes(viewJava)); err != nil {
				return err
			}
			if err := writeFile(filepath.Join("java", dir, "EbitenSurfaceView.java"), replacePrefixes(surfaceViewJava)); err != nil {
				return err
			}
		case "go":
			// Do nothing.
		default:
			panic(fmt.Sprintf("unsupported language: %s", lang))
		}
	}

	return nil
}
