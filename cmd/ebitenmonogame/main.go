// Copyright 2020 The Ebiten Authors
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
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/go2dotnet/gowasm2csharp"
)

const ebitenmonogameName = "ebitenmonogame"

var (
	flagA        bool   // -a
	flagGcflags  string // -gcflags
	flagI        bool   // -i
	flagLdflags  string // -ldflags
	flagN        bool   // -n
	flagTags     string // -tags
	flagTrimpath bool   // -trimpath
	flagV        bool   // -v
	flagWork     bool   // -work
	flagX        bool   // -x

	flagNamespace string // -namespace
	flagO         string // -o
)

func main() {
	if err := checkGOOS(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", ebitenmonogameName, err)
		os.Exit(2)
	}

	var flagset flag.FlagSet
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [-namespace namespace] [-o output] [build flags] [package]\n", ebitenmonogameName)
		os.Exit(2)
	}

	flagset.BoolVar(&flagA, "a", false, "")
	flagset.StringVar(&flagGcflags, "gcflags", "", "")
	flagset.BoolVar(&flagI, "i", false, "")
	flagset.StringVar(&flagLdflags, "ldflags", "", "")
	flagset.BoolVar(&flagN, "n", false, "")
	flagset.StringVar(&flagTags, "tags", "", "")
	flagset.BoolVar(&flagTrimpath, "trimpath", false, "")
	flagset.BoolVar(&flagV, "v", false, "")
	flagset.BoolVar(&flagWork, "work", false, "")
	flagset.BoolVar(&flagX, "x", false, "")

	flagset.StringVar(&flagNamespace, "namespace", "", "")
	flagset.StringVar(&flagO, "o", "", "")

	flagset.Parse(os.Args[1:])

	if flagNamespace == "" {
		fmt.Fprintln(os.Stderr, "-namespace must be specified")
		os.Exit(2)
	}
	if flagO == "" {
		fmt.Fprintln(os.Stderr, "-o must be specified")
		os.Exit(2)
	}

	if flagLdflags != "" {
		flagLdflags += " "
	}
	flagLdflags = "-X github.com/hajimehoshi/ebiten/internal/monogame.namespace=" + flagNamespace
	if flagTags != "" {
		flagTags += ","
	}
	flagTags += "monogame"

	var src string
	switch srcs := flagset.Args(); len(srcs) {
	case 0:
		src = "."
	case 1:
		src = srcs[0]
	default:
		flag.Usage()
	}
	if err := run(src); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", ebitenmonogameName, err)
		os.Exit(2)
	}
}

func checkGOOS() error {
	out, err := exec.Command("go", "env", "GOOS").Output()
	if err != nil {
		return err
	}
	goos := strings.TrimSpace(string(out))
	if goos != "windows" {
		fmt.Fprintln(os.Stderr, "Warning: The output project is buildable on Windows.")
	}
	return nil
}

func run(src string) error {
	// TODO: Check src is a main package?

	// Generate a wasm binary.
	env := []string{
		"GOOS=js",
		"GOARCH=wasm",
	}

	dir, err := ioutil.TempDir("", "ebitenmonogame")
	if err != nil {
		return err
	}
	defer func() {
		if flagWork {
			fmt.Fprintf(os.Stderr, "The temporary work directory: %s\n", dir)
			return
		}
		os.RemoveAll(dir)
	}()

	wasmFile := filepath.Join(dir, "tmp.wasm")

	if err := runGo("build", []string{src}, env, "-o", wasmFile); err != nil {
		return err
	}
	if flagN {
		return nil
	}

	autogenDir := filepath.Join(flagO, "autogen")
	if flagV {
		fmt.Fprintf(os.Stderr, "Writing %s%s*.cs\n", autogenDir, string(filepath.Separator))
	}
	if err := os.MkdirAll(autogenDir, 0755); err != nil {
		return err
	}
	if err := gowasm2csharp.Generate(autogenDir, wasmFile, flagNamespace+".AutoGen"); err != nil {
		return err
	}

	if err := writeFile(filepath.Join(flagO, "GoGame.cs"), replaceNamespace(gogame_cs)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(flagO, "Program.cs"), replaceNamespace(program_cs)); err != nil {
		return err
	}
	abs, err := filepath.Abs(flagO)
	if err != nil {
		return err
	}
	project := filepath.Base(abs)
	if err := writeFile(filepath.Join(flagO, project+".csproj"), replaceNamespace(project_csproj)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(flagO, "Content", "Content.mgcb"), replaceNamespace(content_mgcb)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(flagO, "Content", "Shader.fx"), replaceNamespace(shader_fx)); err != nil {
		return err
	}

	return nil
}

func runGo(subcmd string, srcs []string, env []string, args ...string) error {
	cmd := exec.Command("go", subcmd)
	if flagA {
		cmd.Args = append(cmd.Args, "-a")
	}
	if flagGcflags != "" {
		cmd.Args = append(cmd.Args, "-gcflags", flagGcflags)
	}
	if flagI {
		cmd.Args = append(cmd.Args, "-i")
	}
	if flagLdflags != "" {
		cmd.Args = append(cmd.Args, "-ldflags", flagLdflags)
	}
	if flagTags != "" {
		cmd.Args = append(cmd.Args, "-tags", flagTags)
	}
	if flagTrimpath {
		cmd.Args = append(cmd.Args, "-trimpath")
	}
	if flagV {
		cmd.Args = append(cmd.Args, "-v")
	}
	if flagWork {
		cmd.Args = append(cmd.Args, "-work")
	}
	if flagX {
		cmd.Args = append(cmd.Args, "-x")
	}
	cmd.Args = append(cmd.Args, args...)
	cmd.Args = append(cmd.Args, srcs...)
	if len(env) > 0 {
		cmd.Env = append([]string{}, env...)
	}

	if flagX || flagN {
		env := strings.Join(cmd.Env, " ")
		if env != "" {
			env += " "
		}
		fmt.Fprintf(os.Stderr, "%s%s\n", env, strings.Join(cmd.Args, " "))
	}
	if flagN {
		return nil
	}

	if len(cmd.Env) > 0 {
		cmd.Env = append(os.Environ(), cmd.Env...)
	}

	buf := &bytes.Buffer{}
	buf.WriteByte('\n')
	if flagV {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = buf
		cmd.Stderr = buf
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %v%s", strings.Join(cmd.Args, " "), err, buf)
	}
	return nil
}

func writeFile(dst string, src []byte) error {
	if flagV {
		fmt.Fprintf(os.Stderr, "Writing %s\n", dst)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(dst, src, 0644); err != nil {
		return err
	}
	return nil
}

func replaceNamespace(src []byte) []byte {
	return []byte(strings.ReplaceAll(string(src), "{{.Namespace}}", flagNamespace))
}
