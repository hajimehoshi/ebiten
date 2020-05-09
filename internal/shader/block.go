// Copyright 2020 The Ebiten Authors
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

package shader

import (
	"fmt"
	"go/token"
	"strings"
)

type block struct {
	vars  []variable
	stmts []stmt
	pos   token.Pos
}

func (b *block) dump(indent int) []string {
	idt := strings.Repeat("\t", indent)

	var lines []string

	for _, v := range b.vars {
		lines = append(lines, fmt.Sprintf("%svar %s %s", idt, v.name, v.typ))
	}

	return lines
}

type stmt struct {
}
