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
	"maps"
	"os"
	"os/exec"
	"path/filepath"
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

	cmd := exec.Command("go", "run", "github.com/hajimehoshi/ebiten/v2/internal/shadercollector", "github.com/hajimehoshi/ebiten/v2/internal/shadercollector/testdata/shadercollectortest")
	out, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Error: %v\n%s", err, err.Stderr)
		}
		t.Fatal(err)
	}

	type shader struct {
		Package  string
		GoFile   string
		KageFile string
		Source   string
		SourceID string
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
		hash, err := graphics.CalcSourceID([]byte(s.shader.Source))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := s.shader.SourceID, hash.String(); got != want {
			t.Errorf("s.SourceID: got: %q, want: %q", got, want)
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

	cmd := exec.Command("go", "run", "github.com/hajimehoshi/ebiten/v2/internal/shadercollector", "github.com/ebitengine/purego")
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

func TestManifest(t *testing.T) {
	if !hasGoCommand() {
		t.Skip("go command is missing")
	}

	dir := filepath.Join("testdata", "shadercollectortestfiles")
	// These are the files listed in manifest.json, whose paths are resolved relative to the manifest.
	files := []string{
		filepath.Join(dir, "single.kage"),
		filepath.Join(dir, "dir", "nested.kage"),
		filepath.Join(dir, "dir", "sub", "deep.kage"),
	}

	// Compute the expected source hashes directly from the listed files.
	// These must match the hashes the tool emits, which in turn match the hashes
	// Ebitengine computes at runtime from the same bytes.
	wantHashes := map[string]struct{}{}
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		hash, err := graphics.CalcSourceID(content)
		if err != nil {
			t.Fatal(err)
		}
		wantHashes[hash.String()] = struct{}{}
	}

	// Pass a manifest file without any package. Its listed paths are resolved relative to the manifest.
	cmd := exec.Command("go", "run", "github.com/hajimehoshi/ebiten/v2/internal/shadercollector",
		"-target", "hlsl",
		"-manifest", filepath.Join(dir, "manifest.json"))
	out, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Error: %v\n%s", err, err.Stderr)
		}
		t.Fatal(err)
	}

	type hlsl struct {
		Vertex string
		Pixel  string
	}
	type shader struct {
		Package  string
		GoFile   string
		KageFile string
		Source   string
		SourceID string
		HLSL     *hlsl
	}
	var shaders []shader
	if err := json.Unmarshal(out, &shaders); err != nil {
		t.Fatal(err)
	}

	if got, want := len(shaders), len(files); got != want {
		t.Fatalf("len(shaders): got: %d, want: %d", got, want)
	}

	gotHashes := map[string]struct{}{}
	for _, s := range shaders {
		// A shader listed via -manifest is not tied to any package.
		if s.Package != "" {
			t.Errorf("s.Package: got: %q, want empty", s.Package)
		}
		if s.GoFile != "" {
			t.Errorf("s.GoFile: got: %q, want empty", s.GoFile)
		}
		if s.KageFile == "" {
			t.Errorf("s.KageFile is empty: %v", s)
		}

		hash, err := graphics.CalcSourceID([]byte(s.Source))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := s.SourceID, hash.String(); got != want {
			t.Errorf("s.SourceID: got: %q, want: %q", got, want)
		}
		gotHashes[s.SourceID] = struct{}{}

		if s.HLSL == nil {
			t.Errorf("s.HLSL is nil: %v", s)
			continue
		}
		if s.HLSL.Vertex == "" {
			t.Errorf("s.HLSL.Vertex is empty: %v", s)
		}
		if s.HLSL.Pixel == "" {
			t.Errorf("s.HLSL.Pixel is empty: %v", s)
		}
	}

	// Every file listed in the manifest must appear.
	if !maps.Equal(gotHashes, wantHashes) {
		t.Errorf("source hashes: got: %v, want: %v", gotHashes, wantHashes)
	}
}

func TestGLSLTarget(t *testing.T) {
	if !hasGoCommand() {
		t.Skip("go command is missing")
	}

	dir := filepath.Join("testdata", "shadercollectortestfiles")
	cmd := exec.Command("go", "run", "github.com/hajimehoshi/ebiten/v2/internal/shadercollector",
		"-target", "glsl",
		"-manifest", filepath.Join(dir, "manifest.json"))
	out, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Error: %v\n%s", err, err.Stderr)
		}
		t.Fatal(err)
	}

	type glsl struct {
		Vertex   string
		Fragment string
	}
	type shader struct {
		Source string
		GLSL   *glsl
		GLSLES *glsl
	}
	var shaders []shader
	if err := json.Unmarshal(out, &shaders); err != nil {
		t.Fatal(err)
	}

	if len(shaders) == 0 {
		t.Fatal("no shaders were emitted")
	}

	for _, s := range shaders {
		if s.GLSL == nil {
			t.Errorf("s.GLSL is nil: %v", s)
			continue
		}
		if s.GLSL.Vertex == "" {
			t.Errorf("s.GLSL.Vertex is empty: %v", s)
		}
		if s.GLSL.Fragment == "" {
			t.Errorf("s.GLSL.Fragment is empty: %v", s)
		}

		if s.GLSLES == nil {
			t.Errorf("s.GLSLES is nil: %v", s)
			continue
		}
		if s.GLSLES.Vertex == "" {
			t.Errorf("s.GLSLES.Vertex is empty: %v", s)
		}
		if s.GLSLES.Fragment == "" {
			t.Errorf("s.GLSLES.Fragment is empty: %v", s)
		}
	}
}
