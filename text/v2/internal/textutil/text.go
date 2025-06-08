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
	"iter"
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func Lines(str string) iter.Seq[string] {
	origStr := str
	return func(yield func(s string) bool) {
		var startIdx, endIdx int
		state := -1
		for len(str) > 0 {
			segment, nextStr, mustBreak, nextState := uniseg.FirstLineSegmentInString(str, state)
			endIdx += len(segment)
			if mustBreak {
				if !yield(origStr[startIdx:endIdx]) {
					return
				}
				startIdx = endIdx
			}
			state = nextState
			str = nextStr
		}
		if startIdx < endIdx {
			if !yield(origStr[startIdx:endIdx]) {
				return
			}
		}
	}
}

func TrimTailingLineBreak(str string) string {
	if !uniseg.HasTrailingLineBreakInString(str) {
		return str
	}

	// https://en.wikipedia.org/wiki/Newline#Unicode
	if strings.HasSuffix(str, "\r\n") {
		return str[:len(str)-2]
	}

	_, s := utf8.DecodeLastRuneInString(str)
	return str[:len(str)-s]
}
