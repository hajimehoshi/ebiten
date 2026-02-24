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
)

func Lines(str string) iter.Seq[string] {
	return func(yield func(s string) bool) {
		for len(str) > 0 {
			idx := strings.IndexAny(str, "\n\v\f\r\u0085\u2028\u2029")
			var length int
			if idx < 0 {
				length = len(str)
			} else {
				_, size := utf8.DecodeRuneInString(str[idx:])
				length = idx + size
				if str[idx] == '\r' && length < len(str) && str[length] == '\n' {
					length++
				}
			}
			if !yield(str[:length]) {
				return
			}
			str = str[length:]
		}
	}
}

func TrimTailingLineBreak(str string) string {
	// https://en.wikipedia.org/wiki/Newline#Unicode

	if strings.HasSuffix(str, "\r\n") {
		return str[:len(str)-2]
	}

	r, s := utf8.DecodeLastRuneInString(str)
	if r == '\n' || r == '\v' || r == '\f' || r == '\r' || r == '\u0085' || r == '\u2028' || r == '\u2029' {
		return str[:len(str)-s]
	}
	return str
}
