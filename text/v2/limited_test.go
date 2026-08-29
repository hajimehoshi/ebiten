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

package text

import (
	"testing"
	"unicode/utf8"
)

func TestLimitedFilterMapping(t *testing.T) {
	testCases := []struct {
		name     string
		line     string
		filtered string
	}{
		{
			name:     "no replacements",
			line:     "abc",
			filtered: "abc",
		},
		{
			name:     "2-byte rune replaced",
			line:     "aébc",
			filtered: "a\uFFFD" + "bc",
		},
		{
			name:     "3-byte rune replaced",
			line:     "aあbc",
			filtered: "a\uFFFD" + "bc",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapping := limitedFilterMapping(tc.line, tc.filtered)
			if len(mapping) != len(tc.filtered)+1 {
				t.Fatalf("len(mapping) = %d, want %d", len(mapping), len(tc.filtered)+1)
			}
			for fi := range tc.filtered {
				oi := mapping[fi]
				if oi >= len(tc.line) {
					t.Errorf("mapping[%d] = %d out of range", fi, oi)
					continue
				}
				if !utf8.ValidRune(rune(tc.line[oi])) && oi != len(tc.line) {
					t.Errorf("mapping[%d] = %d is not a rune boundary", fi, oi)
				}
			}
			if got := mapping[len(tc.filtered)]; got != len(tc.line) {
				t.Errorf("mapping[len(filtered)] = %d, want %d", got, len(tc.line))
			}
		})
	}
}
