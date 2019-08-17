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
)

const gomobileHash = "597adff16ade9d88626f8caea514bb189b8c74ee"

func runGo(args ...string) error {
	env := []string{
		"GO111MODULE=on",
	}

	if buildX || buildN {
		for _, e := range env {
			fmt.Printf("%s ", e)
		}
		fmt.Print("go")
		for _, arg := range args {
			fmt.Printf(" %s", arg)
		}
		fmt.Println()
	}

	if buildN {
		return nil
	}

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go %v failed: %v\n%v", args, string(out), err)
	}
	return nil
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
	os.Setenv("PATH", newpath)

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
	if err := runGo("get", "golang.org/x/mobile@"+gomobileHash); err != nil {
		return err
	}
	if err := runGo("build", "-o", filepath.Join("bin", "gomobile"), "golang.org/x/mobile/cmd/gomobile"); err != nil {
		return err
	}
	if err := runGo("build", "-o", filepath.Join("bin", "gobind-original"), "golang.org/x/mobile/cmd/gobind"); err != nil {
		return err
	}

	if err := os.Mkdir("src", 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join("src", "gobind.go"), gobindsrc, 0644); err != nil {
		return err
	}

	if err := runGo("build", "-o", filepath.Join("bin", "gobind"), "-tags", "ebitenmobilegobind", filepath.Join("src", "gobind.go")); err != nil {
		return err
	}

	// TODO: Create a gobind wrapper

	return nil
}
