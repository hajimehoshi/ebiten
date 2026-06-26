// Copyright 2026 The Ebitengine Authors
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

package shaderprecomp_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/shaderprecomp"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func TestShaderSourceIDRoundTrip(t *testing.T) {
	want := shaderir.CalcSourceID([]byte("package main\n\nfunc Fragment() vec4 { return vec4(0) }")).String()
	id, err := shaderprecomp.ParseShaderSourceID(want)
	if err != nil {
		t.Fatal(err)
	}
	if got := id.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseShaderSourceIDInvalid(t *testing.T) {
	cases := []string{
		"",                                     // too short
		"not-base32!",                          // invalid base32 characters
		"aaaaaaaa",                             // valid base32 but wrong byte length
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // valid base32 but too long
	}
	for _, c := range cases {
		if _, err := shaderprecomp.ParseShaderSourceID(c); err == nil {
			t.Errorf("ParseShaderSourceID(%q) must fail", c)
		}
	}
}

func TestMustParseShaderSourceID(t *testing.T) {
	// 26 'a's is the base32 (no padding) encoding of 16 zero bytes.
	const zero = "aaaaaaaaaaaaaaaaaaaaaaaaaa"
	if got := shaderprecomp.MustParseShaderSourceID(zero).String(); got != zero {
		t.Errorf("got %q, want %q", got, zero)
	}

	defer func() {
		if recover() == nil {
			t.Error("MustParseShaderSourceID must panic for an invalid input")
		}
	}()
	shaderprecomp.MustParseShaderSourceID("invalid")
}

func TestRegisterGLSLDuplicatePanics(t *testing.T) {
	id := shaderprecomp.MustParseShaderSourceID(
		shaderir.CalcSourceID([]byte("exp-shaderprecomp-dup")).String())
	shaderprecomp.RegisterGLSL(id, []byte("vs"), []byte("fs"), nil, nil)

	defer func() {
		if recover() == nil {
			t.Error("registering GLSL twice for the same ID must panic")
		}
	}()
	shaderprecomp.RegisterGLSL(id, []byte("vs"), []byte("fs"), nil, nil)
}

// Example shows how generated code registers precompiled GLSL shaders.
func Example() {
	// The ID and the precompiled shaders are usually emitted as generated code from the
	// shadercollector tool's output, so the ID literal is always valid here.
	id := shaderprecomp.MustParseShaderSourceID("aaaaaaaaaaaaaaaaaaaaaaaaaa")

	// vertexGLSL and the others are the precompiled GLSL/GLSL ES shaders for the shader source.
	var vertexGLSL, fragmentGLSL, vertexGLSLES, fragmentGLSLES []byte
	shaderprecomp.RegisterGLSL(id, vertexGLSL, fragmentGLSL, vertexGLSLES, fragmentGLSLES)
}
