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
		File       string
		Source     string
		SourceHash string
	}
	var shaders []shader
	if err := json.Unmarshal(out, &shaders); err != nil {
		t.Fatal(err)
	}

	slices.SortFunc(shaders, func(s1, s2 shader) int {
		return cmp.Compare(s1.Source, s2.Source)
	})

	if got, want := len(shaders), 6; got != want {
		t.Fatalf("len(shaders): got: %d, want: %d", got, want)
	}

	for i, s := range shaders {
		if s.Package == "" {
			t.Errorf("s.Package is empty: %v", s)
		}
		if s.File == "" {
			t.Errorf("s.File is empty: %v", s)
		}
		hash, err := graphics.CalcSourceHash([]byte(s.Source))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := s.SourceHash, hash.String(); got != want {
			t.Errorf("s.SourceHash: got: %q, want: %q", got, want)
		}
		if got, want := s.Source, fmt.Sprintf("shader %d", i+1); got != want {
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
