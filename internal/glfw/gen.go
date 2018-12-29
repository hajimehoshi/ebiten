// Copyright 2018 The Ebiten Authors
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

// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"
)

var srcs = []string{
	"glfw/src/context.c",
	"glfw/src/init.c",
	"glfw/src/input.c",
	"glfw/src/monitor.c",
	"glfw/src/vulkan.c",
	"glfw/src/window.c",
	"glfw/src/win32_init.c",
	"glfw/src/win32_joystick.c",
	"glfw/src/win32_monitor.c",
	"glfw/src/win32_time.c",
	"glfw/src/win32_tls.c",
	"glfw/src/win32_window.c",
	"glfw/src/wgl_context.c",
	"glfw/src/egl_context.c",
}

type arch string

const (
	archAmd64 arch = "amd64"
	arch386   arch = "386"
)

func (a arch) gcc() string {
	switch a {
	case archAmd64:
		return "x86_64-w64-mingw32-gcc"
	case arch386:
		return "i686-w64-mingw32-gcc"
	default:
		panic("not reached")
	}
}

func c2o(str string) string {
	return strings.TrimSuffix(filepath.Base(str), ".c") + ".o"
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func run() error {
	for _, arch := range []arch{archAmd64, arch386} {
		g := errgroup.Group{}
		for _, s := range srcs {
			s := s
			g.Go(func() error {
				args := []string{
					"-c",
					"-o",
					c2o(s),
					"-D_GLFW_WIN32",
					"-D_GLFW_BUILD_DLL",
					"-Iglfw/deps/mingw",
					"-g",
					"-Os",
					s,
				}
				if err := execCommand(arch.gcc(), args...); err != nil {
					return err
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}

		objs := []string{}
		for _, s := range srcs {
			objs = append(objs, c2o(s))
		}
		dll := fmt.Sprintf("glfw_windows_%s.dll", arch)
		args := []string{
			"-shared",
			"-o",
			dll,
			"-Wl,--no-insert-timestamp",
		}
		args = append(args, objs...)
		args = append(args, "-lopengl32", "-lgdi32")
		if err := execCommand(arch.gcc(), args...); err != nil {
			return err
		}

		args = []string{
			"-input",
			dll,
			"-output",
			fmt.Sprintf("glfwdll_windows_%s.go", arch),
			"-package",
			"glfw",
			"-var",
			"glfwDLLCompressed",
			"-compress",
		}
		if err := execCommand("file2byteslice", args...); err != nil {
			return err
		}

		for _, o := range objs {
			if err := os.Remove(o); err != nil {
				return err
			}
		}
		if err := os.Remove(dll); err != nil {
			return err
		}
	}

	if err := execCommand("gofmt", "-s", "-w", "."); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
