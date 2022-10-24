// Copyright 2022 The Ebitengine Authors
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

//go:build ignore
// +build ignore

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// files that should be compiled
var filenames = map[string]struct{}{
	"cocoa_init.m":     {},
	"cocoa_joystick.m": {},
	"cocoa_monitor.m":  {},
	"cocoa_time.c":     {},
	"cocoa_window.m":   {},
	"context.c":        {},
	"egl_context.c":    {},
	"init.c":           {},
	"input.c":          {},
	"monitor.c":        {},
	"nsgl_context.m":   {},
	"osmesa_context.c": {},
	"posix_thread.c":   {},
	"vulkan.c":         {},
	"window.c":         {},
}

type arch string

const (
	archAmd64 arch = "amd64"
	archArm64 arch = "arm64"
)

func (a arch) target() string {
	switch a {
	case archAmd64:
		return "x86_64"
	case archArm64:
		return "arm64"
	default:
		panic("not reached")
	}
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func objectExt(s string) string {
	return strings.Trim(strings.Trim(s, ".c"), ".m") + ".o"
}

func run() error {
	list := exec.Command("go", "list", "-f", "{{.Dir}}", "github.com/go-gl/glfw/v3.3/glfw")
	output, err := list.Output()
	if err != nil {
		return err
	}
	source := strings.TrimSpace(string(output))
	build, err := os.MkdirTemp("", "ebitengine-glfw-*")
	if err != nil {
		return err
	}
	defer func() {
		log.Printf("Deleting build files: %s", build)
		os.RemoveAll(build) // clean up
	}()
	csource := source + "/glfw/src"
	includes := source + "/glfw/include"
	dir, err := os.ReadDir(csource)
	if err != nil {
		return err
	}

	for _, a := range []arch{archAmd64, archArm64} {
		log.Printf("Compiling Files for %s\n", a.target())
		for _, entry := range dir {
			if _, ok := filenames[entry.Name()]; !ok {
				continue
			}
			name := entry.Name()
			//arch := archAmd64
			args := []string{
				"-o", // output
				objectExt(filepath.Join(build, name)),
				"-arch",
				a.target(),
				"-c", // compile without linking
				filepath.Join("-I", includes, "include/GLFW/glfw3.h"),       // include glfw3.h
				filepath.Join("-I", includes, "include/GLFW/glfw3native.h"), // include glfw3native.h
				"-D_GLFW_COCOA",              // define _GLFW_COCOA
				"-D_GLFW_BUILD_DLL",          // define _GLFW_BUILD_DLL to build dylib
				"-Os",                        // optimize for size
				"-g",                         // debug info
				filepath.Join(csource, name), // .c or .m file
			}
			err = execCommand("clang", args...)
			if err != nil {
				return err
			}
		}
		doto, err := filepath.Glob(filepath.Join(build, "*.o"))
		if err != nil {
			return err
		}
		log.Printf("Linking for %s\n", a.target())
		args := []string{
			"-dynamiclib",
			"-Wl",
			"-arch",
			a.target(),
			"-framework", "Cocoa", "-framework", "IOKit", "-framework", "CoreFoundation",
			"-o",
			"glfw-" + a.target() + ".dylib",
		}
		tmp := make([]string, len(args)+len(filenames))
		copy(tmp[copy(tmp, doto):], args)
		args = tmp
		err = execCommand("g++", args...)
		if err != nil {
			return err
		}
	}
	// There are now two files: glfw-arm64.dylib and glfw-x86_64.dylib
	log.Printf("Making universal dylib\n")
	err = execCommand("lipo", "glfw-arm64.dylib", "glfw-x86_64.dylib", "-output", "libglfw.3.3.dylib", "-create")
	if err != nil {
		return err
	}
	log.Printf("Deleting temporary dylibs")
	err = os.Remove("glfw-arm64.dylib")
	if err != nil {
		return err
	}
	os.Remove("glfw-x86_64.dylib")
	if err != nil {
		return err
	}
	log.Printf("Finished!\n")
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}