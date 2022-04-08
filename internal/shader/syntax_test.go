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
	"go/parser"
	"go/token"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func compileToIR(src []byte) (*shaderir.Program, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	ir, err := shader.Compile(fset, f, "Vertex", "Fragment", 0)
	if err != nil {
		return nil, err
	}

	return ir, nil
}

func TestSyntaxShadowing(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var position vec4
	return position
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxDuplicatedVariables(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo vec4
	var foo vec4
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo, foo vec4
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	foo, foo := Foo()
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxDuplicatedFunctions(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Foo() {
}

func Foo() {
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxNoNewVariables(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_ := 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_, _ := Foo()
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a, _ := 1, 1
	_ = a
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return 0.0
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (float, float) {
	return 0
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() float {
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() float {
	return 0.0, 0.0
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo() (float, float, float) {
	return 0.0, 0.0
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxUnspportedSyntax(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := func() {
	}
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	go func() {
	}()
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	ch := make(chan int)
	_ = ch
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 1i
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x [4]float
	y := x[1:2]
	_ = y
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	U = vec4(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var U vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	U.x = 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var U [2]vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	U[0] = vec4(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	texCoord = vec2(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	texCoord.x = 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxBoolLiteral(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	true := vec4(0)
	return true
}
`)); err != nil {
		t.Errorf("error must be nil but was non-nil")
	}
}

func TestSyntaxUnusedVariable(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 0
	x = 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := vec4(0)
	x.x = 1
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	// Increment statement treats a variable 'used'.
	// https://play.golang.org/p/2RuYMrSLjt3
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 0
	x++
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a int
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a, b int
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxBlankLhs(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x int = _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 1
	x = _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 1 + _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_++
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_ += 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_.x = 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxDuplicatedVarsAndConstants(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a = 0
	const a = 0
	return vec4(a)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	const a = 0
	var a = 0
	return vec4(a)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	const a = 0
	const a = 0
	return vec4(a)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

const U0 = 0
var U0 float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(a)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var U0 float
const U0 = 0

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	Foo(1)
	return position
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Foo(x float) {
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	Foo()
	return position
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	Foo(Bar())
	return position
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	Foo(Bar())
	return position
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	position
	return position
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

var Time float
var ScreenSize vec2

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	vec2(position)
	return position
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #1947
func TestSyntaxOperatorMod(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2.0 % 0.5
	return vec4(a)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := int(2) % 0.5
	return vec4(a)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := int(2) % 1.0
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2.0
	b := 0.5
	return vec4(a % b)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2
	b := 0.5
	return vec4(a % b)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2.5
	b := 1
	return vec4(a % b)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2
	b := 1
	return vec4(a % b)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2
	return vec4(a % 1)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 1
	return vec4(2 % a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2
	a %= 1
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2
	a %= 1.0
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2
	a %= 0.5
	return vec4(a)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 2.0
	a %= 1
	return vec4(a)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestSyntaxOperatorAssign(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 1.0
	a += 2
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 1.0
	a += 2.0
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 1.0
	a += 2.1
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 1
	a += 2
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 1
	a += 2.0
	return vec4(a)
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := 1
	a += 2.1
	return vec4(a)
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

// Issue #1963
func TestSyntaxOperatorVecAndNumber(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1) + 2
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1) + 2.1
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1) % 2
	return a.xxyy
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1) % 2.1
	return a.xxyy
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1)
	a += 2
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1)
	a += 2.1
	return a.xxyy
}`)); err != nil {
		t.Error(err)
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1)
	a %= 2
	return a.xxyy
}`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := vec2(1)
	a %= 2.1
	return a.xxyy
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
		{stmt: "a := 1 + vec2(2); _ = a", err: false},
		{stmt: "a := int(1) + vec2(2); _ = a", err: true},
		{stmt: "a := 1.0 / vec2(2); _ = a", err: false},
		{stmt: "a := 1.0 + vec2(2); _ = a", err: false},
		{stmt: "a := 1 * vec3(2); _ = a", err: false},
		{stmt: "a := 1.0 * vec3(2); _ = a", err: false},
		{stmt: "a := 1 * vec4(2); _ = a", err: false},
		{stmt: "a := 1.0 * vec4(2); _ = a", err: false},
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
		{stmt: "a := vec2(1) / 2.0; _ = a", err: false},
		{stmt: "a := vec2(1) + 2.0; _ = a", err: false},
		{stmt: "a := vec2(1) * int(2); _ = a", err: true},
		{stmt: "a := vec2(1) * vec2(2); _ = a", err: false},
		{stmt: "a := vec2(1) + vec2(2); _ = a", err: false},
		{stmt: "a := vec2(1) * vec3(2); _ = a", err: true},
		{stmt: "a := vec2(1) * vec4(2); _ = a", err: true},
		{stmt: "a := vec2(1) * mat2(2); _ = a", err: false},
		{stmt: "a := vec2(1) + mat2(2); _ = a", err: true},
		{stmt: "a := vec2(1) * mat3(2); _ = a", err: true},
		{stmt: "a := vec2(1) * mat4(2); _ = a", err: true},
		{stmt: "a := mat2(1) * 2; _ = a", err: false},
		{stmt: "a := mat2(1) * 2.0; _ = a", err: false},
		{stmt: "a := mat2(1) / 2.0; _ = a", err: false},
		{stmt: "a := mat2(1) / float(2); _ = a", err: false},
		{stmt: "a := mat2(1) * int(2); _ = a", err: true},
		{stmt: "a := mat2(1) + 2.0; _ = a", err: true},
		{stmt: "a := mat2(1) + float(2); _ = a", err: true},
		{stmt: "a := mat2(1) * vec2(2); _ = a", err: false},
		{stmt: "a := mat2(1) + vec2(2); _ = a", err: true},
		{stmt: "a := mat2(1) * vec3(2); _ = a", err: true},
		{stmt: "a := mat2(1) * vec4(2); _ = a", err: true},
		{stmt: "a := mat2(1) * mat2(2); _ = a", err: false},
		{stmt: "a := mat2(1) / mat2(2); _ = a", err: true},
		{stmt: "a := mat2(1) * mat3(2); _ = a", err: true},
		{stmt: "a := mat2(1) * mat4(2); _ = a", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	%s
	return position
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
		{stmt: "a := 1.0; a *= int(2)", err: true},
		{stmt: "a := 1.0; a *= vec2(2)", err: true},
		{stmt: "a := 1.0; a *= vec3(2)", err: true},
		{stmt: "a := 1.0; a *= vec4(2)", err: true},
		{stmt: "a := 1.0; a *= mat2(2)", err: true},
		{stmt: "a := 1.0; a *= mat3(2)", err: true},
		{stmt: "a := 1.0; a *= mat4(2)", err: true},
		{stmt: "a := vec2(1); a *= 2", err: false},
		{stmt: "a := vec2(1); a *= 2.0", err: false},
		{stmt: "a := vec2(1); a /= 2.0", err: false},
		{stmt: "a := vec2(1); a += 2.0", err: false},
		{stmt: "a := vec2(1); a *= int(2)", err: true},
		{stmt: "a := vec2(1); a *= float(2)", err: false},
		{stmt: "a := vec2(1); a /= float(2)", err: false},
		{stmt: "a := vec2(1); a *= vec2(2)", err: false},
		{stmt: "a := vec2(1); a += vec2(2)", err: false},
		{stmt: "a := vec2(1); a *= vec3(2)", err: true},
		{stmt: "a := vec2(1); a *= vec4(2)", err: true},
		{stmt: "a := vec2(1); a *= mat2(2)", err: false},
		{stmt: "a := vec2(1); a += mat2(2)", err: true},
		{stmt: "a := vec2(1); a /= mat2(2)", err: true},
		{stmt: "a := vec2(1); a *= mat3(2)", err: true},
		{stmt: "a := vec2(1); a *= mat4(2)", err: true},
		{stmt: "a := mat2(1); a *= 2", err: false},
		{stmt: "a := mat2(1); a *= 2.0", err: false},
		{stmt: "a := mat2(1); a /= 2.0", err: false},
		{stmt: "a := mat2(1); a += 2.0", err: true},
		{stmt: "a := mat2(1); a *= int(2)", err: true},
		{stmt: "a := mat2(1); a *= float(2)", err: false},
		{stmt: "a := mat2(1); a /= float(2)", err: false},
		{stmt: "a := mat2(1); a *= vec2(2)", err: true},
		{stmt: "a := mat2(1); a += vec2(2)", err: true},
		{stmt: "a := mat2(1); a *= vec3(2)", err: true},
		{stmt: "a := mat2(1); a *= vec4(2)", err: true},
		{stmt: "a := mat2(1); a *= mat2(2)", err: false},
		{stmt: "a := mat2(1); a += mat2(2)", err: false},
		{stmt: "a := mat2(1); a /= mat2(2)", err: true},
		{stmt: "a := mat2(1); a *= mat3(2)", err: true},
		{stmt: "a := mat2(1); a *= mat4(2)", err: true},
	}

	for _, c := range cases {
		_, err := compileToIR([]byte(fmt.Sprintf(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	%s
	return position
}`, c.stmt)))
		if err == nil && c.err {
			t.Errorf("%s must return an error but does not", c.stmt)
		} else if err != nil && !c.err {
			t.Errorf("%s must not return nil but returned %v", c.stmt, err)
		}
	}
}

func TestSyntaxAtan(t *testing.T) {
	// `atan` takes 1 argument.
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return atan(vec4(0))
}
`)); err != nil {
		t.Error(err)
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return atan(vec4(0), vec4(0))
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	// `atan2` takes 2 arguments.
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return atan2(vec4(0))
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return atan2(vec4(0), vec4(0))
}
`)); err != nil {
		t.Error(err)
	}
}

// Issue #1972
func TestSyntaxType(t *testing.T) {
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x vec2 = vec3(0)
	_ = x
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x, y vec2 = vec2(0), vec3(0)
	_, _ = x, y
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x vec2
	x = vec3(0)
	_ = x
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var _ vec2 = vec3(0)
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
	if _, err := compileToIR([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var _, _ vec2 = Foo()
	return color
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}
