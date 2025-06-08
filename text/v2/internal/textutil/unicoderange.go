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

package textutil

import (
	"strings"
)

type UnicodeRange struct {
	start rune
	end   rune
}

type UnicodeRanges struct {
	ranges []UnicodeRange
}

func (u *UnicodeRanges) Add(start, end rune) {
	u.ranges = append(u.ranges, UnicodeRange{
		start: start,
		end:   end,
	})
}

func (u *UnicodeRanges) Contains(r rune) bool {
	for _, rg := range u.ranges {
		if rg.start <= r && r <= rg.end {
			return true
		}
	}
	return false
}

func (u *UnicodeRanges) Filter(str string) string {
	replaceStartIndex := -1
	for i, r := range str {
		if !u.Contains(r) {
			replaceStartIndex = i
			break
		}
	}
	if replaceStartIndex < 0 {
		return str
	}

	var builder strings.Builder
	builder.Grow(len(str))
	_, _ = builder.WriteString(str[:replaceStartIndex])
	for _, r := range str[replaceStartIndex:] {
		if !u.Contains(r) {
			// U+FFFD is "REPLACEMENT CHARACTER".
			r = '\ufffd'
		}
		_, _ = builder.WriteRune(r)
	}
	return builder.String()
}
