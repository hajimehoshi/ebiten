// Copyright 2025 The Ebitengine Authors
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

package textutil_test

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/textutil"
)

func TestUnicodeRangeFilter(t *testing.T) {
	testCases := []struct {
		name     string
		ranges   textutil.UnicodeRanges
		input    string
		expected string
	}{
		{
			name:     "empty ranges",
			ranges:   textutil.UnicodeRanges{},
			input:    "Hello, 世界",
			expected: strings.Repeat("\ufffd", 9),
		},
		{
			name: "allow ASCII",
			ranges: func() textutil.UnicodeRanges {
				var r textutil.UnicodeRanges
				r.Add(0x00, 0x7F)
				return r
			}(),
			input:    "Hello, 世界",
			expected: "Hello, \ufffd\ufffd",
		},
		{
			name: "allow CJK",
			ranges: func() textutil.UnicodeRanges {
				var r textutil.UnicodeRanges
				r.Add(0x4E00, 0x9FFF)
				return r
			}(),
			input:    "Hello, 世界",
			expected: "\ufffd\ufffd\ufffd\ufffd\ufffd\ufffd\ufffd世界",
		},
		{
			name: "allow all",
			ranges: func() textutil.UnicodeRanges {
				var r textutil.UnicodeRanges
				r.Add(0x00, 0x7F)
				r.Add(0x4E00, 0x9FFF)
				return r
			}(),
			input:    "Hello, 世界",
			expected: "Hello, 世界",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.ranges.Filter(tc.input)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
