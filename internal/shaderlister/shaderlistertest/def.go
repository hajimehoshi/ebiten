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

package shaderlistertest

import (
	"github.com/hajimehoshi/ebiten/v2/internal/shaderlister/shaderlistertest2"
)

//ebitengine:shadersource
const _ = "shader 1"

const (
	//ebitengine:shadersource
	_ = "shader 2"

	//ebitengine:shadersource
	a = "shader 3"

	//ebitengine:invalid
	b = "not shader"

	//ebitengine:shadersource
	c = "shader" + " 4"
)

//ebitengine:invalid
const _ = "not shader"

//ebitengine:shadersource
const d = shaderlistertest2.S + " 5"

const _ = "not shader"

//ebitengine:shadersource
const (
	_ = "ignored" // The directive is misplaced.
)

//ebitengine:shadersource
var _ = "ignored" // The directive doesn't work for var.

func f() {
	//ebitengine:shadersource
	const _ = "ignored" // The directive doesn't work for non-top-level const.

	const (
		//ebitengine:shadersource
		_ = "ignored" // The directive doesn't work for non-top-level const.
	)
}

//ebitengine:shadersource
const _, _ = "ignored", "ignored again" // multiple consts are ignored to avoid confusion.

const (
	//ebitengine:shadersource
	_, _ = "ignored", "ignored again" // multiple consts are ignored to avoid confusion.
)

//ebitengine:shaderfile *_kage.go resource nonexistent.go

// Duplicated files are ignored.
//ebitengine:shaderfile *_kage.go *_kage.go *_kage.go

//ebitengine:shaderfile nonexistent.go

func foo() {
	// Non top-level files are ignored.

	//ebitengine:shaderfile *_notkage.go
}

// A directive in a comment block is not ignored.
/*
//ebitengine:shaderfile *_kage.go
*/
