// Copyright 2022 The Ebiten Authors
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

package shader_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func compileToIR(src []byte) (*shaderir.Program, error) {
	return shader.Compile(src, "Vertex", "Fragment", 0)
}

func TestSyntaxShadowing(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var dstPos vec4
	return dstPos
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxDuplicatedVariables(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var foo vec4
	var foo vec4
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var foo, foo vec4
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var foo vec4
	foo := vec4(0)
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (vec4, vec4) {
	return vec4(0), vec4(0)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	foo, foo := Foo()
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func foo() {
	var x int
	var y float
	var x vec2
	_ = x
	_ = y
}

`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

func TestSyntaxDuplicatedFunctions(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Foo() {
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxNoNewVariables(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	_ := 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	_, _ := 1, 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (int, int) {
	return 1, 1
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	_, _ := Foo()
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a, _ := 1, 1
	_ = a
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	_, a := 1, 1
	_ = a
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}
}

func TestSyntaxWrongReturn(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return 0.0
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (float, float) {
	return 0
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() float {
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxMultipleValueReturn(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() (float, float) {
	return 0.0
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() float {
	return 0.0, 0.0
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (float, float, float) {
	return 0.0, 0.0
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (float, float) {
	return 0.0, 0.0
}

func Foo2() (float, float, float) {
	return Foo()
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (float, float, float) {
	return 0.0, 0.0, 0.0
}

func Foo2() (float, float, float) {
	return Foo()
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0.0)
}
`)); err != nil {
		t.Error(err)
	}
}

func TestSyntaxInit(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func init() {
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxUnsupportedSyntax(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := func() {
	}
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	go func() {
	}()
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	ch := make(chan int)
	_ = ch
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := 1i
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x [4]float
	y := x[1:2]
	_ = y
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x [4]float
	y := x[1:2:3]
	_ = y
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxForbidAssigningSpecialVariables(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

var U vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	U = vec4(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var U vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	U.x = 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var U [2]vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	U[0] = vec4(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	srcPos = vec2(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	srcPos.x = 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxBoolLiteral(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	true := vec4(0)
	return true
}
`)); err != nil {
		t.Error(err)
	}
}

func TestSyntaxUnusedVariable(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := 0
	x = 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := vec4(0)
	x.x = 1
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	// Increment statement treats a variable 'used'.
	// https://go.dev/play/p/2RuYMrSLjt3
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := 0
	x++
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var a int
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var a, b int
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	// Issue #2848
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var floats [4]float
	for i := 0; i < 3; i++ {
		j := i + 1
		floats[j] = float(i)
	}
	return vec4(floats[0], floats[1], floats[2], floats[3])
}
`)); err != nil {
		t.Error(err)
	}
}

func TestSyntaxBlankLhs(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x int = _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := 1
	x = _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	x := 1 + _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	_++
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	_ += 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	_.x = 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxDuplicatedVarsAndConstants(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var a = 0
	const a = 0
	_ = a
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	const a = 0
	var a = 0
	_ = a
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	const a = 0
	const a = 0
	_ = a
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

const U0 = 0
var U0 float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(a)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var U0 float
const U0 = 0

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(a)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxUnmatchedArgs(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	Foo(1)
	return dstPos
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo(x float) {
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	Foo()
	return dstPos
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo(x, y float) {
}

func Bar() (float, float, float) {
	return 0, 1
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	Foo(Bar())
	return dstPos
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo(x, y, z float) {
}

func Bar() (float, float) {
	return 0, 1
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	Foo(Bar())
	return dstPos
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #1898
func TestSyntaxMeaninglessSentence(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

var Time float
var ScreenSize vec2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	dstPos
	return dstPos
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var Time float
var ScreenSize vec2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	vec2(dstPos)
	return dstPos
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #1947
func TestSyntaxOperatorMod(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2.0 % 0.5
	_ = a
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// If both are constants, both must be an integer!
	a := 2.0 % 1.0
	_ = a
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := int(2) % 0.5
	_ = a
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := int(2) % 1.0
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2.0
	b := 0.5
	_ = a % b
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2
	b := 0.5
	_ = a % b
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2.5
	b := 1
	_ = a % b
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2
	b := 1
	_ = a % b
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2
	_ = a % 1
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// If only one of two is a consntant, the constant can be a float.
	a := 2
	_ = a % 1.0
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1
	_ = 2 % a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// If only one of two is a consntant, the constant can be a float.
	a := 1
	_ = 2.0 % a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2
	a %= 1
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2
	a %= 1.0
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2
	a %= 0.5
	_ = a
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 2.0
	a %= 1
	_ = a
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxOperatorAssign(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1.0
	a += 2
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1.0
	a += 2.0
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1.0
	a += 2.1
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1
	a += 2
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1
	a += 2.0
	_ = a
	return vec4(0)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1
	a += 2.1
	_ = a
	return vec4(0)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x float = true
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x bool = true
	_ = x
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x int = 1.0
	_ = x
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}
}

// Issue #1963
func TestSyntaxOperatorVecAndNumber(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1) + 2
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1) + 2
	return vec4(a.xxyy)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1) + 2.1
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1) + 2.1
	return vec4(a.xxyy)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1) % 2
	return a.xxyy
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1) % 2
	return vec4(a.xxyy)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1) % 2.1
	return a.xxyy
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1) % 2.1
	return vec4(a.xxyy)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1)
	a += 2
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1)
	a += 2
	return vec4(a.xxyy)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1)
	a += 2.1
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1)
	a += 2.1
	return vec4(a.xxyy)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1)
	a %= 2
	return a.xxyy
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1)
	a %= 2
	return vec4(a.xxyy)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec2(1)
	a %= 2.1
	return a.xxyy
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := ivec2(1)
	a %= 2.1
	return vec4(a.xxyy)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #1971
func TestSyntaxOperatorMultiply(t *testing.T) {
	// Note: mat + float is allowed in GLSL but not in Metal.

	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := 1 * vec2(2); _ = a", err: false},
		{stmt: "a := int(1) * vec2(2); _ = a", err: true},
		{stmt: "a := 1.0 * vec2(2); _ = a", err: false},
		{stmt: "a := 1.1 * vec2(2); _ = a", err: false},
		{stmt: "a := 1 + vec2(2); _ = a", err: false},
		{stmt: "a := int(1) + vec2(2); _ = a", err: true},
		{stmt: "a := 1.0 / vec2(2); _ = a", err: false},
		{stmt: "a := 1.1 / vec2(2); _ = a", err: false},
		{stmt: "a := 1.0 + vec2(2); _ = a", err: false},
		{stmt: "a := 1.1 + vec2(2); _ = a", err: false},
		{stmt: "a := 1 * vec3(2); _ = a", err: false},
		{stmt: "a := 1.0 * vec3(2); _ = a", err: false},
		{stmt: "a := 1.1 * vec3(2); _ = a", err: false},
		{stmt: "a := 1 * vec4(2); _ = a", err: false},
		{stmt: "a := 1.0 * vec4(2); _ = a", err: false},
		{stmt: "a := 1.1 * vec4(2); _ = a", err: false},

		{stmt: "a := 1 * ivec2(2); _ = a", err: false},
		{stmt: "a := int(1) * ivec2(2); _ = a", err: false},
		{stmt: "a := 1.0 * ivec2(2); _ = a", err: false},
		{stmt: "a := 1.1 * ivec2(2); _ = a", err: true},
		{stmt: "a := 1 + ivec2(2); _ = a", err: false},
		{stmt: "a := int(1) + ivec2(2); _ = a", err: false},
		{stmt: "a := 1.0 / ivec2(2); _ = a", err: false},
		{stmt: "a := 1.1 / ivec2(2); _ = a", err: true},
		{stmt: "a := 1.0 + ivec2(2); _ = a", err: false},
		{stmt: "a := 1.1 + ivec2(2); _ = a", err: true},
		{stmt: "a := 1 * ivec3(2); _ = a", err: false},
		{stmt: "a := 1.0 * ivec3(2); _ = a", err: false},
		{stmt: "a := 1.1 * ivec3(2); _ = a", err: true},
		{stmt: "a := 1 * ivec4(2); _ = a", err: false},
		{stmt: "a := 1.0 * ivec4(2); _ = a", err: false},
		{stmt: "a := 1.1 * ivec4(2); _ = a", err: true},

		{stmt: "a := 1 * mat2(2); _ = a", err: false},
		{stmt: "a := 1.0 * mat2(2); _ = a", err: false},
		{stmt: "a := float(1.0) / mat2(2); _ = a", err: true},
		{stmt: "a := 1.0 / mat2(2); _ = a", err: true},
		{stmt: "a := float(1.0) + mat2(2); _ = a", err: true},
		{stmt: "a := 1.0 + mat2(2); _ = a", err: true},
		{stmt: "a := 1 * mat3(2); _ = a", err: false},
		{stmt: "a := 1.0 * mat3(2); _ = a", err: false},
		{stmt: "a := 1 * mat4(2); _ = a", err: false},
		{stmt: "a := 1.0 * mat4(2); _ = a", err: false},

		{stmt: "a := vec2(1) * 2; _ = a", err: false},
		{stmt: "a := vec2(1) * 2.0; _ = a", err: false},
		{stmt: "a := vec2(1) * 2.1; _ = a", err: false},
		{stmt: "a := vec2(1) / 2.0; _ = a", err: false},
		{stmt: "a := vec2(1) / 2.1; _ = a", err: false},
		{stmt: "a := vec2(1) + 2.0; _ = a", err: false},
		{stmt: "a := vec2(1) + 2.1; _ = a", err: false},
		{stmt: "a := vec2(1) * int(2); _ = a", err: true},
		{stmt: "a := vec2(1) * vec2(2); _ = a", err: false},
		{stmt: "a := vec2(1) + vec2(2); _ = a", err: false},
		{stmt: "a := vec2(1) * vec3(2); _ = a", err: true},
		{stmt: "a := vec2(1) * vec4(2); _ = a", err: true},
		{stmt: "a := vec2(1) * ivec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) + ivec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) * ivec3(2); _ = a", err: true},
		{stmt: "a := vec2(1) * ivec4(2); _ = a", err: true},
		{stmt: "a := vec2(1) * mat2(2); _ = a", err: false},
		{stmt: "a := vec2(1) + mat2(2); _ = a", err: true},
		{stmt: "a := vec2(1) * mat3(2); _ = a", err: true},
		{stmt: "a := vec2(1) * mat4(2); _ = a", err: true},

		{stmt: "a := ivec2(1) * 2; _ = a", err: false},
		{stmt: "a := ivec2(1) * 2.0; _ = a", err: false},
		{stmt: "a := ivec2(1) * 2.1; _ = a", err: true},
		{stmt: "a := ivec2(1) / 2.0; _ = a", err: false},
		{stmt: "a := ivec2(1) / 2.1; _ = a", err: true},
		{stmt: "a := ivec2(1) + 2.0; _ = a", err: false},
		{stmt: "a := ivec2(1) + 2.1; _ = a", err: true},
		{stmt: "a := ivec2(1) * int(2); _ = a", err: false},
		{stmt: "a := ivec2(1) * vec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) + vec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) * vec3(2); _ = a", err: true},
		{stmt: "a := ivec2(1) * vec4(2); _ = a", err: true},
		{stmt: "a := ivec2(1) * ivec2(2); _ = a", err: false},
		{stmt: "a := ivec2(1) + ivec2(2); _ = a", err: false},
		{stmt: "a := ivec2(1) * ivec3(2); _ = a", err: true},
		{stmt: "a := ivec2(1) * ivec4(2); _ = a", err: true},
		{stmt: "a := ivec2(1) * mat2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) + mat2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) * mat3(2); _ = a", err: true},
		{stmt: "a := ivec2(1) * mat4(2); _ = a", err: true},

		{stmt: "a := mat2(1) * 2; _ = a", err: false},
		{stmt: "a := mat2(1) * 2.0; _ = a", err: false},
		{stmt: "a := mat2(1) * 2.1; _ = a", err: false},
		{stmt: "a := mat2(1) / 2.0; _ = a", err: false},
		{stmt: "a := mat2(1) / 2.1; _ = a", err: false},
		{stmt: "a := mat2(1) / float(2); _ = a", err: false},
		{stmt: "a := mat2(1) * int(2); _ = a", err: true},
		{stmt: "a := mat2(1) + 2.0; _ = a", err: true},
		{stmt: "a := mat2(1) + 2.1; _ = a", err: true},
		{stmt: "a := mat2(1) + float(2); _ = a", err: true},
		{stmt: "a := mat2(1) * vec2(2); _ = a", err: false},
		{stmt: "a := mat2(1) + vec2(2); _ = a", err: true},
		{stmt: "a := mat2(1) * vec3(2); _ = a", err: true},
		{stmt: "a := mat2(1) * vec4(2); _ = a", err: true},
		{stmt: "a := mat2(1) * ivec2(2); _ = a", err: true},
		{stmt: "a := mat2(1) + ivec2(2); _ = a", err: true},
		{stmt: "a := mat2(1) * ivec3(2); _ = a", err: true},
		{stmt: "a := mat2(1) * ivec4(2); _ = a", err: true},
		{stmt: "a := mat2(1) * mat2(2); _ = a", err: false},
		{stmt: "a := mat2(1) / mat2(2); _ = a", err: true},
		{stmt: "a := mat2(1) * mat3(2); _ = a", err: true},
		{stmt: "a := mat2(1) * mat4(2); _ = a", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

// Issue: #2755
func TestSyntaxOperatorShift(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := 1 << 2; _ = a", err: false},
		{stmt: "a := 1 << 2.0; _ = a", err: false},
		{stmt: "a := 1.0 << 2; _ = a", err: false},
		{stmt: "a := 1.0 << 2.0; _ = a", err: false},
		{stmt: "a := 1.0 << int(1); _ = a", err: false},
		{stmt: "a := int(1) << 2.0; _ = a", err: false},
		{stmt: "a := ivec2(1) << 2.0; _ = a", err: false},
		{stmt: "var a = 1; b := a << 2.0; _ = b", err: false},
		{stmt: "var a = 1; b := 2.0 << a; _ = b", err: false}, // PR: #2916
		{stmt: "a := float(1.0) << 2; _ = a", err: true},
		{stmt: "a := 1 << float(2.0); _ = a", err: true},
		{stmt: "a := ivec2(1) << 2; _ = a", err: false},
		{stmt: "a := 1 << ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) << float(2.0); _ = a", err: true},
		{stmt: "a := float(1.0) << ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) << ivec2(2); _ = a", err: false},
		{stmt: "a := ivec3(1) << ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) << ivec3(2); _ = a", err: true},
		{stmt: "a := 1 << vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) << 2; _ = a", err: true},
		{stmt: "a := float(1.0) << vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) << float(2.0); _ = a", err: true},
		{stmt: "a := vec2(1) << vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) << vec3(2); _ = a", err: true},
		{stmt: "a := vec3(1) << vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) << ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) << vec2(2); _ = a", err: true},
		{stmt: "a := vec3(1) << ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) << vec3(2); _ = a", err: true},

		{stmt: "a := 1 >> 2; _ = a", err: false},
		{stmt: "a := 1 >> 2.0; _ = a", err: false},
		{stmt: "a := 1.0 >> 2; _ = a", err: false},
		{stmt: "a := 1.0 >> 2.0; _ = a", err: false},
		{stmt: "a := 1.0 >> int(1); _ = a", err: false},
		{stmt: "a := int(1) >> 2.0; _ = a", err: false},
		{stmt: "a := ivec2(1) >> 2.0; _ = a", err: false},
		{stmt: "var a = 1; b := a >> 2.0; _ = b", err: false},
		{stmt: "var a = 1; b := 2.0 >> a; _ = b", err: false}, // PR: #2916
		{stmt: "a := float(1.0) >> 2; _ = a", err: true},
		{stmt: "a := 1 >> float(2.0); _ = a", err: true},
		{stmt: "a := ivec2(1) >> 2; _ = a", err: false},
		{stmt: "a := 1 >> ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) >> float(2.0); _ = a", err: true},
		{stmt: "a := float(1.0) >> ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) >> ivec2(2); _ = a", err: false},
		{stmt: "a := ivec3(1) >> ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) >> ivec3(2); _ = a", err: true},
		{stmt: "a := 1 >> vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) >> 2; _ = a", err: true},
		{stmt: "a := float(1.0) >> vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) >> float(2.0); _ = a", err: true},
		{stmt: "a := vec2(1) >> vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) >> vec3(2); _ = a", err: true},
		{stmt: "a := vec3(1) >> vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1) >> ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) >> vec2(2); _ = a", err: true},
		{stmt: "a := vec3(1) >> ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1) >> vec3(2); _ = a", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

func TestSyntaxOperatorShiftAssign(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := 1; a <<= 2; _ = a", err: false},
		{stmt: "a := 1; a <<= 2.0; _ = a", err: false},
		{stmt: "a := float(1.0); a <<= 2; _ = a", err: true},
		{stmt: "a := 1; a <<= float(2.0); _ = a", err: true},
		{stmt: "a := ivec2(1); a <<= 2; _ = a", err: false},
		{stmt: "a := 1;  a <<= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a <<= float(2.0); _ = a", err: true},
		{stmt: "a := float(1.0); a <<= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a <<= ivec2(2); _ = a", err: false},
		{stmt: "a := ivec3(1); a <<= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a <<= ivec3(2); _ = a", err: true},
		{stmt: "a := 1; a <<= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a <<= 2; _ = a", err: true},
		{stmt: "a := float(1.0); a <<= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a <<= float(2.0); _ = a", err: true},
		{stmt: "a := vec2(1); a <<= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a <<= vec3(2); _ = a", err: true},
		{stmt: "a := vec3(1); a <<= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a <<= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a <<= vec2(2); _ = a", err: true},
		{stmt: "a := vec3(1); a <<= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a <<= vec3(2); _ = a", err: true},
		{stmt: "const c = 2; a := 1; a <<= c; _ = a", err: false},
		{stmt: "const c = 2.0; a := 1; a <<= c; _ = a", err: false},
		{stmt: "const c = 2; a := float(1.0); a <<= c; _ = a", err: true},
		{stmt: "const c float = 2; a := 1; a <<= c; _ = a", err: true},
		{stmt: "const c float = 2.0; a := 1; a <<= c; _ = a", err: true},
		{stmt: "const c int = 2; a := ivec2(1); a <<= c; _ = a", err: false},
		{stmt: "const c int = 2; a := vec2(1); a <<= c; _ = a", err: true},

		{stmt: "a := 1; a >>= 2; _ = a", err: false},
		{stmt: "a := 1; a >>= 2.0; _ = a", err: false},
		{stmt: "a := float(1.0); a >>= 2; _ = a", err: true},
		{stmt: "a := 1; a >>= float(2.0); _ = a", err: true},
		{stmt: "a := ivec2(1); a >>= 2; _ = a", err: false},
		{stmt: "a := 1;  a >>= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a >>= float(2.0); _ = a", err: true},
		{stmt: "a := float(1.0); a >>= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a >>= ivec2(2); _ = a", err: false},
		{stmt: "a := ivec3(1); a >>= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a >>= ivec3(2); _ = a", err: true},
		{stmt: "a := 1; a >>= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a >>= 2; _ = a", err: true},
		{stmt: "a := float(1.0); a >>= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a >>= float(2.0); _ = a", err: true},
		{stmt: "a := vec2(1); a >>= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a >>= vec3(2); _ = a", err: true},
		{stmt: "a := vec3(1); a >>= vec2(2); _ = a", err: true},
		{stmt: "a := vec2(1); a >>= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a >>= vec2(2); _ = a", err: true},
		{stmt: "a := vec3(1); a >>= ivec2(2); _ = a", err: true},
		{stmt: "a := ivec2(1); a >>= vec3(2); _ = a", err: true},
		{stmt: "const c = 2; a := 1; a >>= c; _ = a", err: false},
		{stmt: "const c = 2.0; a := 1; a >>= c; _ = a", err: false},
		{stmt: "const c = 2; a := float(1.0); a >>= c; _ = a", err: true},
		{stmt: "const c float = 2; a := 1; a >>= c; _ = a", err: true},
		{stmt: "const c float = 2.0; a := 1; a >>= c; _ = a", err: true},
		{stmt: "const c int = 2; a := ivec2(1); a >>= c; _ = a", err: false},
		{stmt: "const c int = 2; a := vec2(1); a >>= c; _ = a", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

// Issue #1971
func TestSyntaxOperatorMultiplyAssign(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := 1.0; a *= 2", err: false},
		{stmt: "a := 1.0; a *= 2.0", err: false},
		{stmt: "const c = 2; a := 1.0; a *= c", err: false},
		{stmt: "const c = 2.0; a := 1.0; a *= c", err: false},
		{stmt: "const c int = 2; a := 1.0; a *= c", err: true},
		{stmt: "const c int = 2.0; a := 1.0; a *= c", err: true},
		{stmt: "const c float = 2; a := 1.0; a *= c", err: false},
		{stmt: "const c float = 2.0; a := 1.0; a *= c", err: false},
		{stmt: "a := 1.0; a *= int(2)", err: true},
		{stmt: "a := 1.0; a *= vec2(2)", err: true},
		{stmt: "a := 1.0; a *= vec3(2)", err: true},
		{stmt: "a := 1.0; a *= vec4(2)", err: true},
		{stmt: "a := 1.0; a *= ivec2(2)", err: true},
		{stmt: "a := 1.0; a *= ivec3(2)", err: true},
		{stmt: "a := 1.0; a *= ivec4(2)", err: true},
		{stmt: "a := 1.0; a *= mat2(2)", err: true},
		{stmt: "a := 1.0; a *= mat3(2)", err: true},
		{stmt: "a := 1.0; a *= mat4(2)", err: true},

		{stmt: "a := vec2(1); a *= 2", err: false},
		{stmt: "a := vec2(1); a *= 2.0", err: false},
		{stmt: "const c = 2; a := vec2(1); a *= c", err: false},
		{stmt: "const c = 2.0; a := vec2(1); a *= c", err: false},
		{stmt: "const c int = 2; a := vec2(1); a *= c", err: true},
		{stmt: "const c int = 2.0; a := vec2(1); a *= c", err: true},
		{stmt: "const c float = 2; a := vec2(1); a *= c", err: false},
		{stmt: "const c float = 2.0; a := vec2(1); a *= c", err: false},
		{stmt: "a := vec2(1); a /= 2.0", err: false},
		{stmt: "a := vec2(1); a += 2.0", err: false},
		{stmt: "a := vec2(1); a *= int(2)", err: true},
		{stmt: "a := vec2(1); a *= float(2)", err: false},
		{stmt: "a := vec2(1); a /= float(2)", err: false},
		{stmt: "a := vec2(1); a *= vec2(2)", err: false},
		{stmt: "a := vec2(1); a += vec2(2)", err: false},
		{stmt: "a := vec2(1); a *= vec3(2)", err: true},
		{stmt: "a := vec2(1); a *= vec4(2)", err: true},
		{stmt: "a := vec2(1); a *= ivec2(2)", err: true},
		{stmt: "a := vec2(1); a += ivec2(2)", err: true},
		{stmt: "a := vec2(1); a *= ivec3(2)", err: true},
		{stmt: "a := vec2(1); a *= ivec4(2)", err: true},
		{stmt: "a := vec2(1); a *= mat2(2)", err: false},
		{stmt: "a := vec2(1); a += mat2(2)", err: true},
		{stmt: "a := vec2(1); a /= mat2(2)", err: true},
		{stmt: "a := vec2(1); a *= mat3(2)", err: true},
		{stmt: "a := vec2(1); a *= mat4(2)", err: true},

		{stmt: "a := ivec2(1); a *= 2", err: false},
		{stmt: "a := ivec2(1); a *= 2.0", err: false},
		{stmt: "const c = 2; a := ivec2(1); a *= c", err: false},
		{stmt: "const c = 2.0; a := ivec2(1); a *= c", err: false},
		{stmt: "const c int = 2; a := ivec2(1); a *= c", err: false},
		{stmt: "const c int = 2.0; a := ivec2(1); a *= c", err: false},
		{stmt: "const c float = 2; a := ivec2(1); a *= c", err: true},
		{stmt: "const c float = 2.0; a := ivec2(1); a *= c", err: true},
		{stmt: "a := ivec2(1); a /= 2.0", err: false},
		{stmt: "a := ivec2(1); a += 2.0", err: false},
		{stmt: "a := ivec2(1); a *= int(2)", err: false},
		{stmt: "a := ivec2(1); a *= float(2)", err: true},
		{stmt: "a := ivec2(1); a /= float(2)", err: true},
		{stmt: "a := ivec2(1); a *= vec2(2)", err: true},
		{stmt: "a := ivec2(1); a += vec2(2)", err: true},
		{stmt: "a := ivec2(1); a *= vec3(2)", err: true},
		{stmt: "a := ivec2(1); a *= vec4(2)", err: true},
		{stmt: "a := ivec2(1); a *= ivec2(2)", err: false},
		{stmt: "a := ivec2(1); a += ivec2(2)", err: false},
		{stmt: "a := ivec2(1); a *= ivec3(2)", err: true},
		{stmt: "a := ivec2(1); a *= ivec4(2)", err: true},
		{stmt: "a := ivec2(1); a *= mat2(2)", err: true},
		{stmt: "a := ivec2(1); a += mat2(2)", err: true},
		{stmt: "a := ivec2(1); a /= mat2(2)", err: true},
		{stmt: "a := ivec2(1); a *= mat3(2)", err: true},
		{stmt: "a := ivec2(1); a *= mat4(2)", err: true},

		{stmt: "a := mat2(1); a *= 2", err: false},
		{stmt: "a := mat2(1); a *= 2.0", err: false},
		{stmt: "const c = 2; a := mat2(1); a *= c", err: false},
		{stmt: "const c = 2.0; a := mat2(1); a *= c", err: false},
		{stmt: "const c int = 2; a := mat2(1); a *= c", err: true},
		{stmt: "const c int = 2.0; a := mat2(1); a *= c", err: true},
		{stmt: "const c float = 2; a := mat2(1); a *= c", err: false},
		{stmt: "const c float = 2.0; a := mat2(1); a *= c", err: false},
		{stmt: "a := mat2(1); a /= 2.0", err: false},
		{stmt: "a := mat2(1); a += 2.0", err: true},
		{stmt: "a := mat2(1); a *= int(2)", err: true},
		{stmt: "a := mat2(1); a *= float(2)", err: false},
		{stmt: "a := mat2(1); a /= float(2)", err: false},
		{stmt: "a := mat2(1); a *= vec2(2)", err: true},
		{stmt: "a := mat2(1); a += vec2(2)", err: true},
		{stmt: "a := mat2(1); a *= vec3(2)", err: true},
		{stmt: "a := mat2(1); a *= vec4(2)", err: true},
		{stmt: "a := mat2(1); a *= ivec2(2)", err: true},
		{stmt: "a := mat2(1); a += ivec2(2)", err: true},
		{stmt: "a := mat2(1); a *= ivec3(2)", err: true},
		{stmt: "a := mat2(1); a *= ivec4(2)", err: true},
		{stmt: "a := mat2(1); a *= mat2(2)", err: false},
		{stmt: "a := mat2(1); a += mat2(2)", err: false},
		{stmt: "a := mat2(1); a /= mat2(2)", err: true},
		{stmt: "a := mat2(1); a *= mat3(2)", err: true},
		{stmt: "a := mat2(1); a *= mat4(2)", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

// Issue #2754
func TestSyntaxBitwiseOperatorAssign(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := 1; a &= 2", err: false},
		{stmt: "a := 1; a &= 2.0", err: false},
		{stmt: "const c = 2; a := 1; a &= c", err: false},
		{stmt: "const c = 2.0; a := 1; a &= c", err: false},
		{stmt: "const c int = 2; a := 1; a &= c", err: false},
		{stmt: "const c int = 2.0; a := 1; a &= c", err: false},
		{stmt: "const c float = 2; a := 1; a &= c", err: true},
		{stmt: "const c float = 2.0; a := 1; a &= c", err: true},
		{stmt: "a := 1; a &= int(2)", err: false},
		{stmt: "a := 1; a &= vec2(2)", err: true},
		{stmt: "a := 1; a &= vec3(2)", err: true},
		{stmt: "a := 1; a &= vec4(2)", err: true},
		{stmt: "a := 1; a &= ivec2(2)", err: true},
		{stmt: "a := 1; a &= ivec3(2)", err: true},
		{stmt: "a := 1; a &= ivec4(2)", err: true},
		{stmt: "a := 1; a &= mat2(2)", err: true},
		{stmt: "a := 1; a &= mat3(2)", err: true},
		{stmt: "a := 1; a &= mat4(2)", err: true},
		{stmt: "a := 1.0; a &= 2", err: true},
		{stmt: "a := ivec2(1); a &= 2", err: false},
		{stmt: "a := ivec2(1); a &= ivec2(1)", err: false},
		{stmt: "a := ivec2(1); a &= ivec3(1)", err: true},
		{stmt: "a := ivec2(1); a &= ivec4(1)", err: true},
		{stmt: "a := vec2(1); a &= 2", err: true},
		{stmt: "a := vec2(1); a &= vec2(2)", err: true},
		{stmt: "a := mat2(1); a &= 2", err: true},
		{stmt: "a := mat2(1); a &= mat2(2)", err: true},

		{stmt: "a := 1; a |= 2", err: false},
		{stmt: "a := 1; a |= 2.0", err: false},
		{stmt: "const c = 2; a := 1; a |= c", err: false},
		{stmt: "const c = 2.0; a := 1; a |= c", err: false},
		{stmt: "const c int = 2; a := 1; a |= c", err: false},
		{stmt: "const c int = 2.0; a := 1; a |= c", err: false},
		{stmt: "const c float = 2; a := 1; a |= c", err: true},
		{stmt: "const c float = 2.0; a := 1; a |= c", err: true},
		{stmt: "a := 1; a |= int(2)", err: false},
		{stmt: "a := 1; a |= vec2(2)", err: true},
		{stmt: "a := 1; a |= vec3(2)", err: true},
		{stmt: "a := 1; a |= vec4(2)", err: true},
		{stmt: "a := 1; a |= ivec2(2)", err: true},
		{stmt: "a := 1; a |= ivec3(2)", err: true},
		{stmt: "a := 1; a |= ivec4(2)", err: true},
		{stmt: "a := 1; a |= mat2(2)", err: true},
		{stmt: "a := 1; a |= mat3(2)", err: true},
		{stmt: "a := 1; a |= mat4(2)", err: true},
		{stmt: "a := 1.0; a |= 2", err: true},
		{stmt: "a := ivec2(1); a |= 2", err: false},
		{stmt: "a := ivec2(1); a |= ivec2(1)", err: false},
		{stmt: "a := ivec2(1); a |= ivec3(1)", err: true},
		{stmt: "a := ivec2(1); a |= ivec4(1)", err: true},
		{stmt: "a := vec2(1); a |= 2", err: true},
		{stmt: "a := vec2(1); a |= vec2(2)", err: true},
		{stmt: "a := mat2(1); a |= 2", err: true},
		{stmt: "a := mat2(1); a |= mat2(2)", err: true},

		{stmt: "a := 1; a ^= 2", err: false},
		{stmt: "a := 1; a ^= 2.0", err: false},
		{stmt: "const c = 2; a := 1; a ^= c", err: false},
		{stmt: "const c = 2.0; a := 1; a ^= c", err: false},
		{stmt: "const c int = 2; a := 1; a ^= c", err: false},
		{stmt: "const c int = 2.0; a := 1; a ^= c", err: false},
		{stmt: "const c float = 2; a := 1; a ^= c", err: true},
		{stmt: "const c float = 2.0; a := 1; a ^= c", err: true},
		{stmt: "a := 1; a ^= int(2)", err: false},
		{stmt: "a := 1; a ^= vec2(2)", err: true},
		{stmt: "a := 1; a ^= vec3(2)", err: true},
		{stmt: "a := 1; a ^= vec4(2)", err: true},
		{stmt: "a := 1; a ^= ivec2(2)", err: true},
		{stmt: "a := 1; a ^= ivec3(2)", err: true},
		{stmt: "a := 1; a ^= ivec4(2)", err: true},
		{stmt: "a := 1; a ^= mat2(2)", err: true},
		{stmt: "a := 1; a ^= mat3(2)", err: true},
		{stmt: "a := 1; a ^= mat4(2)", err: true},
		{stmt: "a := 1.0; a ^= 2", err: true},
		{stmt: "a := ivec2(1); a ^= 2", err: false},
		{stmt: "a := ivec2(1); a ^= ivec2(1)", err: false},
		{stmt: "a := ivec2(1); a ^= ivec3(1)", err: true},
		{stmt: "a := ivec2(1); a ^= ivec4(1)", err: true},
		{stmt: "a := vec2(1); a ^= 2", err: true},
		{stmt: "a := vec2(1); a ^= vec2(2)", err: true},
		{stmt: "a := mat2(1); a ^= 2", err: true},
		{stmt: "a := mat2(1); a ^= mat2(2)", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

func TestSyntaxAtan(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		// `atan` takes 1 argument.
		{stmt: "_ = atan(vec4(0))", err: false},
		{stmt: "_ = atan(vec4(0), vec4(0))", err: true},

		// `atan2` takes 2 arguments.
		{stmt: "_ = atan2(vec4(0))", err: true},
		{stmt: "_ = atan2(vec4(0), vec4(0))", err: false},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

// Issue #1972
func TestSyntaxType(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x vec2 = vec3(0)
	_ = x
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x, y vec2 = vec2(0), vec3(0)
	_, _ = x, y
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x vec2
	x = vec3(0)
	_ = x
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x, y vec2
	x, y = vec2(0), vec3(0)
	_ = x
	_ = y
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x vec2
	x = 0
	_ = x
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() (vec3, vec3) {
	return vec3(0), vec3(1)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x, y vec2 = Foo()
	_ = x
	_ = y
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() (vec3, vec3) {
	return vec3(0), vec3(1)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x, y vec2
	x, y = Foo()
	_ = x
	_ = y
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #1972
func TestSyntaxTypeBlankVar(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var _ vec2 = vec3(0)
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var _, _ vec2 = vec2(0), vec3(0)
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() (vec3, vec3) {
	return vec3(0), vec3(1)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var _, _ vec2 = Foo()
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #2032
func TestSyntaxTypeFuncCall(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo(x vec2) {
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	Foo(0)
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Foo(x vec2, y vec3) {
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	Foo(0, 1)
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Foo(x vec2, y vec3) {
}

func Bar() (int, int) {
	return 0, 1
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	Foo(Bar())
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	// Issue #2965
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	abs(sign)
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #2184
func TestSyntaxConstructorFuncType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := bool(false); _ = a", err: false},
		{stmt: "i := false; a := bool(i); _ = a", err: false},
		{stmt: "a := bool(1); _ = a", err: true},
		{stmt: "a := bool(1.0); _ = a", err: true},
		{stmt: "a := bool(); _ = a", err: true},
		{stmt: "a := bool(false, true); _ = a", err: true},

		{stmt: "a := int(1); _ = a", err: false},
		{stmt: "a := int(1.0); _ = a", err: false},
		{stmt: "i := 1; a := int(i); _ = a", err: false},
		{stmt: "i := 1.0; a := int(i); _ = a", err: false},
		{stmt: "i := 1.1; a := int(i); _ = a", err: false},
		{stmt: "a := int(1.1); _ = a", err: true},
		{stmt: "a := int(false); _ = a", err: true},
		{stmt: "a := int(); _ = a", err: true},
		{stmt: "a := int(1, 2); _ = a", err: true},

		{stmt: "a := float(1); _ = a", err: false},
		{stmt: "a := float(1.0); _ = a", err: false},
		{stmt: "a := float(1.1); _ = a", err: false},
		{stmt: "i := 1; a := float(i); _ = a", err: false},
		{stmt: "i := 1.0; a := float(i); _ = a", err: false},
		{stmt: "i := 1.1; a := float(i); _ = a", err: false},
		{stmt: "a := float(false); _ = a", err: true},
		{stmt: "a := float(); _ = a", err: true},
		{stmt: "a := float(1, 2); _ = a", err: true},

		{stmt: "a := vec2(1); _ = a", err: false},
		{stmt: "a := vec2(1.0); _ = a", err: false},
		{stmt: "i := 1; a := vec2(i); _ = a", err: true},
		{stmt: "i := 1.0; a := vec2(i); _ = a", err: false},
		{stmt: "a := vec2(vec2(1)); _ = a", err: false},
		{stmt: "a := vec2(vec3(1)); _ = a", err: true},
		{stmt: "a := vec2(ivec2(1)); _ = a", err: false},
		{stmt: "a := vec2(ivec3(1)); _ = a", err: true},

		{stmt: "a := vec2(1, 1); _ = a", err: false},
		{stmt: "a := vec2(1.0, 1.0); _ = a", err: false},
		{stmt: "a := vec2(1.1, 1.1); _ = a", err: false},
		{stmt: "i := 1; a := vec2(i, i); _ = a", err: true},
		{stmt: "i := 1.0; a := vec2(i, i); _ = a", err: false},
		{stmt: "a := vec2(vec2(1), 1); _ = a", err: true},
		{stmt: "a := vec2(1, vec2(1)); _ = a", err: true},
		{stmt: "a := vec2(vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := vec2(1, 1, 1); _ = a", err: true},

		{stmt: "a := vec3(1); _ = a", err: false},
		{stmt: "a := vec3(1.0); _ = a", err: false},
		{stmt: "a := vec3(1.1); _ = a", err: false},
		{stmt: "i := 1; a := vec3(i); _ = a", err: true},
		{stmt: "i := 1.0; a := vec3(i); _ = a", err: false},
		{stmt: "a := vec3(vec3(1)); _ = a", err: false},
		{stmt: "a := vec3(vec2(1)); _ = a", err: true},
		{stmt: "a := vec3(vec4(1)); _ = a", err: true},
		{stmt: "a := vec3(ivec3(1)); _ = a", err: false},
		{stmt: "a := vec3(ivec2(1)); _ = a", err: true},
		{stmt: "a := vec3(ivec4(1)); _ = a", err: true},

		{stmt: "a := vec3(1, 1, 1); _ = a", err: false},
		{stmt: "a := vec3(1.0, 1.0, 1.0); _ = a", err: false},
		{stmt: "a := vec3(1.1, 1.1, 1.1); _ = a", err: false},
		{stmt: "i := 1; a := vec3(i, i, i); _ = a", err: true},
		{stmt: "i := 1.0; a := vec3(i, i, i); _ = a", err: false},
		{stmt: "a := vec3(vec2(1), 1); _ = a", err: false},
		{stmt: "a := vec3(1, vec2(1)); _ = a", err: false},
		{stmt: "a := vec3(ivec2(1), 1); _ = a", err: true},
		{stmt: "a := vec3(1, ivec2(1)); _ = a", err: true},
		{stmt: "a := vec3(vec3(1), 1); _ = a", err: true},
		{stmt: "a := vec3(1, vec3(1)); _ = a", err: true},
		{stmt: "a := vec3(vec3(1), vec3(1), vec3(1)); _ = a", err: true},
		{stmt: "a := vec3(1, 1, 1, 1); _ = a", err: true},

		{stmt: "a := vec4(1); _ = a", err: false},
		{stmt: "a := vec4(1.0); _ = a", err: false},
		{stmt: "i := 1; a := vec4(i); _ = a", err: true},
		{stmt: "i := 1.0; a := vec4(i); _ = a", err: false},
		{stmt: "a := vec4(vec4(1)); _ = a", err: false},
		{stmt: "a := vec4(vec2(1)); _ = a", err: true},
		{stmt: "a := vec4(vec3(1)); _ = a", err: true},
		{stmt: "a := vec4(ivec4(1)); _ = a", err: false},
		{stmt: "a := vec4(ivec2(1)); _ = a", err: true},
		{stmt: "a := vec4(ivec3(1)); _ = a", err: true},

		{stmt: "a := vec4(1, 1, 1, 1); _ = a", err: false},
		{stmt: "a := vec4(1.0, 1.0, 1.0, 1.0); _ = a", err: false},
		{stmt: "a := vec4(1.1, 1.1, 1.1, 1.1); _ = a", err: false},
		{stmt: "i := 1; a := vec4(i, i, i, i); _ = a", err: true},
		{stmt: "i := 1.0; a := vec4(i, i, i, i); _ = a", err: false},
		{stmt: "a := vec4(vec2(1), 1, 1); _ = a", err: false},
		{stmt: "a := vec4(1, vec2(1), 1); _ = a", err: false},
		{stmt: "a := vec4(ivec2(1), 1, 1); _ = a", err: true},
		{stmt: "a := vec4(1, ivec2(1), 1); _ = a", err: true},
		{stmt: "a := vec4(1, 1, vec2(1)); _ = a", err: false},
		{stmt: "a := vec4(vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := vec4(ivec2(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := vec4(vec3(1), 1); _ = a", err: false},
		{stmt: "a := vec4(1, vec3(1)); _ = a", err: false},
		{stmt: "a := vec4(ivec3(1), 1); _ = a", err: true},
		{stmt: "a := vec4(1, ivec3(1)); _ = a", err: true},
		{stmt: "a := vec4(vec4(1), 1); _ = a", err: true},
		{stmt: "a := vec4(1, vec4(1)); _ = a", err: true},
		{stmt: "a := vec4(vec4(1), vec4(1), vec4(1), vec4(1)); _ = a", err: true},
		{stmt: "a := vec4(1, 1, 1, 1, 1); _ = a", err: true},

		{stmt: "a := ivec2(1); _ = a", err: false},
		{stmt: "a := ivec2(1.0); _ = a", err: false},
		{stmt: "i := 1; a := ivec2(i); _ = a", err: false},
		{stmt: "i := 1.0; a := ivec2(i); _ = a", err: true},
		{stmt: "a := ivec2(vec2(1)); _ = a", err: false},
		{stmt: "a := ivec2(vec3(1)); _ = a", err: true},
		{stmt: "a := ivec2(ivec2(1)); _ = a", err: false},
		{stmt: "a := ivec2(ivec3(1)); _ = a", err: true},

		{stmt: "a := ivec2(1, 1); _ = a", err: false},
		{stmt: "a := ivec2(1.0, 1.0); _ = a", err: false},
		{stmt: "i := 1; a := ivec2(i, i); _ = a", err: false},
		{stmt: "i := 1.0; a := ivec2(i, i); _ = a", err: true},
		{stmt: "a := ivec2(vec2(1), 1); _ = a", err: true},
		{stmt: "a := ivec2(1, vec2(1)); _ = a", err: true},
		{stmt: "a := ivec2(ivec2(1), 1); _ = a", err: true},
		{stmt: "a := ivec2(1, ivec2(1)); _ = a", err: true},
		{stmt: "a := ivec2(ivec2(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := ivec2(1, 1, 1); _ = a", err: true},

		{stmt: "a := ivec3(1); _ = a", err: false},
		{stmt: "a := ivec3(1.0); _ = a", err: false},
		{stmt: "a := ivec3(1.1); _ = a", err: true},
		{stmt: "i := 1; a := ivec3(i); _ = a", err: false},
		{stmt: "i := 1.0; a := ivec3(i); _ = a", err: true},
		{stmt: "a := ivec3(vec3(1)); _ = a", err: false},
		{stmt: "a := ivec3(vec2(1)); _ = a", err: true},
		{stmt: "a := ivec3(vec4(1)); _ = a", err: true},
		{stmt: "a := ivec3(ivec3(1)); _ = a", err: false},
		{stmt: "a := ivec3(ivec2(1)); _ = a", err: true},
		{stmt: "a := ivec3(ivec4(1)); _ = a", err: true},

		{stmt: "a := ivec3(1, 1, 1); _ = a", err: false},
		{stmt: "a := ivec3(1.0, 1.0, 1.0); _ = a", err: false},
		{stmt: "a := ivec3(1.1, 1.1, 1.1); _ = a", err: true},
		{stmt: "i := 1; a := ivec3(i, i, i); _ = a", err: false},
		{stmt: "i := 1.0; a := ivec3(i, i, i); _ = a", err: true},
		{stmt: "a := ivec3(vec2(1), 1); _ = a", err: true},
		{stmt: "a := ivec3(1, vec2(1)); _ = a", err: true},
		{stmt: "a := ivec3(ivec2(1), 1); _ = a", err: false},
		{stmt: "a := ivec3(1, ivec2(1)); _ = a", err: false},
		{stmt: "a := ivec3(vec3(1), 1); _ = a", err: true},
		{stmt: "a := ivec3(1, vec3(1)); _ = a", err: true},
		{stmt: "a := ivec3(vec3(1), vec3(1), vec3(1)); _ = a", err: true},
		{stmt: "a := ivec3(1, 1, 1, 1); _ = a", err: true},

		{stmt: "a := ivec4(1); _ = a", err: false},
		{stmt: "a := ivec4(1.0); _ = a", err: false},
		{stmt: "i := 1; a := ivec4(i); _ = a", err: false},
		{stmt: "i := 1.0; a := ivec4(i); _ = a", err: true},
		{stmt: "a := ivec4(vec4(1)); _ = a", err: false},
		{stmt: "a := ivec4(vec2(1)); _ = a", err: true},
		{stmt: "a := ivec4(vec3(1)); _ = a", err: true},
		{stmt: "a := ivec4(ivec4(1)); _ = a", err: false},
		{stmt: "a := ivec4(ivec2(1)); _ = a", err: true},
		{stmt: "a := ivec4(ivec3(1)); _ = a", err: true},

		{stmt: "a := ivec4(1, 1, 1, 1); _ = a", err: false},
		{stmt: "a := ivec4(1.0, 1.0, 1.0, 1.0); _ = a", err: false},
		{stmt: "a := ivec4(1.1, 1.1, 1.1, 1.1); _ = a", err: true},
		{stmt: "i := 1; a := ivec4(i, i, i, i); _ = a", err: false},
		{stmt: "i := 1.0; a := ivec4(i, i, i, i); _ = a", err: true},
		{stmt: "a := ivec4(vec2(1), 1, 1); _ = a", err: true},
		{stmt: "a := ivec4(1, vec2(1), 1); _ = a", err: true},
		{stmt: "a := ivec4(1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := ivec4(ivec2(1), 1, 1); _ = a", err: false},
		{stmt: "a := ivec4(1, ivec2(1), 1); _ = a", err: false},
		{stmt: "a := ivec4(1, 1, ivec2(1)); _ = a", err: false},
		{stmt: "a := ivec4(vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := ivec4(ivec2(1), ivec2(1)); _ = a", err: false},
		{stmt: "a := ivec4(vec3(1), 1); _ = a", err: true},
		{stmt: "a := ivec4(1, vec3(1)); _ = a", err: true},
		{stmt: "a := ivec4(ivec3(1), 1); _ = a", err: false},
		{stmt: "a := ivec4(1, ivec3(1)); _ = a", err: false},
		{stmt: "a := ivec4(vec4(1), 1); _ = a", err: true},
		{stmt: "a := ivec4(1, vec4(1)); _ = a", err: true},
		{stmt: "a := ivec4(vec4(1), vec4(1), vec4(1), vec4(1)); _ = a", err: true},
		{stmt: "a := ivec4(1, 1, 1, 1, 1); _ = a", err: true},

		{stmt: "a := mat2(1); _ = a", err: false},
		{stmt: "a := mat2(1.0); _ = a", err: false},
		{stmt: "i := 1; a := mat2(i); _ = a", err: true},
		{stmt: "i := 1.0; a := mat2(i); _ = a", err: false},
		{stmt: "a := mat2(mat2(1)); _ = a", err: false},
		{stmt: "a := mat2(vec2(1)); _ = a", err: true},
		{stmt: "a := mat2(ivec2(1)); _ = a", err: true},
		{stmt: "a := mat2(mat3(1)); _ = a", err: true},
		{stmt: "a := mat2(mat4(1)); _ = a", err: true},

		{stmt: "a := mat2(vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := mat2(ivec2(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := mat2(1, 1); _ = a", err: true},
		{stmt: "a := mat2(1, vec2(1)); _ = a", err: true},
		{stmt: "a := mat2(vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := mat2(mat2(1), mat2(1)); _ = a", err: true},

		{stmt: "a := mat2(1, 1, 1, 1); _ = a", err: false},
		{stmt: "a := mat2(1.0, 1.0, 1.0, 1.0); _ = a", err: false},
		{stmt: "i := 1; a := mat2(i, i, i, i); _ = a", err: true},
		{stmt: "i := 1.0; a := mat2(i, i, i, i); _ = a", err: false},
		{stmt: "a := mat2(vec2(1), vec2(1), vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := mat2(1, 1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := mat2(1, 1, 1, vec3(1)); _ = a", err: true},
		{stmt: "a := mat2(mat2(1), mat2(1), mat2(1), mat2(1)); _ = a", err: true},
		{stmt: "a := mat2(1, 1, 1); _ = a", err: true},
		{stmt: "a := mat2(1, 1, 1, 1, 1); _ = a", err: true},

		{stmt: "a := mat3(1); _ = a", err: false},
		{stmt: "a := mat3(1.0); _ = a", err: false},
		{stmt: "i := 1; a := mat3(i); _ = a", err: true},
		{stmt: "i := 1.0; a := mat3(i); _ = a", err: false},
		{stmt: "a := mat3(mat3(1)); _ = a", err: false},
		{stmt: "a := mat3(vec2(1)); _ = a", err: true},
		{stmt: "a := mat3(ivec2(1)); _ = a", err: true},
		{stmt: "a := mat3(mat2(1)); _ = a", err: true},
		{stmt: "a := mat3(mat4(1)); _ = a", err: true},

		{stmt: "a := mat3(vec3(1), vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := mat3(ivec3(1), ivec3(1), ivec3(1)); _ = a", err: true},
		{stmt: "a := mat3(1, 1, 1); _ = a", err: true},
		{stmt: "a := mat3(1, 1, vec3(1)); _ = a", err: true},
		{stmt: "a := mat3(vec3(1), vec3(1), vec4(1)); _ = a", err: true},
		{stmt: "a := mat3(mat3(1), mat3(1), mat3(1)); _ = a", err: true},

		{stmt: "a := mat3(1, 1, 1, 1, 1, 1, 1, 1, 1); _ = a", err: false},
		{stmt: "a := mat3(1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0); _ = a", err: false},
		{stmt: "i := 1; a := mat3(i, i, i, i, i, i, i, i, i); _ = a", err: true},
		{stmt: "i := 1.0; a := mat3(i, i, i, i, i, i, i, i, i); _ = a", err: false},
		{stmt: "a := mat3(vec3(1), vec3(1), vec3(1), vec3(1), vec3(1), vec3(1), vec3(1), vec3(1), vec3(1)); _ = a", err: true},
		{stmt: "a := mat3(1, 1, 1, 1, 1, 1, 1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := mat3(1, 1, 1, 1, 1, 1, 1, 1, vec3(1)); _ = a", err: true},
		{stmt: "a := mat3(mat3(1), mat3(1), mat3(1), mat3(1), mat3(1), mat3(1), mat3(1), mat3(1), mat3(1)); _ = a", err: true},
		{stmt: "a := mat3(1, 1, 1, 1, 1, 1, 1, 1); _ = a", err: true},
		{stmt: "a := mat3(1, 1, 1, 1, 1, 1, 1, 1, 1, 1); _ = a", err: true},

		{stmt: "a := mat4(1); _ = a", err: false},
		{stmt: "a := mat4(1.0); _ = a", err: false},
		{stmt: "i := 1; a := mat4(i); _ = a", err: true},
		{stmt: "i := 1.0; a := mat4(i); _ = a", err: false},
		{stmt: "a := mat4(mat4(1)); _ = a", err: false},
		{stmt: "a := mat4(vec2(1)); _ = a", err: true},
		{stmt: "a := mat4(ivec2(1)); _ = a", err: true},
		{stmt: "a := mat4(mat2(1)); _ = a", err: true},
		{stmt: "a := mat4(mat3(1)); _ = a", err: true},

		{stmt: "a := mat4(vec4(1), vec4(1), vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := mat4(ivec4(1), ivec4(1), ivec4(1), ivec4(1)); _ = a", err: true},
		{stmt: "a := mat4(1, 1, 1, 1); _ = a", err: true},
		{stmt: "a := mat4(1, 1, 1, vec4(1)); _ = a", err: true},
		{stmt: "a := mat4(vec4(1), vec4(1), vec4(1), vec2(1)); _ = a", err: true},
		{stmt: "a := mat4(mat4(1), mat4(1), mat4(1), mat4(1)); _ = a", err: true},

		{stmt: "a := mat4(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1); _ = a", err: false},
		{stmt: "a := mat4(1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0); _ = a", err: false},
		{stmt: "i := 1; a := mat4(i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i); _ = a", err: true},
		{stmt: "i := 1.0; a := mat4(i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i); _ = a", err: false},
		{stmt: "a := mat4(vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1), vec4(1)); _ = a", err: true},
		{stmt: "a := mat4(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := mat4(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, vec3(1)); _ = a", err: true},
		{stmt: "a := mat4(mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1), mat4(1)); _ = a", err: true},
		{stmt: "a := mat4(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1); _ = a", err: true},
		{stmt: "a := mat4(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

// Issue #2248
func TestSyntaxDiscard(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	if true {
		discard()
	}
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}
	// discard without return doesn't work so far.
	// TODO: Allow discard without return.
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	discard()
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func foo() {
	discard()
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	foo()
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncSingleArgType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := {{.Func}}(); _ = a", err: true},
		{stmt: "a := {{.Func}}(false); _ = a", err: true},
		{stmt: "a := {{.Func}}(1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(mat2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(mat3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(mat4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1); _ = a", err: true},
	}

	funcs := []string{
		"sin",
		"cos",
		"tan",
		"asin",
		"acos",
		"atan",
		"exp",
		"log",
		"exp2",
		"log2",
		"sqrt",
		"inversesqrt",
		"floor",
		"ceil",
		"fract",
		"length",
		"normalize",
		"dfdx",
		"dfdy",
		"fwidth",
	}
	for _, c := range cases {
		for _, f := range funcs {
			stmt := strings.ReplaceAll(c.stmt, "{{.Func}}", f)
			src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
			_, err := compileToIR([]byte(src))
			if err == nil && c.err {
				t.Errorf("%s must return an error but does not", stmt)
			} else if err != nil && !c.err {
				t.Errorf("%s must not return nil but returned %v", stmt, err)
			}
		}
	}
}

func TestSyntaxBuiltinFuncSingleArgTypeInteger(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := {{.Func}}(); _ = a", err: true},
		{stmt: "a := {{.Func}}(false); _ = a", err: true},
		{stmt: "a := {{.Func}}(1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(mat2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(mat3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(mat4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1); _ = a", err: true},
	}

	funcs := []string{
		"abs",
		"sign",
	}
	for _, c := range cases {
		for _, f := range funcs {
			stmt := strings.ReplaceAll(c.stmt, "{{.Func}}", f)
			src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
			_, err := compileToIR([]byte(src))
			if err == nil && c.err {
				t.Errorf("%s must return an error but does not", stmt)
			} else if err != nil && !c.err {
				t.Errorf("%s must not return nil but returned %v", stmt, err)
			}
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncDoubleArgsType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := {{.Func}}(); _ = a", err: true},
		{stmt: "a := {{.Func}}(1); _ = a", err: true},
		{stmt: "a := {{.Func}}(false, false); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(int(1), 1.0); _ = a", err: true},
		{stmt: "a := {{.Func}}(int(1), 1.1); _ = a", err: true},
		{stmt: "a := {{.Func}}(float(1), 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(float(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(float(1), 1.1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1.0, int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1.1, int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.1, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1), int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec3(1), vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(mat2(1), mat2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec2(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec2(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1, 1); _ = a", err: true},
	}

	funcs := []string{
		"atan2",
		"pow",
		"distance",
		"dot",
		"reflect",
	}
	for _, c := range cases {
		for _, f := range funcs {
			stmt := strings.ReplaceAll(c.stmt, "{{.Func}}", f)
			src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
			_, err := compileToIR([]byte(src))
			if err == nil && c.err {
				t.Errorf("%s must return an error but does not", stmt)
			} else if err != nil && !c.err {
				t.Errorf("%s must not return nil but returned %v", stmt, err)
			}
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncDoubleArgsType2(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := {{.Func}}(); _ = a", err: true},
		{stmt: "a := {{.Func}}(1); _ = a", err: true},
		{stmt: "a := {{.Func}}(false, false); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(int(1), 1.0); _ = a", err: true},
		{stmt: "a := {{.Func}}(int(1), 1.1); _ = a", err: true},
		{stmt: "a := {{.Func}}(float(1), 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(float(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(float(1), 1.1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1.0, int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1.1, int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.1, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1), int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, ivec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, ivec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, ivec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(vec3(1), vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec3(1), vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(vec4(1), vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(mat2(1), mat2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec2(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec2(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1, 1); _ = a", err: true},
	}

	funcs := []string{
		"mod",
	}
	for _, c := range cases {
		for _, f := range funcs {
			stmt := strings.ReplaceAll(c.stmt, "{{.Func}}", f)
			src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
			_, err := compileToIR([]byte(src))
			if err == nil && c.err {
				t.Errorf("%s must return an error but does not", stmt)
			} else if err != nil && !c.err {
				t.Errorf("%s must not return nil but returned %v", stmt, err)
			}
		}
	}
}

func TestSyntaxBuiltinFuncArgsMinMax(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := {{.Func}}(); _ = a", err: true},
		{stmt: "a := {{.Func}}(1); _ = a", err: false},
		{stmt: "a := {{.Func}}(false, false); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1); var _ int = a", err: false},
		{stmt: "a := {{.Func}}(1.0, 1); var _ float = a", err: false},
		{stmt: "a := {{.Func}}(1, 1.0); var _ float = a", err: false},
		{stmt: "a := {{.Func}}(1.1, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, 1.1); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1), 1); var _ int = a", err: false},
		{stmt: "a := {{.Func}}(int(1), 1.0); var _ int = a", err: false},
		{stmt: "a := {{.Func}}(int(1), 1.1); _ = a", err: true},
		{stmt: "a := {{.Func}}(float(1), 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(float(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(float(1), 1.1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, int(1)); var _ int = a", err: false},
		{stmt: "a := {{.Func}}(1.0, int(1)); var _ int = a", err: false},
		{stmt: "a := {{.Func}}(1.1, int(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.1, float(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(int(1), int(1)); var _ int = a", err: false},
		{stmt: "a := {{.Func}}(int(1), float(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(float(1), int(1)); _ = a", err: true},
		{stmt: "x := 1.1; a := {{.Func}}(int(x), 1); _ = a", err: false},
		{stmt: "x := 1; a := {{.Func}}(float(x), 1.1); _ = a", err: false},
		{stmt: "x := 1.1; a := {{.Func}}(1, int(x)); _ = a", err: false},
		{stmt: "x := 1; a := {{.Func}}(1.1, float(x)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, ivec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, ivec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, ivec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(vec2(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), 1.1); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(vec3(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec3(1), 1.1); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec3(1), vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec3(1), vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(vec4(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec4(1), 1.1); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec4(1), vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(mat2(1), mat2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec2(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(ivec2(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec2(1), 1.1); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec2(1), ivec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec2(1), ivec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec2(1), ivec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec3(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(ivec3(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec3(1), 1.1); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec3(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec3(1), ivec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec3(1), ivec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec4(1), 1); _ = a", err: false}, // The second argument can be a scalar.
		{stmt: "a := {{.Func}}(ivec4(1), 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec4(1), 1.1); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec4(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec4(1), ivec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(ivec4(1), ivec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, 1, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, 1, 1, 1, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0, 2.0, 3.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), vec2(2), vec2(3)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), 1, 1); _ = a", err: true},
	}

	funcs := []string{
		"min",
		"max",
	}
	for _, c := range cases {
		for _, f := range funcs {
			stmt := strings.ReplaceAll(c.stmt, "{{.Func}}", f)
			src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
			_, err := compileToIR([]byte(src))
			if err == nil && c.err {
				t.Errorf("%s must return an error but does not", stmt)
			} else if err != nil && !c.err {
				t.Errorf("%s must not return nil but returned %v", stmt, err)
			}
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncStepType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := step(); _ = a", err: true},
		{stmt: "a := step(1); _ = a", err: true},
		{stmt: "a := step(false, false); _ = a", err: true},
		{stmt: "a := step(1, 1); _ = a", err: false},
		{stmt: "a := step(1.0, 1); _ = a", err: false},
		{stmt: "a := step(1, 1.0); _ = a", err: false},
		{stmt: "a := step(int(1), int(1)); _ = a", err: true},
		{stmt: "a := step(1, vec2(1)); _ = a", err: false}, // The first argument can be a scalar.
		{stmt: "a := step(1, vec3(1)); _ = a", err: false}, // The first argument can be a scalar.
		{stmt: "a := step(1, vec4(1)); _ = a", err: false}, // The first argument can be a scalar.
		{stmt: "a := step(1, ivec2(1)); _ = a", err: true},
		{stmt: "a := step(1, ivec3(1)); _ = a", err: true},
		{stmt: "a := step(1, ivec4(1)); _ = a", err: true},
		{stmt: "a := step(vec2(1), 1); _ = a", err: true},
		{stmt: "a := step(vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := step(vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := step(vec2(1), vec4(1)); _ = a", err: true},
		{stmt: "a := step(vec3(1), 1); _ = a", err: true},
		{stmt: "a := step(vec3(1), vec2(1)); _ = a", err: true},
		{stmt: "a := step(vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := step(vec3(1), vec4(1)); _ = a", err: true},
		{stmt: "a := step(vec4(1), 1); _ = a", err: true},
		{stmt: "a := step(vec4(1), vec2(1)); _ = a", err: true},
		{stmt: "a := step(vec4(1), vec3(1)); _ = a", err: true},
		{stmt: "a := step(vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := step(mat2(1), mat2(1)); _ = a", err: true},
		{stmt: "a := step(ivec2(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := step(1, 1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncTripleArgsType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := {{.Func}}(); _ = a", err: true},
		{stmt: "a := {{.Func}}(1); _ = a", err: true},
		{stmt: "a := {{.Func}}(false, false); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(false, false, false); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1.0, 1, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, 1.0, 1); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, 1, 1.0); _ = a", err: false},
		{stmt: "a := {{.Func}}(1, vec2(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(1, vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), 1, 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), 1, vec2(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), vec2(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec2(1), vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec2(1), vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), 1, 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), 1, vec3(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), vec3(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec3(1), vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(vec4(1), 1, 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), 1, vec4(1)); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec4(1), 1); _ = a", err: true},
		{stmt: "a := {{.Func}}(vec4(1), vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := {{.Func}}(ivec2(1), ivec2(1), ivec2(1)); _ = a", err: true},
	}

	funcs := []string{
		"faceforward",
	}
	for _, c := range cases {
		for _, f := range funcs {
			stmt := strings.ReplaceAll(c.stmt, "{{.Func}}", f)
			src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
			_, err := compileToIR([]byte(src))
			if err == nil && c.err {
				t.Errorf("%s must return an error but does not", stmt)
			} else if err != nil && !c.err {
				t.Errorf("%s must not return nil but returned %v", stmt, err)
			}
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncClampType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := clamp(); _ = a", err: true},
		{stmt: "a := clamp(1); _ = a", err: true},
		{stmt: "a := clamp(false, false); _ = a", err: true},
		{stmt: "a := clamp(1, 1); _ = a", err: true},
		{stmt: "a := clamp(false, false, false); _ = a", err: true},
		{stmt: "a := clamp(1, 1, 1); var _ int = a", err: false},
		{stmt: "a := clamp(int(1), 1, 1); var _ int = a", err: false},
		{stmt: "a := clamp(int(1), 1.0, 1); var _ int = a", err: false},
		{stmt: "a := clamp(int(1), 1, 1.0); var _ int = a", err: false},
		{stmt: "a := clamp(int(1), 1.1, 1); _ = a", err: true},
		{stmt: "a := clamp(int(1), 1, 1.1); _ = a", err: true},
		{stmt: "a := clamp(float(1), 1, 1); var _ float = a", err: false},
		{stmt: "a := clamp(float(1), 1.0, 1); var _ float = a", err: false},
		{stmt: "a := clamp(float(1), 1, 1.0); var _ float = a", err: false},
		{stmt: "a := clamp(float(1), 1.1, 1); _ = a", err: false},
		{stmt: "a := clamp(float(1), 1, 1.1); _ = a", err: false},
		{stmt: "x := 1.1; a := clamp(int(x), 1, 1); _ = a", err: false},
		{stmt: "x := 1; a := clamp(float(x), 1.1, 1.1); _ = a", err: false},
		{stmt: "x := 1.1; a := clamp(1, int(x), 1); _ = a", err: false},
		{stmt: "x := 1; a := clamp(1.1, float(x), 1.1); _ = a", err: false},
		{stmt: "x := 1.1; a := clamp(1, 1, int(x)); _ = a", err: false},
		{stmt: "x := 1; a := clamp(1.1, 1.1, float(x)); _ = a", err: false},
		{stmt: "a := clamp(1.0, 1, 1); var _ float = a", err: false},
		{stmt: "a := clamp(1, 1.0, 1); var _ float = a", err: false},
		{stmt: "a := clamp(1, 1, 1.0); var _ float = a", err: false},
		{stmt: "a := clamp(1.1, 1, 1); var _ float = a", err: false},
		{stmt: "a := clamp(1, 1.1, 1); var _ float = a", err: false},
		{stmt: "a := clamp(1, 1, 1.1); var _ float = a", err: false},
		{stmt: "a := clamp(1, vec2(1), 1); _ = a", err: true},
		{stmt: "a := clamp(1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := clamp(1, vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := clamp(vec2(1), 1, 1); _ = a", err: false},
		{stmt: "a := clamp(vec2(1), 1, vec2(1)); _ = a", err: true},
		{stmt: "a := clamp(vec2(1), vec2(1), 1); _ = a", err: true},
		{stmt: "a := clamp(vec2(1), vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := clamp(vec2(1), vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := clamp(vec3(1), 1, 1); _ = a", err: false},
		{stmt: "a := clamp(vec3(1), 1, vec3(1)); _ = a", err: true},
		{stmt: "a := clamp(vec3(1), vec3(1), 1); _ = a", err: true},
		{stmt: "a := clamp(vec3(1), vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := clamp(vec4(1), 1, 1); _ = a", err: false},
		{stmt: "a := clamp(vec4(1), 1, vec4(1)); _ = a", err: true},
		{stmt: "a := clamp(vec4(1), vec4(1), 1); _ = a", err: true},
		{stmt: "a := clamp(vec4(1), vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := clamp(ivec2(1), 1, 1); _ = a", err: false},
		{stmt: "a := clamp(ivec2(1), 1.1, 1); _ = a", err: true},
		{stmt: "a := clamp(ivec2(1), 1, 1.1); _ = a", err: true},
		{stmt: "a := clamp(ivec2(1), 1, ivec2(1)); _ = a", err: true},
		{stmt: "a := clamp(ivec2(1), ivec2(1), 1); _ = a", err: true},
		{stmt: "a := clamp(ivec2(1), ivec2(1), ivec2(1)); _ = a", err: false},
		{stmt: "a := clamp(ivec2(1), ivec2(1), ivec3(1)); _ = a", err: true},
		{stmt: "a := clamp(ivec3(1), 1, 1); _ = a", err: false},
		{stmt: "a := clamp(ivec3(1), 1.1, 1); _ = a", err: true},
		{stmt: "a := clamp(ivec3(1), 1, 1.1); _ = a", err: true},
		{stmt: "a := clamp(ivec3(1), 1, ivec3(1)); _ = a", err: true},
		{stmt: "a := clamp(ivec3(1), ivec3(1), 1); _ = a", err: true},
		{stmt: "a := clamp(ivec3(1), ivec3(1), ivec3(1)); _ = a", err: false},
		{stmt: "a := clamp(ivec4(1), 1, 1); _ = a", err: false},
		{stmt: "a := clamp(ivec4(1), 1.1, 1); _ = a", err: true},
		{stmt: "a := clamp(ivec4(1), 1, 1.1); _ = a", err: true},
		{stmt: "a := clamp(ivec4(1), 1, ivec4(1)); _ = a", err: true},
		{stmt: "a := clamp(ivec4(1), ivec4(1), 1); _ = a", err: true},
		{stmt: "a := clamp(ivec4(1), ivec4(1), ivec4(1)); _ = a", err: false},
		{stmt: "a := clamp(1, 1, 1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncMixType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := mix(); _ = a", err: true},
		{stmt: "a := mix(1); _ = a", err: true},
		{stmt: "a := mix(false, false); _ = a", err: true},
		{stmt: "a := mix(1, 1); _ = a", err: true},
		{stmt: "a := mix(false, false, false); _ = a", err: true},
		{stmt: "a := mix(1, 1, 1); _ = a", err: false},
		{stmt: "a := mix(1.0, 1, 1); _ = a", err: false},
		{stmt: "a := mix(1, 1.0, 1); _ = a", err: false},
		{stmt: "a := mix(1, 1, 1.0); _ = a", err: false},
		{stmt: "a := mix(1, vec2(1), 1); _ = a", err: true},
		{stmt: "a := mix(1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := mix(1, vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := mix(vec2(1), 1, 1); _ = a", err: true},
		{stmt: "a := mix(vec2(1), 1, vec2(1)); _ = a", err: true},
		{stmt: "a := mix(vec2(1), vec2(1), 1); _ = a", err: false}, // The thrid argument can be a float.
		{stmt: "a := mix(vec2(1), vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := mix(vec2(1), vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := mix(vec3(1), 1, 1); _ = a", err: true},
		{stmt: "a := mix(vec3(1), 1, vec3(1)); _ = a", err: true},
		{stmt: "a := mix(vec3(1), vec3(1), 1); _ = a", err: false}, // The thrid argument can be a float.
		{stmt: "a := mix(vec3(1), vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := mix(vec4(1), 1, 1); _ = a", err: true},
		{stmt: "a := mix(vec4(1), 1, vec4(1)); _ = a", err: true},
		{stmt: "a := mix(vec4(1), vec4(1), 1); _ = a", err: false}, // The thrid argument can be a float.
		{stmt: "a := mix(vec4(1), vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := mix(ivec2(1), ivec2(1), 1); _ = a", err: true},
		{stmt: "a := mix(ivec2(1), ivec2(1), ivec2(1)); _ = a", err: true},
		{stmt: "a := mix(1, 1, 1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncSmoothstepType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := smoothstep(); _ = a", err: true},
		{stmt: "a := smoothstep(1); _ = a", err: true},
		{stmt: "a := smoothstep(false, false); _ = a", err: true},
		{stmt: "a := smoothstep(1, 1); _ = a", err: true},
		{stmt: "a := smoothstep(false, false, false); _ = a", err: true},
		{stmt: "a := smoothstep(1, 1, 1); _ = a", err: false},
		{stmt: "a := smoothstep(1.0, 1, 1); _ = a", err: false},
		{stmt: "a := smoothstep(1, 1.0, 1); _ = a", err: false},
		{stmt: "a := smoothstep(1, 1, 1.0); _ = a", err: false},
		{stmt: "a := smoothstep(1, vec2(1), 1); _ = a", err: true},
		{stmt: "a := smoothstep(1, 1, vec2(1)); _ = a", err: false},
		{stmt: "a := smoothstep(1, 1, vec3(1)); _ = a", err: false},
		{stmt: "a := smoothstep(1, 1, vec4(1)); _ = a", err: false},
		{stmt: "a := smoothstep(1, vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := smoothstep(vec2(1), 1, 1); _ = a", err: true},
		{stmt: "a := smoothstep(vec2(1), 1, vec2(1)); _ = a", err: true},
		{stmt: "a := smoothstep(vec2(1), vec2(1), 1); _ = a", err: true},
		{stmt: "a := smoothstep(vec2(1), vec2(1), vec2(1)); _ = a", err: false},
		{stmt: "a := smoothstep(vec2(1), vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := smoothstep(vec3(1), 1, 1); _ = a", err: true},
		{stmt: "a := smoothstep(vec3(1), 1, vec3(1)); _ = a", err: true},
		{stmt: "a := smoothstep(vec3(1), vec3(1), 1); _ = a", err: true},
		{stmt: "a := smoothstep(vec3(1), vec3(1), vec3(1)); _ = a", err: false},
		{stmt: "a := smoothstep(vec4(1), 1, 1); _ = a", err: true},
		{stmt: "a := smoothstep(vec4(1), 1, vec4(1)); _ = a", err: true},
		{stmt: "a := smoothstep(vec4(1), vec4(1), 1); _ = a", err: true},
		{stmt: "a := smoothstep(vec4(1), vec4(1), vec4(1)); _ = a", err: false},
		{stmt: "a := smoothstep(ivec2(1), 1, 1); _ = a", err: true},
		{stmt: "a := smoothstep(1, ivec2(1), 1); _ = a", err: true},
		{stmt: "a := smoothstep(1, 1, ivec2(1)); _ = a", err: true},
		{stmt: "a := smoothstep(1, 1, 1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncRefractType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := refract(); _ = a", err: true},
		{stmt: "a := refract(1); _ = a", err: true},
		{stmt: "a := refract(false, false); _ = a", err: true},
		{stmt: "a := refract(1, 1); _ = a", err: true},
		{stmt: "a := refract(false, false, false); _ = a", err: true},
		{stmt: "a := refract(1, 1, 1); _ = a", err: false},
		{stmt: "a := refract(1.0, 1, 1); _ = a", err: false},
		{stmt: "a := refract(1, 1.0, 1); _ = a", err: false},
		{stmt: "a := refract(1, 1, 1.0); _ = a", err: false},
		{stmt: "a := refract(1, vec2(1), 1); _ = a", err: true},
		{stmt: "a := refract(1, 1, vec2(1)); _ = a", err: true},
		{stmt: "a := refract(1, vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := refract(vec2(1), 1, 1); _ = a", err: true},
		{stmt: "a := refract(vec2(1), 1, vec2(1)); _ = a", err: true},
		{stmt: "a := refract(vec2(1), vec2(1), 1); _ = a", err: false}, // The third argument must be a float.
		{stmt: "a := refract(vec2(1), vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := refract(vec2(1), vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := refract(vec3(1), 1, 1); _ = a", err: true},
		{stmt: "a := refract(vec3(1), 1, vec3(1)); _ = a", err: true},
		{stmt: "a := refract(vec3(1), vec3(1), 1); _ = a", err: false}, // The third argument must be a float.
		{stmt: "a := refract(vec3(1), vec3(1), vec3(1)); _ = a", err: true},
		{stmt: "a := refract(vec4(1), 1, 1); _ = a", err: true},
		{stmt: "a := refract(vec4(1), 1, vec4(1)); _ = a", err: true},
		{stmt: "a := refract(vec4(1), vec4(1), 1); _ = a", err: false}, // The third argument must be a float.
		{stmt: "a := refract(vec4(1), vec4(1), vec4(1)); _ = a", err: true},
		{stmt: "a := refract(ivec2(1), ivec2(1), 1); _ = a", err: true},
		{stmt: "a := refract(1, 1, 1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncCrossType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := cross(); _ = a", err: true},
		{stmt: "a := cross(1); _ = a", err: true},
		{stmt: "a := cross(false, false); _ = a", err: true},
		{stmt: "a := cross(1, 1); _ = a", err: true},
		{stmt: "a := cross(1.0, 1); _ = a", err: true},
		{stmt: "a := cross(1, 1.0); _ = a", err: true},
		{stmt: "a := cross(int(1), int(1)); _ = a", err: true},
		{stmt: "a := cross(1, vec2(1)); _ = a", err: true},
		{stmt: "a := cross(1, vec3(1)); _ = a", err: true},
		{stmt: "a := cross(1, vec4(1)); _ = a", err: true},
		{stmt: "a := cross(vec2(1), 1); _ = a", err: true},
		{stmt: "a := cross(vec2(1), vec2(1)); _ = a", err: true},
		{stmt: "a := cross(vec2(1), vec3(1)); _ = a", err: true},
		{stmt: "a := cross(vec2(1), vec4(1)); _ = a", err: true},
		{stmt: "a := cross(vec3(1), 1); _ = a", err: true},
		{stmt: "a := cross(vec3(1), vec2(1)); _ = a", err: true},
		{stmt: "a := cross(vec3(1), vec3(1)); _ = a", err: false}, // Only two vec3s are allowed
		{stmt: "a := cross(vec3(1), vec4(1)); _ = a", err: true},
		{stmt: "a := cross(vec4(1), 1); _ = a", err: true},
		{stmt: "a := cross(vec4(1), vec2(1)); _ = a", err: true},
		{stmt: "a := cross(vec4(1), vec3(1)); _ = a", err: true},
		{stmt: "a := cross(vec4(1), vec4(1)); _ = a", err: true},
		{stmt: "a := cross(mat2(1), mat2(1)); _ = a", err: true},
		{stmt: "a := cross(ivec3(1), ivec3(1)); _ = a", err: true},
		{stmt: "a := cross(1, 1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2184
func TestSyntaxBuiltinFuncTransposeType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := transpose(); _ = a", err: true},
		{stmt: "a := transpose(false); _ = a", err: true},
		{stmt: "a := transpose(1); _ = a", err: true},
		{stmt: "a := transpose(1.0); _ = a", err: true},
		{stmt: "a := transpose(int(1)); _ = a", err: true},
		{stmt: "a := transpose(vec2(1)); _ = a", err: true},
		{stmt: "a := transpose(vec3(1)); _ = a", err: true},
		{stmt: "a := transpose(vec4(1)); _ = a", err: true},
		{stmt: "a := transpose(ivec2(1)); _ = a", err: true},
		{stmt: "a := transpose(ivec3(1)); _ = a", err: true},
		{stmt: "a := transpose(ivec4(1)); _ = a", err: true},
		{stmt: "a := transpose(mat2(1)); _ = a", err: false},
		{stmt: "a := transpose(mat3(1)); _ = a", err: false},
		{stmt: "a := transpose(mat4(1)); _ = a", err: false},
		{stmt: "a := transpose(1, 1); _ = a", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2187
func TestSyntaxEqual(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "_ = false == true", err: false},
		{stmt: "_ = false != true", err: false},
		{stmt: "_ = false == 1", err: true},
		{stmt: "_ = false != 1", err: true},
		{stmt: "_ = false == 1.0", err: true},
		{stmt: "_ = false != 1.0", err: true},
		{stmt: "_ = false == 1.1", err: true},
		{stmt: "_ = false != 1.1", err: true},
		{stmt: "a, b := false, true; _ = a == b", err: false},
		{stmt: "a, b := false, true; _ = a != b", err: false},
		{stmt: "a, b := false, 1; _ = a == b", err: true},
		{stmt: "a, b := false, 1; _ = a != b", err: true},
		{stmt: "a, b := false, 1.0; _ = a == b", err: true},
		{stmt: "a, b := false, 1.0; _ = a != b", err: true},
		{stmt: "a, b := false, 1.1; _ = a == b", err: true},
		{stmt: "a, b := false, 1.1; _ = a != b", err: true},
		{stmt: "a, b := false, vec2(1); _ = a == b", err: true},
		{stmt: "a, b := false, vec2(1); _ = a != b", err: true},
		{stmt: "a, b := false, ivec2(1); _ = a == b", err: true},
		{stmt: "a, b := false, ivec2(1); _ = a != b", err: true},
		{stmt: "a, b := false, mat2(1); _ = a == b", err: true},
		{stmt: "a, b := false, mat2(1); _ = a != b", err: true},

		{stmt: "_ = 1 == true", err: true},
		{stmt: "_ = 1 != true", err: true},
		{stmt: "_ = 1 == 1", err: false},
		{stmt: "_ = 1 != 1", err: false},
		{stmt: "_ = 1 == 1.0", err: false},
		{stmt: "_ = 1 != 1.0", err: false},
		{stmt: "_ = 1 == 1.1", err: false},
		{stmt: "_ = 1 != 1.1", err: false},
		{stmt: "a, b := 1, true; _ = a == b", err: true},
		{stmt: "a, b := 1, true; _ = a != b", err: true},
		{stmt: "a, b := 1, 1; _ = a == b", err: false},
		{stmt: "a, b := 1, 1; _ = a != b", err: false},
		{stmt: "a, b := 1, 1.0; _ = a == b", err: true},
		{stmt: "a, b := 1, 1.0; _ = a != b", err: true},
		{stmt: "a, b := 1, 1.1; _ = a == b", err: true},
		{stmt: "a, b := 1, 1.1; _ = a != b", err: true},
		{stmt: "a, b := 1, vec2(1); _ = a == b", err: true},
		{stmt: "a, b := 1, vec2(1); _ = a != b", err: true},
		{stmt: "a, b := 1, ivec2(1); _ = a == b", err: true},
		{stmt: "a, b := 1, ivec2(1); _ = a != b", err: true},
		{stmt: "a, b := 1, mat2(1); _ = a == b", err: true},
		{stmt: "a, b := 1, mat2(1); _ = a != b", err: true},

		{stmt: "_ = 1.0 == true", err: true},
		{stmt: "_ = 1.0 != true", err: true},
		{stmt: "_ = 1.0 == 1", err: false},
		{stmt: "_ = 1.0 != 1", err: false},
		{stmt: "_ = 1.0 == 1.0", err: false},
		{stmt: "_ = 1.0 != 1.0", err: false},
		{stmt: "_ = 1.0 == 1.1", err: false},
		{stmt: "_ = 1.0 != 1.1", err: false},
		{stmt: "a, b := 1.0, true; _ = a == b", err: true},
		{stmt: "a, b := 1.0, true; _ = a != b", err: true},
		{stmt: "a, b := 1.0, 1; _ = a == b", err: true},
		{stmt: "a, b := 1.0, 1; _ = a != b", err: true},
		{stmt: "a, b := 1.0, 1.0; _ = a == b", err: false},
		{stmt: "a, b := 1.0, 1.0; _ = a != b", err: false},
		{stmt: "a, b := 1.0, 1.1; _ = a == b", err: false},
		{stmt: "a, b := 1.0, 1.1; _ = a != b", err: false},
		{stmt: "a, b := 1.0, vec2(1); _ = a == b", err: true},
		{stmt: "a, b := 1.0, vec2(1); _ = a != b", err: true},
		{stmt: "a, b := 1.0, ivec2(1); _ = a == b", err: true},
		{stmt: "a, b := 1.0, ivec2(1); _ = a != b", err: true},
		{stmt: "a, b := 1.0, mat2(1); _ = a == b", err: true},
		{stmt: "a, b := 1.0, mat2(1); _ = a != b", err: true},

		{stmt: "_ = 1.1 == true", err: true},
		{stmt: "_ = 1.1 != true", err: true},
		{stmt: "_ = 1.1 == 1", err: false},
		{stmt: "_ = 1.1 != 1", err: false},
		{stmt: "_ = 1.1 == 1.0", err: false},
		{stmt: "_ = 1.1 != 1.0", err: false},
		{stmt: "_ = 1.1 == 1.1", err: false},
		{stmt: "_ = 1.1 != 1.1", err: false},
		{stmt: "a, b := 1.1, true; _ = a == b", err: true},
		{stmt: "a, b := 1.1, true; _ = a != b", err: true},
		{stmt: "a, b := 1.1, 1; _ = a == b", err: true},
		{stmt: "a, b := 1.1, 1; _ = a != b", err: true},
		{stmt: "a, b := 1.1, 1.0; _ = a == b", err: false},
		{stmt: "a, b := 1.1, 1.0; _ = a != b", err: false},
		{stmt: "a, b := 1.1, 1.1; _ = a == b", err: false},
		{stmt: "a, b := 1.1, 1.1; _ = a != b", err: false},
		{stmt: "a, b := 1.1, vec2(1); _ = a == b", err: true},
		{stmt: "a, b := 1.1, vec2(1); _ = a != b", err: true},
		{stmt: "a, b := 1.1, ivec2(1); _ = a == b", err: true},
		{stmt: "a, b := 1.1, ivec2(1); _ = a != b", err: true},
		{stmt: "a, b := 1.1, mat2(1); _ = a == b", err: true},
		{stmt: "a, b := 1.1, mat2(1); _ = a != b", err: true},

		{stmt: "_ = vec2(1) == true", err: true},
		{stmt: "_ = vec2(1) != true", err: true},
		{stmt: "_ = vec2(1) == 1", err: true},
		{stmt: "_ = vec2(1) != 1", err: true},
		{stmt: "_ = vec2(1) == 1.0", err: true},
		{stmt: "_ = vec2(1) != 1.0", err: true},
		{stmt: "_ = vec2(1) == 1.1", err: true},
		{stmt: "_ = vec2(1) != 1.1", err: true},
		{stmt: "a, b := vec2(1), true; _ = a == b", err: true},
		{stmt: "a, b := vec2(1), true; _ = a != b", err: true},
		{stmt: "a, b := vec2(1), 1; _ = a == b", err: true},
		{stmt: "a, b := vec2(1), 1; _ = a != b", err: true},
		{stmt: "a, b := vec2(1), 1.0; _ = a == b", err: true},
		{stmt: "a, b := vec2(1), 1.0; _ = a != b", err: true},
		{stmt: "a, b := vec2(1), 1.1; _ = a == b", err: true},
		{stmt: "a, b := vec2(1), 1.1; _ = a != b", err: true},
		{stmt: "a, b := vec2(1), vec2(1); _ = a == b", err: false},
		{stmt: "a, b := vec2(1), vec2(1); _ = a != b", err: false},
		{stmt: "a, b := vec2(1), ivec2(1); _ = a == b", err: true},
		{stmt: "a, b := vec2(1), ivec2(1); _ = a != b", err: true},
		{stmt: "a, b := vec2(1), mat2(1); _ = a == b", err: true},
		{stmt: "a, b := vec2(1), mat2(1); _ = a != b", err: true},

		{stmt: "_ = ivec2(1) == true", err: true},
		{stmt: "_ = ivec2(1) != true", err: true},
		{stmt: "_ = ivec2(1) == 1", err: true},
		{stmt: "_ = ivec2(1) != 1", err: true},
		{stmt: "_ = ivec2(1) == 1.0", err: true},
		{stmt: "_ = ivec2(1) != 1.0", err: true},
		{stmt: "_ = ivec2(1) == 1.1", err: true},
		{stmt: "_ = ivec2(1) != 1.1", err: true},
		{stmt: "a, b := ivec2(1), true; _ = a == b", err: true},
		{stmt: "a, b := ivec2(1), true; _ = a != b", err: true},
		{stmt: "a, b := ivec2(1), 1; _ = a == b", err: true},
		{stmt: "a, b := ivec2(1), 1; _ = a != b", err: true},
		{stmt: "a, b := ivec2(1), 1.0; _ = a == b", err: true},
		{stmt: "a, b := ivec2(1), 1.0; _ = a != b", err: true},
		{stmt: "a, b := ivec2(1), 1.1; _ = a == b", err: true},
		{stmt: "a, b := ivec2(1), 1.1; _ = a != b", err: true},
		{stmt: "a, b := ivec2(1), vec2(1); _ = a == b", err: true},
		{stmt: "a, b := ivec2(1), vec2(1); _ = a != b", err: true},
		{stmt: "a, b := ivec2(1), ivec2(1); _ = a == b", err: false},
		{stmt: "a, b := ivec2(1), ivec2(1); _ = a != b", err: false},
		{stmt: "a, b := ivec2(1), mat2(1); _ = a == b", err: true},
		{stmt: "a, b := ivec2(1), mat2(1); _ = a != b", err: true},

		{stmt: "_ = mat2(1) == true", err: true},
		{stmt: "_ = mat2(1) != true", err: true},
		{stmt: "_ = mat2(1) == 1", err: true},
		{stmt: "_ = mat2(1) != 1", err: true},
		{stmt: "_ = mat2(1) == 1.0", err: true},
		{stmt: "_ = mat2(1) != 1.0", err: true},
		{stmt: "_ = mat2(1) == 1.1", err: true},
		{stmt: "_ = mat2(1) != 1.1", err: true},
		{stmt: "a, b := mat2(1), true; _ = a == b", err: true},
		{stmt: "a, b := mat2(1), true; _ = a != b", err: true},
		{stmt: "a, b := mat2(1), 1; _ = a == b", err: true},
		{stmt: "a, b := mat2(1), 1; _ = a != b", err: true},
		{stmt: "a, b := mat2(1), 1.0; _ = a == b", err: true},
		{stmt: "a, b := mat2(1), 1.0; _ = a != b", err: true},
		{stmt: "a, b := mat2(1), 1.1; _ = a == b", err: true},
		{stmt: "a, b := mat2(1), 1.1; _ = a != b", err: true},
		{stmt: "a, b := mat2(1), vec2(1); _ = a == b", err: true},
		{stmt: "a, b := mat2(1), vec2(1); _ = a != b", err: true},
		{stmt: "a, b := mat2(1), ivec2(1); _ = a == b", err: true},
		{stmt: "a, b := mat2(1), ivec2(1); _ = a != b", err: true},
		{stmt: "a, b := mat2(1), mat2(1); _ = a == b", err: true}, // Comparing matrices are not allowed.
		{stmt: "a, b := mat2(1), mat2(1); _ = a != b", err: true}, // Comparing matrices are not allowed.

		{stmt: "_ = false && true", err: false},
		{stmt: "_ = false || true", err: false},
		{stmt: "_ = false && 1", err: true},
		{stmt: "_ = false || 1", err: true},
		{stmt: "_ = false && 1.0", err: true},
		{stmt: "_ = false || 1.0", err: true},
		{stmt: "_ = false && 1.1", err: true},
		{stmt: "_ = false || 1.1", err: true},
		{stmt: "a, b := false, true; _ = a && b", err: false},
		{stmt: "a, b := false, true; _ = a || b", err: false},
		{stmt: "a, b := false, 1; _ = a && b", err: true},
		{stmt: "a, b := false, 1; _ = a || b", err: true},
		{stmt: "a, b := false, 1.0; _ = a && b", err: true},
		{stmt: "a, b := false, 1.0; _ = a || b", err: true},
		{stmt: "a, b := false, 1.1; _ = a && b", err: true},
		{stmt: "a, b := false, 1.1; _ = a || b", err: true},
		{stmt: "a, b := false, vec2(1); _ = a && b", err: true},
		{stmt: "a, b := false, vec2(1); _ = a || b", err: true},
		{stmt: "a, b := false, ivec2(1); _ = a && b", err: true},
		{stmt: "a, b := false, ivec2(1); _ = a || b", err: true},
		{stmt: "a, b := false, mat2(1); _ = a && b", err: true},
		{stmt: "a, b := false, mat2(1); _ = a || b", err: true},

		{stmt: "_ = 1.0 && true", err: true},
		{stmt: "_ = 1.0 || true", err: true},
		{stmt: "_ = 1.0 && 1", err: true},
		{stmt: "_ = 1.0 || 1", err: true},
		{stmt: "_ = 1.0 && 1.0", err: true},
		{stmt: "_ = 1.0 || 1.0", err: true},
		{stmt: "_ = 1.0 && 1.1", err: true},
		{stmt: "_ = 1.0 || 1.1", err: true},
		{stmt: "a, b := 1.0, true; _ = a && b", err: true},
		{stmt: "a, b := 1.0, true; _ = a || b", err: true},
		{stmt: "a, b := 1.0, 1; _ = a && b", err: true},
		{stmt: "a, b := 1.0, 1; _ = a || b", err: true},
		{stmt: "a, b := 1.0, 1.0; _ = a && b", err: true},
		{stmt: "a, b := 1.0, 1.0; _ = a || b", err: true},
		{stmt: "a, b := 1.0, 1.1; _ = a && b", err: true},
		{stmt: "a, b := 1.0, 1.1; _ = a || b", err: true},
		{stmt: "a, b := 1.0, vec2(1); _ = a && b", err: true},
		{stmt: "a, b := 1.0, vec2(1); _ = a || b", err: true},
		{stmt: "a, b := 1.0, ivec2(1); _ = a && b", err: true},
		{stmt: "a, b := 1.0, ivec2(1); _ = a || b", err: true},
		{stmt: "a, b := 1.0, mat2(1); _ = a && b", err: true},
		{stmt: "a, b := 1.0, mat2(1); _ = a || b", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

func TestSyntaxTypeRedeclaration(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "type Foo int; type Foo int", err: true},
		{stmt: "type Foo int; type Foo float", err: true},
		{stmt: "type Foo int; { type Foo int }", err: false},
		{stmt: "type Foo int; type Bar int", err: false},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

func TestSyntaxSwizzling(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "var a vec2; var b float = a.x; _ = b", err: false},
		{stmt: "var a vec2; var b float = a.y; _ = b", err: false},
		{stmt: "var a vec2; var b float = a.z; _ = b", err: true},
		{stmt: "var a vec2; var b float = a.w; _ = b", err: true},
		{stmt: "var a vec2; var b vec2 = a.xy; _ = b", err: false},
		{stmt: "var a vec2; var b vec3 = a.xyz; _ = b", err: true},
		{stmt: "var a vec2; var b vec3 = a.xyw; _ = b", err: true},
		{stmt: "var a vec2; var b vec3 = a.xyy; _ = b", err: false},
		{stmt: "var a vec2; var b vec3 = a.zzz; _ = b", err: true},
		{stmt: "var a vec2; var b vec4 = a.xyzw; _ = b", err: true},

		{stmt: "var a vec2; var b float = a.r; _ = b", err: false},
		{stmt: "var a vec2; var b float = a.g; _ = b", err: false},
		{stmt: "var a vec2; var b float = a.b; _ = b", err: true},
		{stmt: "var a vec2; var b float = a.a; _ = b", err: true},
		{stmt: "var a vec2; var b vec2 = a.rg; _ = b", err: false},
		{stmt: "var a vec2; var b vec3 = a.rgb; _ = b", err: true},
		{stmt: "var a vec2; var b vec3 = a.rga; _ = b", err: true},
		{stmt: "var a vec2; var b vec3 = a.rgg; _ = b", err: false},
		{stmt: "var a vec2; var b vec3 = a.bbb; _ = b", err: true},
		{stmt: "var a vec2; var b vec4 = a.rgba; _ = b", err: true},

		{stmt: "var a vec2; var b float = a.s; _ = b", err: false},
		{stmt: "var a vec2; var b float = a.t; _ = b", err: false},
		{stmt: "var a vec2; var b float = a.p; _ = b", err: true},
		{stmt: "var a vec2; var b float = a.q; _ = b", err: true},
		{stmt: "var a vec2; var b vec2 = a.st; _ = b", err: false},
		{stmt: "var a vec2; var b vec3 = a.stp; _ = b", err: true},
		{stmt: "var a vec2; var b vec3 = a.stq; _ = b", err: true},
		{stmt: "var a vec2; var b vec3 = a.stt; _ = b", err: false},
		{stmt: "var a vec2; var b vec3 = a.ppp; _ = b", err: true},
		{stmt: "var a vec2; var b vec4 = a.stpq; _ = b", err: true},

		{stmt: "var a vec3; var b float = a.x; _ = b", err: false},
		{stmt: "var a vec3; var b float = a.y; _ = b", err: false},
		{stmt: "var a vec3; var b float = a.z; _ = b", err: false},
		{stmt: "var a vec3; var b float = a.w; _ = b", err: true},
		{stmt: "var a vec3; var b vec2 = a.xy; _ = b", err: false},
		{stmt: "var a vec3; var b vec3 = a.xyz; _ = b", err: false},
		{stmt: "var a vec3; var b vec3 = a.xyw; _ = b", err: true},
		{stmt: "var a vec3; var b vec3 = a.xyy; _ = b", err: false},
		{stmt: "var a vec3; var b vec3 = a.zzz; _ = b", err: false},
		{stmt: "var a vec3; var b vec4 = a.xyzw; _ = b", err: true},

		{stmt: "var a vec4; var b float = a.x; _ = b", err: false},
		{stmt: "var a vec4; var b float = a.y; _ = b", err: false},
		{stmt: "var a vec4; var b float = a.z; _ = b", err: false},
		{stmt: "var a vec4; var b float = a.w; _ = b", err: false},
		{stmt: "var a vec4; var b vec2 = a.xy; _ = b", err: false},
		{stmt: "var a vec4; var b vec3 = a.xyz; _ = b", err: false},
		{stmt: "var a vec4; var b vec3 = a.xyw; _ = b", err: false},
		{stmt: "var a vec4; var b vec3 = a.xyy; _ = b", err: false},
		{stmt: "var a vec4; var b vec3 = a.zzz; _ = b", err: false},
		{stmt: "var a vec4; var b vec4 = a.xyzw; _ = b", err: false},

		{stmt: "var a ivec2; var b int = a.x; _ = b", err: false},
		{stmt: "var a ivec2; var b int = a.y; _ = b", err: false},
		{stmt: "var a ivec2; var b int = a.z; _ = b", err: true},
		{stmt: "var a ivec2; var b int = a.w; _ = b", err: true},
		{stmt: "var a ivec2; var b ivec2 = a.xy; _ = b", err: false},
		{stmt: "var a ivec2; var b ivec3 = a.xyz; _ = b", err: true},
		{stmt: "var a ivec2; var b ivec3 = a.xyw; _ = b", err: true},
		{stmt: "var a ivec2; var b ivec3 = a.xyy; _ = b", err: false},
		{stmt: "var a ivec2; var b ivec3 = a.zzz; _ = b", err: true},
		{stmt: "var a ivec2; var b ivec4 = a.xyzw; _ = b", err: true},

		{stmt: "var a ivec3; var b int = a.x; _ = b", err: false},
		{stmt: "var a ivec3; var b int = a.y; _ = b", err: false},
		{stmt: "var a ivec3; var b int = a.z; _ = b", err: false},
		{stmt: "var a ivec3; var b int = a.w; _ = b", err: true},
		{stmt: "var a ivec3; var b ivec2 = a.xy; _ = b", err: false},
		{stmt: "var a ivec3; var b ivec3 = a.xyz; _ = b", err: false},
		{stmt: "var a ivec3; var b ivec3 = a.xyw; _ = b", err: true},
		{stmt: "var a ivec3; var b ivec3 = a.xyy; _ = b", err: false},
		{stmt: "var a ivec3; var b ivec3 = a.zzz; _ = b", err: false},
		{stmt: "var a ivec3; var b ivec4 = a.xyzw; _ = b", err: true},

		{stmt: "var a ivec4; var b int = a.x; _ = b", err: false},
		{stmt: "var a ivec4; var b int = a.y; _ = b", err: false},
		{stmt: "var a ivec4; var b int = a.z; _ = b", err: false},
		{stmt: "var a ivec4; var b int = a.w; _ = b", err: false},
		{stmt: "var a ivec4; var b ivec2 = a.xy; _ = b", err: false},
		{stmt: "var a ivec4; var b ivec3 = a.xyz; _ = b", err: false},
		{stmt: "var a ivec4; var b ivec3 = a.xyw; _ = b", err: false},
		{stmt: "var a ivec4; var b ivec3 = a.xyy; _ = b", err: false},
		{stmt: "var a ivec4; var b ivec3 = a.zzz; _ = b", err: false},
		{stmt: "var a ivec4; var b ivec4 = a.xyzw; _ = b", err: false},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

func TestSyntaxConstType(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "const a = false", err: false},
		{stmt: "const a bool = false", err: false},
		{stmt: "const a int = false", err: true},
		{stmt: "const a float = false", err: true},
		{stmt: "const a vec2 = false", err: true},
		{stmt: "const a ivec2 = false", err: true},

		{stmt: "const a = bool(false)", err: false},
		{stmt: "const a bool = bool(false)", err: false},
		{stmt: "const a int = bool(false)", err: true},
		{stmt: "const a float = bool(false)", err: true},
		{stmt: "const a vec2 = bool(false)", err: true},
		{stmt: "const a ivec2 = bool(false)", err: true},

		{stmt: "const a = int(false)", err: true},
		{stmt: "const a bool = int(false)", err: true},
		{stmt: "const a int = int(false)", err: true},
		{stmt: "const a float = int(false)", err: true},
		{stmt: "const a vec2 = int(false)", err: true},
		{stmt: "const a ivec2 = int(false)", err: true},

		{stmt: "const a = float(false)", err: true},
		{stmt: "const a bool = float(false)", err: true},
		{stmt: "const a int = float(false)", err: true},
		{stmt: "const a float = float(false)", err: true},
		{stmt: "const a vec2 = float(false)", err: true},
		{stmt: "const a ivec2 = float(false)", err: true},

		{stmt: "const a = 1", err: false},
		{stmt: "const a bool = 1", err: true},
		{stmt: "const a int = 1", err: false},
		{stmt: "const a float = 1", err: false},
		{stmt: "const a vec2 = 1", err: true},
		{stmt: "const a ivec2 = 1", err: true},

		{stmt: "const a = int(1)", err: false},
		{stmt: "const a bool = int(1)", err: true},
		{stmt: "const a int = int(1)", err: false},
		{stmt: "const a float = int(1)", err: true},
		{stmt: "const a vec2 = int(1)", err: true},
		{stmt: "const a ivec2 = int(1)", err: true},

		{stmt: "const a = float(1)", err: false},
		{stmt: "const a bool = float(1)", err: true},
		{stmt: "const a int = float(1)", err: true},
		{stmt: "const a float = float(1)", err: false},
		{stmt: "const a vec2 = float(1)", err: true},
		{stmt: "const a ivec2 = float(1)", err: true},

		{stmt: "const a = 1.0", err: false},
		{stmt: "const a bool = 1.0", err: true},
		{stmt: "const a int = 1.0", err: false},
		{stmt: "const a float = 1.0", err: false},
		{stmt: "const a vec2 = 1.0", err: true},
		{stmt: "const a ivec2 = 1.0", err: true},

		{stmt: "const a = int(1.0)", err: false},
		{stmt: "const a bool = int(1.0)", err: true},
		{stmt: "const a int = int(1.0)", err: false},
		{stmt: "const a float = int(1.0)", err: true},
		{stmt: "const a vec2 = int(1.0)", err: true},
		{stmt: "const a ivec2 = int(1.0)", err: true},

		{stmt: "const a = float(1.0)", err: false},
		{stmt: "const a bool = float(1.0)", err: true},
		{stmt: "const a int = float(1.0)", err: true},
		{stmt: "const a float = float(1.0)", err: false},
		{stmt: "const a vec2 = float(1.0)", err: true},
		{stmt: "const a ivec2 = float(1.0)", err: true},

		{stmt: "const a = 1.1", err: false},
		{stmt: "const a bool = 1.1", err: true},
		{stmt: "const a int = 1.1", err: true},
		{stmt: "const a float = 1.1", err: false},
		{stmt: "const a vec2 = 1.1", err: true},
		{stmt: "const a ivec2 = 1.1", err: true},

		{stmt: "const a = int(1.1)", err: true},
		{stmt: "const a bool = int(1.1)", err: true},
		{stmt: "const a int = int(1.1)", err: true},
		{stmt: "const a float = int(1.1)", err: true},
		{stmt: "const a vec2 = int(1.1)", err: true},
		{stmt: "const a ivec2 = int(1.1)", err: true},

		{stmt: "const a = float(1.1)", err: false},
		{stmt: "const a bool = float(1.1)", err: true},
		{stmt: "const a int = float(1.1)", err: true},
		{stmt: "const a float = float(1.1)", err: false},
		{stmt: "const a vec2 = float(1.1)", err: true},
		{stmt: "const a ivec2 = float(1.1)", err: true},

		{stmt: "const a = vec2(0)", err: true},
		{stmt: "const a bool = vec2(0)", err: true},
		{stmt: "const a int = vec2(0)", err: true},
		{stmt: "const a float = vec2(0)", err: true},
		{stmt: "const a vec2 = vec2(0)", err: true},
		{stmt: "const a ivec2 = vec2(0)", err: true},

		{stmt: "const a = ivec2(0)", err: true},
		{stmt: "const a bool = ivec2(0)", err: true},
		{stmt: "const a int = ivec2(0)", err: true},
		{stmt: "const a float = ivec2(0)", err: true},
		{stmt: "const a vec2 = ivec2(0)", err: true},
		{stmt: "const a ivec2 = ivec2(0)", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2549
func TestSyntaxConstType2(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "const x = 1; y := x*x; _ = vec4(1) / y", err: true},
		{stmt: "const x = 1.0; y := x*x; _ = vec4(1) / y", err: false},
		{stmt: "const x int = 1; y := x*x; _ = vec4(1) / y", err: true},
		{stmt: "const x int = 1.0; y := x*x; _ = vec4(1) / y", err: true},
		{stmt: "const x float = 1; y := x*x; _ = vec4(1) / y", err: false},
		{stmt: "const x float = 1.0; y := x*x; _ = vec4(1) / y", err: false},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2704
func TestSyntaxConstType3(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "const x = 1; const y = 1; _ = x * y", err: false},
		{stmt: "const x = 1; const y int = 1; _ = x * y", err: false},
		{stmt: "const x int = 1; const y = 1; _ = x * y", err: false},
		{stmt: "const x int = 1; const y int = 1; _ = x * y", err: false},
		{stmt: "const x = 1; const y float = 1; _ = x * y", err: false},
		{stmt: "const x float = 1; const y = 1; _ = x * y", err: false},
		{stmt: "const x float = 1; const y float = 1; _ = x * y", err: false},
		{stmt: "const x int = 1; const y float = 1; _ = x * y", err: true},
		{stmt: "const x float = 1; const y int = 1; _ = x * y", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2348
func TestSyntaxCompositeLit(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "_ = undefined{1, 2, 3, 4}", err: true},
		{stmt: "_ = int{1, 2, 3, 4}", err: true},
		{stmt: "_ = vec4{1, 2, 3, 4}", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

func TestSyntaxCompilerDirective(t *testing.T) {
	cases := []struct {
		src  string
		unit shaderir.Unit
		err  bool
	}{
		{
			src: `package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: shaderir.Texels,
			err:  false,
		},
		{
			src: `//kage:unit texels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: shaderir.Texels,
			err:  false,
		},
		{
			src: `//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: shaderir.Pixels,
			err:  false,
		},
		{
			src: `//kage:unit foo

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			err: true,
		},
		{
			src: `//kage:unit pixels
//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			err: true,
		},
		{
			src: `//kage:unit pixels
//kage:unit texels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			err: true,
		},
		{
			src: "\t    " + `//kage:unit pixels` + "    \t\r" + `
package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: shaderir.Pixels,
			err:  false,
		},
	}
	for _, c := range cases {
		ir, err := compileToIR([]byte(c.src))
		if err == nil && c.err {
			t.Errorf("Compile(%q) must return an error but does not", c.src)
		} else if err != nil && !c.err {
			t.Errorf("Compile(%q) must not return nil but returned %v", c.src, err)
		}
		if err != nil || c.err {
			continue
		}
		if got, want := ir.Unit, c.unit; got != want {
			t.Errorf("Compile(%q).Unit: got: %d, want: %d", c.src, got, want)
		}
	}
}

// Issue #2654
func TestSyntaxOmittedReturnType(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func foo(x vec2) {
	x = bar(x)
	_ = x
}

func bar(x vec2) {
	return x
}`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

// Issue #2590
func TestSyntaxAssignToUniformVariables(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

var Foo float

func foo(x vec2) {
	Foo = 0
}`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}

	if _, err := compileToIR([]byte(`package main

var Foo float

func foo(x vec2) {
	var x int
	x, Foo = 0, 0
	_ = x
}`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}

	if _, err := compileToIR([]byte(`package main

var Foo float

func foo(x vec2) {
	Foo += 0
}`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}

	// Issue #2711
	if _, err := compileToIR([]byte(`package main

var Foo float = 1
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

var Foo, Bar int = 1, 1
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

// Issue #2705
func TestSyntaxInitWithNegativeInteger(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x float = -0
	_ = x
	return dstPos
}`)); err != nil {
		t.Error(err)
	}
}

// Issue #2706
func TestSyntaxReturnConst(t *testing.T) {
	cases := []struct {
		typ  string
		stmt string
		err  bool
	}{
		{typ: "bool", stmt: "true", err: false},
		{typ: "int", stmt: "true", err: true},
		{typ: "float", stmt: "true", err: true},
		{typ: "bool", stmt: "1", err: true},
		{typ: "int", stmt: "1", err: false},
		{typ: "float", stmt: "1", err: false},
		{typ: "bool", stmt: "1.0", err: true},
		{typ: "int", stmt: "1.0", err: false},
		{typ: "float", stmt: "1.0", err: false},
		{typ: "bool", stmt: "1.1", err: true},
		{typ: "int", stmt: "1.1", err: true},
		{typ: "float", stmt: "1.1", err: false},
	}

	for _, c := range cases {
		typ := c.typ
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Foo() %s {
	return %s
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`, typ, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("return %s for type %s must return an error but does not", stmt, typ)
		} else if err != nil && !c.err {
			t.Errorf("return %s for type %s must not return nil but returned %v", stmt, typ, err)
		}
	}
}

// Issue #2706
func TestSyntaxScalarAndVector(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := vec2(1) + 1; var b vec2 = a; _ = b", err: false},
		{stmt: "a := 1 + vec2(1); var b vec2 = a; _ = b", err: false},
		{stmt: "a := vec2(1); b := 1; var c vec2 = a + b; _ = c", err: true},
		{stmt: "a := vec2(1); b := 1; var c vec2 = b + a; _ = c", err: true},
		{stmt: "a := vec2(1) + 1.0; var b vec2 = a; _ = b", err: false},
		{stmt: "a := 1.0 + vec2(1); var b vec2 = a; _ = b", err: false},
		{stmt: "a := vec2(1); b := 1.0; var c vec2 = a + b; _ = c", err: false},
		{stmt: "a := vec2(1); b := 1.0; var c vec2 = b + a; _ = c", err: false},
		{stmt: "a := vec2(1) + 1.1; var b vec2 = a; _ = b", err: false},
		{stmt: "a := 1.1 + vec2(1); var b vec2 = a; _ = b", err: false},
		{stmt: "a := vec2(1); b := 1.1; var c vec2 = a + b; _ = c", err: false},
		{stmt: "a := vec2(1); b := 1.1; var c vec2 = b + a; _ = c", err: false},

		{stmt: "a := ivec2(1) + 1; var b ivec2 = a; _ = b", err: false},
		{stmt: "a := 1 + ivec2(1); var b ivec2 = a; _ = b", err: false},
		{stmt: "a := ivec2(1); b := 1; var c ivec2 = a + b; _ = c", err: false},
		{stmt: "a := ivec2(1); b := 1; var c ivec2 = b + a; _ = c", err: false},
		{stmt: "a := ivec2(1) + 1.0; var b ivec2 = a; _ = b", err: false},
		{stmt: "a := 1.0 + ivec2(1); var b ivec2 = a; _ = b", err: false},
		{stmt: "a := ivec2(1); b := 1.0; var c ivec2 = a + b; _ = c", err: true},
		{stmt: "a := ivec2(1); b := 1.0; var c ivec2 = b + a; _ = c", err: true},
		{stmt: "a := ivec2(1) + 1.1; var b ivec2 = a; _ = b", err: true},
		{stmt: "a := 1.1 + ivec2(1); var b ivec2 = a; _ = b", err: true},
		{stmt: "a := ivec2(1); b := 1.1; var c ivec2 = a + b; _ = c", err: true},
		{stmt: "a := ivec2(1); b := 1.1; var c ivec2 = b + a; _ = c", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2712
func TestSyntaxCast(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := int(1); _ = a", err: false},
		{stmt: "a := int(1.0); _ = a", err: false},
		{stmt: "a := int(1.1); _ = a", err: true},
		{stmt: "const c = 1.1; a := int(c); _ = a", err: true},
		{stmt: "const c float = 1.1; a := int(c); _ = a", err: true},
		{stmt: "a := float(1); _ = a", err: false},
		{stmt: "a := float(1.0); _ = a", err: false},
		{stmt: "a := float(1.1); _ = a", err: false},
		{stmt: "a := 1; _ = int(a)", err: false},
		{stmt: "a := 1.0; _ = int(a)", err: false},
		{stmt: "a := 1.1; _ = int(a)", err: false},
		{stmt: "a := 1; _ = float(a)", err: false},
		{stmt: "a := 1.0; _ = float(a)", err: false},
		{stmt: "a := 1.1; _ = float(a)", err: false},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2718
func TestSyntaxCompare(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "_ = false == true", err: false},
		{stmt: "_ = int(0) == int(1)", err: false},
		{stmt: "_ = float(0) == float(1)", err: false},
		{stmt: "_ = vec2(0) == vec2(1)", err: false},
		{stmt: "_ = vec3(0) == vec3(1)", err: false},
		{stmt: "_ = vec4(0) == vec4(1)", err: false},
		{stmt: "_ = ivec2(0) == ivec2(1)", err: false},
		{stmt: "_ = ivec3(0) == ivec3(1)", err: false},
		{stmt: "_ = ivec4(0) == ivec4(1)", err: false},
		{stmt: "_ = mat2(0) == mat2(1)", err: true},
		{stmt: "_ = mat3(0) == mat3(1)", err: true},
		{stmt: "_ = mat4(0) == mat4(1)", err: true},

		{stmt: "_ = false != true", err: false},
		{stmt: "_ = int(0) != int(1)", err: false},
		{stmt: "_ = float(0) != float(1)", err: false},
		{stmt: "_ = vec2(0) != vec2(1)", err: false},
		{stmt: "_ = vec3(0) != vec3(1)", err: false},
		{stmt: "_ = vec4(0) != vec4(1)", err: false},
		{stmt: "_ = ivec2(0) != ivec2(1)", err: false},
		{stmt: "_ = ivec3(0) != ivec3(1)", err: false},
		{stmt: "_ = ivec4(0) != ivec4(1)", err: false},
		{stmt: "_ = mat2(0) != mat2(1)", err: true},
		{stmt: "_ = mat3(0) != mat3(1)", err: true},
		{stmt: "_ = mat4(0) != mat4(1)", err: true},

		{stmt: "_ = false < true", err: true},
		{stmt: "_ = int(0) < int(1)", err: false},
		{stmt: "_ = float(0) < float(1)", err: false},
		{stmt: "_ = vec2(0) < vec2(1)", err: true},
		{stmt: "_ = vec3(0) < vec3(1)", err: true},
		{stmt: "_ = vec4(0) < vec4(1)", err: true},
		{stmt: "_ = ivec2(0) < ivec2(1)", err: true},
		{stmt: "_ = ivec3(0) < ivec3(1)", err: true},
		{stmt: "_ = ivec4(0) < ivec4(1)", err: true},
		{stmt: "_ = mat2(0) < mat2(1)", err: true},
		{stmt: "_ = mat3(0) < mat3(1)", err: true},
		{stmt: "_ = mat4(0) < mat4(1)", err: true},

		{stmt: "_ = false <= true", err: true},
		{stmt: "_ = int(0) <= int(1)", err: false},
		{stmt: "_ = float(0) <= float(1)", err: false},
		{stmt: "_ = vec2(0) <= vec2(1)", err: true},
		{stmt: "_ = vec3(0) <= vec3(1)", err: true},
		{stmt: "_ = vec4(0) <= vec4(1)", err: true},
		{stmt: "_ = ivec2(0) <= ivec2(1)", err: true},
		{stmt: "_ = ivec3(0) <= ivec3(1)", err: true},
		{stmt: "_ = ivec4(0) <= ivec4(1)", err: true},
		{stmt: "_ = mat2(0) <= mat2(1)", err: true},
		{stmt: "_ = mat3(0) <= mat3(1)", err: true},
		{stmt: "_ = mat4(0) <= mat4(1)", err: true},

		{stmt: "_ = false > true", err: true},
		{stmt: "_ = int(0) > int(1)", err: false},
		{stmt: "_ = float(0) > float(1)", err: false},
		{stmt: "_ = vec2(0) > vec2(1)", err: true},
		{stmt: "_ = vec3(0) > vec3(1)", err: true},
		{stmt: "_ = vec4(0) > vec4(1)", err: true},
		{stmt: "_ = ivec2(0) > ivec2(1)", err: true},
		{stmt: "_ = ivec3(0) > ivec3(1)", err: true},
		{stmt: "_ = ivec4(0) > ivec4(1)", err: true},
		{stmt: "_ = mat2(0) > mat2(1)", err: true},
		{stmt: "_ = mat3(0) > mat3(1)", err: true},
		{stmt: "_ = mat4(0) > mat4(1)", err: true},

		{stmt: "_ = false >= true", err: true},
		{stmt: "_ = int(0) >= int(1)", err: false},
		{stmt: "_ = float(0) >= float(1)", err: false},
		{stmt: "_ = vec2(0) >= vec2(1)", err: true},
		{stmt: "_ = vec3(0) >= vec3(1)", err: true},
		{stmt: "_ = vec4(0) >= vec4(1)", err: true},
		{stmt: "_ = ivec2(0) >= ivec2(1)", err: true},
		{stmt: "_ = ivec3(0) >= ivec3(1)", err: true},
		{stmt: "_ = ivec4(0) >= ivec4(1)", err: true},
		{stmt: "_ = mat2(0) >= mat2(1)", err: true},
		{stmt: "_ = mat3(0) >= mat3(1)", err: true},
		{stmt: "_ = mat4(0) >= mat4(1)", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2680
func TestSyntaxForWithLocalVariable(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func foo() {
	i := 0
	for i = 0; i < 1; i++ {
	}
}`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

func foo() {
	for i, j := 0, 0; i < 1; i++ {
		_ = j
	}
}`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

// Issue #2648
func TestSyntaxDuplicatedUniformVariables(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

var Foo int
var Foo int
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

var Foo int
var Bar float
var Foo vec2
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

// Issue #2747
func TestSyntaxMultipleAssignmentsAndTypeCheck(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() (float, bool) {
	return 0, false
}

func Bar() {
	f, b := Foo()
	_, _ = f, b
	return
}
`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func Foo() (float, bool) {
	return 0, false
}

func Bar() {
	var f float
	var b bool
	f, b = Foo()
	_, _ = f, b
	return
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Foo() {
	a, b := 0
	_, _ = a, b
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() {
	a, b, c := 0, 0
	_, _ = a, b, c
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() {
	var a, b int
	a, b = 0
	_, _ = a, b
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() {
	var a, b, c int
	a, b, c = 0, 0
	_, _ = a, b, c
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

func TestSyntaxBitwiseOperator(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "_ = false & true", err: true},
		{stmt: "_ = int(0) & int(1)", err: false},
		{stmt: "_ = float(0) & float(1)", err: true},
		{stmt: "_ = vec2(0) & vec2(1)", err: true},
		{stmt: "_ = vec3(0) & vec3(1)", err: true},
		{stmt: "_ = vec4(0) & vec4(1)", err: true},
		{stmt: "_ = ivec2(0) & ivec2(1)", err: false},
		{stmt: "_ = ivec3(0) & ivec3(1)", err: false},
		{stmt: "_ = ivec4(0) & ivec4(1)", err: false},
		{stmt: "_ = ivec2(0) & int(1)", err: false},
		{stmt: "_ = ivec3(0) & int(1)", err: false},
		{stmt: "_ = ivec4(0) & int(1)", err: false},
		{stmt: "_ = int(0) & ivec2(1)", err: false},
		{stmt: "_ = int(0) & ivec3(1)", err: false},
		{stmt: "_ = int(0) & ivec4(1)", err: false},
		{stmt: "_ = mat2(0) & mat2(1)", err: true},
		{stmt: "_ = mat3(0) & mat3(1)", err: true},
		{stmt: "_ = mat4(0) & mat4(1)", err: true},

		{stmt: "_ = false | true", err: true},
		{stmt: "_ = int(0) | int(1)", err: false},
		{stmt: "_ = float(0) | float(1)", err: true},
		{stmt: "_ = vec2(0) | vec2(1)", err: true},
		{stmt: "_ = vec3(0) | vec3(1)", err: true},
		{stmt: "_ = vec4(0) | vec4(1)", err: true},
		{stmt: "_ = ivec2(0) | ivec2(1)", err: false},
		{stmt: "_ = ivec3(0) | ivec3(1)", err: false},
		{stmt: "_ = ivec4(0) | ivec4(1)", err: false},
		{stmt: "_ = ivec2(0) | int(1)", err: false},
		{stmt: "_ = ivec3(0) | int(1)", err: false},
		{stmt: "_ = ivec4(0) | int(1)", err: false},
		{stmt: "_ = int(0) | ivec2(1)", err: false},
		{stmt: "_ = int(0) | ivec3(1)", err: false},
		{stmt: "_ = int(0) | ivec4(1)", err: false},
		{stmt: "_ = mat2(0) | mat2(1)", err: true},
		{stmt: "_ = mat3(0) | mat3(1)", err: true},
		{stmt: "_ = mat4(0) | mat4(1)", err: true},

		{stmt: "_ = false ^ true", err: true},
		{stmt: "_ = int(0) ^ int(1)", err: false},
		{stmt: "_ = float(0) ^ float(1)", err: true},
		{stmt: "_ = vec2(0) ^ vec2(1)", err: true},
		{stmt: "_ = vec3(0) ^ vec3(1)", err: true},
		{stmt: "_ = vec4(0) ^ vec4(1)", err: true},
		{stmt: "_ = ivec2(0) ^ ivec2(1)", err: false},
		{stmt: "_ = ivec3(0) ^ ivec3(1)", err: false},
		{stmt: "_ = ivec4(0) ^ ivec4(1)", err: false},
		{stmt: "_ = ivec2(0) ^ int(1)", err: false},
		{stmt: "_ = ivec3(0) ^ int(1)", err: false},
		{stmt: "_ = ivec4(0) ^ int(1)", err: false},
		{stmt: "_ = int(0) ^ ivec2(1)", err: false},
		{stmt: "_ = int(0) ^ ivec3(1)", err: false},
		{stmt: "_ = int(0) ^ ivec4(1)", err: false},
		{stmt: "_ = mat2(0) ^ mat2(1)", err: true},
		{stmt: "_ = mat3(0) ^ mat3(1)", err: true},
		{stmt: "_ = mat4(0) ^ mat4(1)", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

// Issue #2891
func TestSyntaxInvalidArgument(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo(x int) int {
	return 0
}

func Bar() int {
	return Foo(Foo)
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

// Issue #2891, #2910
func TestSyntaxTailingUnaryOperator(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() {
	1 + x := vec2(2)
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() {
	1 + x, y := Bar()
}

func Bar() (int, int) {
	return 0, 0
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

// Issue #2926, #2989
func TestSyntaxNonTypeExpression(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Bar() float {
	return +Foo
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Bar() float {
	return Foo + 1.0
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Bar() float {
	return 1.0 + Foo
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Bar() float {
	return Foo.x
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Bar() float {
	return Foo[0]
}
`)); err == nil {
		t.Error("compileToIR must return an error but did not")
	}
}

// Issue #2993
func TestSyntaxIfAndConstBool(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() int {
	const X = true
	if X {
		return 1
	}
	return 0
}
`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func Foo() int {
	const X bool = true
	if X {
		return 1
	}
	return 0
}
`)); err != nil {
		t.Error(err)
	}
}

// Issue #3111
func TestSyntaxTooManyElementsAtInitialization(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "_ = [-1]int{}", err: true},
		{stmt: "_ = [-1]int{0}", err: true},
		{stmt: "_ = [-1]int{0, 0}", err: true},
		{stmt: "_ = [-1]int{0, 0, 0}", err: true},
		{stmt: "_ = [0]int{}", err: false},
		{stmt: "_ = [0]int{0}", err: true},
		{stmt: "_ = [0]int{0, 0}", err: true},
		{stmt: "_ = [0]int{0, 0, 0}", err: true},
		{stmt: "_ = [1]int{}", err: false},
		{stmt: "_ = [1]int{0}", err: false},
		{stmt: "_ = [1]int{0, 0}", err: true},
		{stmt: "_ = [1]int{0, 0, 0}", err: true},
		{stmt: "_ = [2]int{}", err: false},
		{stmt: "_ = [2]int{0}", err: false},
		{stmt: "_ = [2]int{0, 0}", err: false},
		{stmt: "_ = [2]int{0, 0, 0}", err: true},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}

func TestSyntaxArrayOutOfBounds(t *testing.T) {
	cases := []struct {
		stmt string
		err  bool
	}{
		{stmt: "a := [0]int{}; _ = a[-1]", err: true},
		{stmt: "a := [0]int{}; _ = a[0]", err: true},
		{stmt: "a := [0]int{}; _ = a[1]", err: true},
		{stmt: "a := [0]int{}; _ = a[2]", err: true},
		{stmt: "a := [0]int{}; _ = a[3]", err: true},
		{stmt: "a := [0]int{}; b := -1; _ = a[b]", err: false},
		{stmt: "a := [0]int{}; b := 0; _ = a[b]", err: false},
		{stmt: "a := [0]int{}; b := 1; _ = a[b]", err: false},
		{stmt: "a := [0]int{}; b := 2; _ = a[b]", err: false},
		{stmt: "a := [0]int{}; b := 3; _ = a[b]", err: false},

		{stmt: "a := [1]int{}; _ = a[-1]", err: true},
		{stmt: "a := [1]int{}; _ = a[0]", err: false},
		{stmt: "a := [1]int{}; _ = a[1]", err: true},
		{stmt: "a := [1]int{}; _ = a[2]", err: true},
		{stmt: "a := [1]int{}; _ = a[3]", err: true},
		{stmt: "a := [1]int{}; b := -1; _ = a[b]", err: false},
		{stmt: "a := [1]int{}; b := 0; _ = a[b]", err: false},
		{stmt: "a := [1]int{}; b := 1; _ = a[b]", err: false},
		{stmt: "a := [1]int{}; b := 2; _ = a[b]", err: false},
		{stmt: "a := [1]int{}; b := 3; _ = a[b]", err: false},

		{stmt: "a := [2]int{}; _ = a[-1]", err: true},
		{stmt: "a := [2]int{}; _ = a[0]", err: false},
		{stmt: "a := [2]int{}; _ = a[1]", err: false},
		{stmt: "a := [2]int{}; _ = a[2]", err: true},
		{stmt: "a := [2]int{}; _ = a[3]", err: true},
		{stmt: "a := [2]int{}; b := -1; _ = a[b]", err: false},
		{stmt: "a := [2]int{}; b := 0; _ = a[b]", err: false},
		{stmt: "a := [2]int{}; b := 1; _ = a[b]", err: false},
		{stmt: "a := [2]int{}; b := 2; _ = a[b]", err: false},
		{stmt: "a := [2]int{}; b := 3; _ = a[b]", err: false},

		{stmt: "a := vec2(0); _ = a[-1]", err: true},
		{stmt: "a := vec2(0); _ = a[0]", err: false},
		{stmt: "a := vec2(0); _ = a[1]", err: false},
		{stmt: "a := vec2(0); _ = a[2]", err: true},
		{stmt: "a := vec2(0); _ = a[3]", err: true},
		{stmt: "a := vec2(0); b := -1; _ = a[b]", err: false},
		{stmt: "a := vec2(0); b := 0; _ = a[b]", err: false},
		{stmt: "a := vec2(0); b := 1; _ = a[b]", err: false},
		{stmt: "a := vec2(0); b := 2; _ = a[b]", err: false},
		{stmt: "a := vec2(0); b := 3; _ = a[b]", err: false},

		{stmt: "a := mat3(0); _ = a[-1]", err: true},
		{stmt: "a := mat3(0); _ = a[0]", err: false},
		{stmt: "a := mat3(0); _ = a[1]", err: false},
		{stmt: "a := mat3(0); _ = a[2]", err: false},
		{stmt: "a := mat3(0); _ = a[3]", err: true},
		{stmt: "a := mat3(0); b := -1; _ = a[b]", err: false},
		{stmt: "a := mat3(0); b := 0; _ = a[b]", err: false},
		{stmt: "a := mat3(0); b := 1; _ = a[b]", err: false},
		{stmt: "a := mat3(0); b := 2; _ = a[b]", err: false},
		{stmt: "a := mat3(0); b := 3; _ = a[b]", err: false},
	}

	for _, c := range cases {
		stmt := c.stmt
		src := fmt.Sprintf(`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
	return dstPos
}`, stmt)
		_, err := compileToIR([]byte(src))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", stmt, err)
		}
	}
}
