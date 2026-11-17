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
	"flag"
	"fmt"
	"path/filepath"
	"reflect"
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

func TestAppendBindLdflags(t *testing.T) {
	for _, args := range [][]string{
		{"bind"},
		{"bind", "./game"},
		{"bind", "--", "./game", "./other"},
		{"bind", "-ldflags", "-X main.Name=value with spaces", "./game"},
		{"bind", "-ldflags=-s -w", "--", "./game"},
		{"bind", "-ldflags=-s", "-ldflags", "-w", "./game"},
	} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			fs := flag.NewFlagSet("", flag.ContinueOnError)
			var ldflags string
			fs.StringVar(&ldflags, "ldflags", "", "")
			if err := fs.Parse(args[1:]); err != nil {
				t.Fatal(err)
			}
			packages := append([]string(nil), fs.Args()...)
			want := ldflags
			if want != "" {
				want += " "
			}
			want += "-extldflags=-Wl,-soname,libgojni.so"
			forwarded := ebitenmobile.AppendBindLdflags(args, fs, want)
			if err := fs.Parse(forwarded[1:]); err != nil {
				t.Fatal(err)
			}
			if ldflags != want {
				t.Errorf("ldflags: got %q, want %q", ldflags, want)
			}
			if !reflect.DeepEqual(append([]string(nil), fs.Args()...), packages) {
				t.Errorf("packages: got %v, want %v", fs.Args(), packages)
			}
		})
	}
}
