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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const gomobileHash = "f462b3930c8f01e0d9daa52b47ac887019ffa5b0"

func runCommand(command string, args []string, env []string) error {
	if buildX || buildN {
		for _, e := range env {
			fmt.Printf("%s ", e)
		}
		fmt.Print(command)
		for _, arg := range args {
			fmt.Printf(" %s", arg)
		}
		fmt.Println()
	}

	if buildN {
		return nil
	}

	cmd := exec.Command(command, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v failed: %v\n%v", command, args, string(out), err)
	}
	return nil
}

func runGo(args ...string) error {
	env := []string{
		"GO111MODULE=on",
	}
	return runCommand("go", args, env)
}

// exe adds the .exe extension to the given filename.
// Without .exe, the executable won't be found by exec.LookPath on Windows (#1096).
func exe(filename string) string {
	if runtime.GOOS == "windows" {
		return filename + ".exe"
	}
	return filename
}

func prepareGomobileCommands() error {
	tmp, err := ioutil.TempDir("", "ebitenmobile-")
	if err != nil {
		return err
	}

	newpath := filepath.Join(tmp, "bin")
	if path := os.Getenv("PATH"); path != "" {
		newpath += string(filepath.ListSeparator) + path
	}
	if buildX || buildN {
		fmt.Printf("PATH=%s\n", newpath)
	}
	if !buildN {
		os.Setenv("PATH", newpath)
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// cd
	if buildX {
		fmt.Printf("cd %s\n", tmp)
	}
	if err := os.Chdir(tmp); err != nil {
		return err
	}
	defer func() {
		os.Chdir(pwd)
	}()

	if err := runGo("mod", "init", "ebitenmobiletemporary"); err != nil {
		return err
	}

	// To record gomobile to go.sum for Go 1.16 and later, go-get gomobile instaed of golang.org/x/mobile (#1487).
	// This also records gobind as gomobile depends on gobind indirectly.
	// Using `...` doesn't work on Windows since mobile/internal/mobileinit cannot be compiled on Windows w/o Cgo (#1493).
	// Note that `go mod tidy` doesn't work since this removes all the indirect imports.
	if err := runGo("get", "golang.org/x/mobile/cmd/gomobile@"+gomobileHash); err != nil {
		return err
	}
	if localgm := os.Getenv("EBITENMOBILE_GOMOBILE"); localgm != "" {
		if !filepath.IsAbs(localgm) {
			localgm = filepath.Join(pwd, localgm)
		}
		if err := runGo("mod", "edit", "-replace=golang.org/x/mobile="+localgm); err != nil {
			return err
		}
	}
	if err := runGo("build", "-o", exe(filepath.Join("bin", "gomobile")), "golang.org/x/mobile/cmd/gomobile"); err != nil {
		return err
	}
	if err := runGo("build", "-o", exe(filepath.Join("bin", "gobind-original")), "golang.org/x/mobile/cmd/gobind"); err != nil {
		return err
	}

	if err := os.Mkdir("src", 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join("src", "gobind.go"), gobindsrc, 0644); err != nil {
		return err
	}

	if err := runGo("build", "-o", exe(filepath.Join("bin", "gobind")), "-tags", "ebitenmobilegobind", filepath.Join("src", "gobind.go")); err != nil {
		return err
	}

	if err := runCommand("gomobile", []string{"init"}, nil); err != nil {
		return err
	}

	return nil
}
