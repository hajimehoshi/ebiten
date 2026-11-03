// Copyright 2024 The Ebitengine Authors
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

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ebitenmobile "github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile"
)

func TestJavaPackageName(t *testing.T) {
	testCases := []struct {
		in  string
		out bool
	}{
		{
			in:  "",
			out: false,
		},
		{
			in:  ".",
			out: false,
		},
		{
			in:  "com.hajimehoshi.goinovation",
			out: true,
		},
		{
			in:  "com.hajimehoshi.$goinovation",
			out: true,
		},
		{
			in:  "com.hajimehoshi..goinovation",
			out: false,
		},
		{
			in:  "com.hajimehoshi.go-inovation",
			out: false,
		},
		{
			in:  "com.hajimehoshi.strictfp", // strictfp is a Java keyword.
			out: false,
		},
		{
			in:  "com.hajimehoshi.null",
			out: false,
		},
		{
			in:  "com.hajimehoshi.go1inovation",
			out: true,
		},
		{
			in:  "com.hajimehoshi.1goinovation",
			out: false,
		},
		{
			in:  "あ.いうえお",
			out: true,
		},
	}
	for _, tc := range testCases {
		if got, want := ebitenmobile.IsValidJavaPackageName(tc.in), tc.out; got != want {
			t.Errorf("isValidJavaPackageName(%q) = %v; want %v", tc.in, got, want)
		}
	}
}

func TestFrameworkName(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{
			in:  "MyGame.xcframework",
			out: "MyGame",
		},
		{
			in:  "mygame.xcframework",
			out: "Mygame",
		},
		{
			in:  "MG.xcframework",
			out: "MG",
		},
		{
			in:  "my_game.xcframework",
			out: "My_game",
		},
		{
			in:  filepath.Join("path", "to", "MyGame.xcframework"),
			out: "MyGame",
		},
	}
	for _, tc := range testCases {
		if got, want := ebitenmobile.FrameworkName(tc.in), tc.out; got != want {
			t.Errorf("frameworkName(%q) = %q; want %q", tc.in, got, want)
		}
	}
}

func TestGobindRunsGeneratorOnce(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command is unavailable")
	}
	dir := t.TempDir()
	wrapper := filepath.Join(dir, "gobind.exe")
	if out, err := exec.Command("go", "build", "-buildvcs=false", "-o", wrapper, "gobind.go").CombinedOutput(); err != nil {
		t.Fatalf("build wrapper: %v\n%s", err, out)
	}
	// Count actual subprocess invocations without requiring an Android or iOS SDK.
	helperSource := filepath.Join(dir, "helper.go")
	if err := os.WriteFile(helperSource, []byte(`package main
import "os"
func main() {
 f, err := os.OpenFile(os.Getenv("GOBIND_TEST_COUNT"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
 if err != nil { panic(err) }
 if _, err := f.WriteString("call\n"); err != nil { panic(err) }
 if err := f.Close(); err != nil { panic(err) }
 if os.Getenv("GOBIND_TEST_FAIL") != "" { os.Exit(7) }
}
`), 0600); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(dir, "gobind-original")
	if os.PathSeparator == '\\' {
		helper += ".exe"
	}
	if out, err := exec.Command("go", "build", "-buildvcs=false", "-o", helper, helperSource).CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}
	fixture := filepath.Join(dir, "fixture")
	if err := os.Mkdir(fixture, 0700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"go.mod": "module example.test/fixture\n\ngo 1.26.0\n", "fixture.go": "package fixture\n\nfunc Answer() int { return 42 }\n"} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, langs := range []string{"go,java", "go,objc"} {
		t.Run(langs, func(t *testing.T) {
			outdir := t.TempDir()
			for _, subdir := range []string{"java/fixture", "src/gobind"} {
				if err := os.MkdirAll(filepath.Join(outdir, subdir), 0700); err != nil {
					t.Fatal(err)
				}
			}
			countFile := filepath.Join(outdir, "calls")
			cmd := exec.Command(wrapper, "-lang="+langs, "-outdir="+outdir, ".")
			cmd.Dir = fixture
			// Package discovery must use an architecture supported by both
			// Android and iOS, even when this test runs on a 32-bit host.
			cmd.Env = append(os.Environ(), "GOARCH=amd64", "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "GOBIND_TEST_COUNT="+countFile)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("wrapper: %v\n%s", err, out)
			}
			calls, err := os.ReadFile(countFile)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(string(calls), "call\n"); got != 1 {
				t.Errorf("generator invocations: got %d, want 1", got)
			}
			files := []string{"java/fixture/EbitenView.java", "java/fixture/EbitenSurfaceView.java"}
			if langs == "go,objc" {
				files = []string{"src/gobind/fixtureebitenviewcontroller_ios.m", "src/gobind/fixtureebitenviewcontroller_ios.go"}
			}
			for _, name := range files {
				if _, err := os.Stat(filepath.Join(outdir, name)); err != nil {
					t.Error(err)
				}
			}
		})
	}
	t.Run("failure", func(t *testing.T) {
		countFile := filepath.Join(t.TempDir(), "calls")
		cmd := exec.Command(wrapper, "-lang=go,java", "-outdir="+t.TempDir(), ".")
		cmd.Dir = fixture
		cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "GOBIND_TEST_COUNT="+countFile, "GOBIND_TEST_FAIL=1")
		if err := cmd.Run(); err == nil {
			t.Fatal("wrapper succeeded after the generator failed")
		}
		calls, err := os.ReadFile(countFile)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(calls), "call\n"); got != 1 {
			t.Errorf("generator invocations: got %d, want 1", got)
		}
	})
}
