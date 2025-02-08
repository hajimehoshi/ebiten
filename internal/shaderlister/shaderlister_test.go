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
	"cmp"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
)

func hasGoCommand() bool {
	if _, err := exec.LookPath("go"); err != nil {
		return false
	}
	return true
}

func TestRun(t *testing.T) {
	if !hasGoCommand() {
		t.Skip("go command is missing")
	}

	cmd := exec.Command("go", "run", "github.com/hajimehoshi/ebiten/v2/internal/shaderlister", "github.com/hajimehoshi/ebiten/v2/internal/shaderlister/shaderlistertest")
	out, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Error: %v\n%s", err, err.Stderr)
		}
		t.Fatal(err)
	}

	type shader struct {
		Package    string
		GoFile     string
		KageFile   string
		Source     string
		SourceHash string
	}
	var shaders []shader
	if err := json.Unmarshal(out, &shaders); err != nil {
		t.Fatal(err)
	}

	type filteredShader struct {
		shader         shader
		filteredSource string
	}

	re := regexp.MustCompile(`shader \d+`)
	var filteredShaders []filteredShader
	for _, s := range shaders {
		m := re.FindAllString(s.Source, 1)
		if len(m) != 1 {
			t.Fatalf("invalid source: %q", s.Source)
		}
		filteredShaders = append(filteredShaders, filteredShader{
			shader:         s,
			filteredSource: m[0],
		})
	}

	slices.SortFunc(filteredShaders, func(s1, s2 filteredShader) int {
		return cmp.Compare(s1.filteredSource, s2.filteredSource)
	})

	if got, want := len(filteredShaders), 9; got != want {
		t.Errorf("len(shaders): got: %d, want: %d", got, want)
	}

	for i, s := range filteredShaders {
		if s.shader.Package == "" {
			t.Errorf("s.Package is empty: %v", s)
		}
		if s.shader.GoFile == "" {
			t.Errorf("s.File is empty: %v", s)
		}
		// KageFile can be empty.
		hash, err := graphics.CalcSourceHash([]byte(s.shader.Source))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := s.shader.SourceHash, hash.String(); got != want {
			t.Errorf("s.SourceHash: got: %q, want: %q", got, want)
		}
		if got, want := s.filteredSource, fmt.Sprintf("shader %d", i+1); got != want {
			t.Errorf("s.Source: got: %q, want: %q", got, want)
		}
	}
}

func TestEmpty(t *testing.T) {
	if !hasGoCommand() {
		t.Skip("go command is missing")
	}

	cmd := exec.Command("go", "run", "github.com/hajimehoshi/ebiten/v2/internal/shaderlister", "github.com/ebitengine/purego")
	out, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Error: %v\n%s", err, err.Stderr)
		}
		t.Fatal(err)
	}

	// Check the output is `[]`, not `null`.
	if got, want := strings.TrimSpace(string(out)), "[]"; got != want {
		t.Errorf("output: got: %q, want: %q", got, want)
	}
}
