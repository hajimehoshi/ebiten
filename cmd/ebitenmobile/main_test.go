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

package main

import (
	"testing"
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
		if got, want := isValidJavaPackageName(tc.in), tc.out; got != want {
			t.Errorf("isValidJavaPackageName(%q) = %v; want %v", tc.in, got, want)
		}
	}
}
