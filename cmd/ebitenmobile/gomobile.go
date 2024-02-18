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
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	// Add a dependency on gomobile in order to get the version via debug.ReadBuildInfo().
	_ "github.com/ebitengine/gomobile/geom"
	exec "golang.org/x/sys/execabs"
)

//go:embed gobind.go
var gobind_go []byte

//go:embed _files/EbitenViewController.m
var objcM []byte

//go:embed _files/EbitenView.java
var viewJava []byte

//go:embed _files/EbitenSurfaceView.java
var surfaceViewJava []byte

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

func removeAll(path string) error {
	if buildX || buildN {
		fmt.Printf("rm -rf %s\n", path)
	}
	if buildN {
		return nil
	}
	return os.RemoveAll(path)
}

func runGo(args ...string) error {
	return runCommand("go", args, nil)
}

// exe adds the .exe extension to the given filename.
// Without .exe, the executable won't be found by exec.LookPath on Windows (#1096).
func exe(filename string) string {
	if runtime.GOOS == "windows" {
		return filename + ".exe"
	}
	return filename
}

func prepareGomobileCommands() (string, error) {
	tmp, err := os.MkdirTemp("", "ebitenmobile-")
	if err != nil {
		return "", err
	}

	newpath := filepath.Join(tmp, "bin")
	if path := os.Getenv("PATH"); path != "" {
		newpath += string(filepath.ListSeparator) + path
	}
	if buildX || buildN {
		fmt.Printf("PATH=%s\n", newpath)
	}
	if !buildN {
		if err := os.Setenv("PATH", newpath); err != nil {
			return tmp, err
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		return tmp, err
	}

	// cd
	if buildX {
		fmt.Printf("cd %s\n", tmp)
	}
	if err := os.Chdir(tmp); err != nil {
		return tmp, err
	}
	defer func() {
		_ = os.Chdir(pwd)
	}()

	const (
		modname   = "ebitenmobiletemporary"
		buildtags = "//go:build tools"
	)
	if err := runGo("mod", "init", modname); err != nil {
		return tmp, err
	}
	if err := os.WriteFile("tools.go", []byte(fmt.Sprintf(`%s

package %s

import (
	_ "github.com/ebitengine/gomobile/cmd/gobind"
	_ "github.com/ebitengine/gomobile/cmd/gomobile"
)
`, buildtags, modname)), 0644); err != nil {
		return tmp, err
	}

	h, err := gomobileHash()
	if err != nil {
		return tmp, err
	}

	// To record gomobile to go.sum for Go 1.16 and later, go-get gomobile instead of github.com/ebitengine/gomobile (#1487).
	// This also records gobind as gomobile depends on gobind indirectly.
	// Using `...` doesn't work on Windows since mobile/internal/mobileinit cannot be compiled on Windows w/o Cgo (#1493).
	if err := runGo("get", "github.com/ebitengine/gomobile/cmd/gomobile@"+h); err != nil {
		return tmp, err
	}
	if localgm := os.Getenv("EBITENMOBILE_GOMOBILE"); localgm != "" {
		if !filepath.IsAbs(localgm) {
			localgm = filepath.Join(pwd, localgm)
		}
		if err := runGo("mod", "edit", "-replace=github.com/ebitengine/gomobile="+localgm); err != nil {
			return tmp, err
		}
	}
	if err := runGo("mod", "tidy"); err != nil {
		return tmp, err
	}
	if err := runGo("build", "-o", exe(filepath.Join("bin", "gomobile")), "github.com/ebitengine/gomobile/cmd/gomobile"); err != nil {
		return tmp, err
	}
	if err := runGo("build", "-o", exe(filepath.Join("bin", "gobind-original")), "github.com/ebitengine/gomobile/cmd/gobind"); err != nil {
		return tmp, err
	}

	if err := os.Mkdir("src", 0755); err != nil {
		return tmp, err
	}
	if err := os.WriteFile(filepath.Join("src", "gobind.go"), gobind_go, 0644); err != nil {
		return tmp, err
	}

	if err := os.Mkdir(filepath.Join("src", "_files"), 0755); err != nil {
		return tmp, err
	}
	if err := os.WriteFile(filepath.Join("src", "_files", "EbitenViewController.m"), objcM, 0644); err != nil {
		return tmp, err
	}
	if err := os.WriteFile(filepath.Join("src", "_files", "EbitenView.java"), viewJava, 0644); err != nil {
		return tmp, err
	}
	if err := os.WriteFile(filepath.Join("src", "_files", "EbitenSurfaceView.java"), surfaceViewJava, 0644); err != nil {
		return tmp, err
	}

	// The newly added Go files like gobind.go might add new dependencies.
	if err := runGo("mod", "tidy"); err != nil {
		return tmp, err
	}
	if err := runGo("build", "-o", exe(filepath.Join("bin", "gobind")), "-tags", "ebitenmobilegobind", filepath.Join("src", "gobind.go")); err != nil {
		return tmp, err
	}

	if err := runCommand("gomobile", []string{"init"}, nil); err != nil {
		return tmp, err
	}

	return tmp, nil
}

func gomobileHash() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", fmt.Errorf("ebitenmobile: debug.ReadBuildInfo failed")
	}
	for _, m := range info.Deps {
		if m.Path == "github.com/ebitengine/gomobile" {
			return m.Version, nil
		}
	}
	return "", fmt.Errorf("ebitenmobile: getting the gomobile version failed")
}
