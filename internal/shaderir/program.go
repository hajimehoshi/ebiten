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

// Package shaderir offers intermediate representation for shader programs.
package shaderir

import (
	"go/constant"
	"go/token"
	"strings"
)

type Program struct {
	UniformNames []string
	Uniforms     []Type
	TextureNum   int
	Attributes   []Type
	Varyings     []Type
	Funcs        []Func
	VertexFunc   VertexFunc
	FragmentFunc FragmentFunc
}

type Func struct {
	Index     int
	InParams  []Type
	OutParams []Type
	Return    Type
	Block     *Block
}

// VertexFunc takes pseudo params, and the number if len(attributes) + len(varyings) + 1.
// If 0 <= index < len(attributes), the params are in-params and represent attribute variables.
// If index == len(attributes), the param is an out-param and repesents the position in vec4 (gl_Position in GLSL)
// If len(attributes) + 1 <= index < len(attributes) + len(varyings) + 1, the params are out-params and represent
// varying variables.
type VertexFunc struct {
	Block *Block
}

// FragmentFunc takes pseudo params, and the number is len(varyings) + 2.
// If index == 0, the param represents the coordinate of the fragment (gl_FragCoord in GLSL).
// If index == len(varyings), the param represents (index-1)th verying variable.
// If index == len(varyings)+1, the param is an out-param representing the color of the pixel (gl_FragColor in GLSL).
type FragmentFunc struct {
	Block *Block
}

type Block struct {
	LocalVars           []Type
	LocalVarIndexOffset int
	Stmts               []Stmt
}

type Stmt struct {
	Type        StmtType
	Exprs       []Expr
	Blocks      []*Block
	ForVarType  Type
	ForVarIndex int
	ForInit     constant.Value
	ForEnd      constant.Value
	ForOp       Op
	ForDelta    constant.Value
	InitIndex   int
}

type StmtType int

const (
	ExprStmt StmtType = iota
	BlockStmt
	Assign
	Init
	If
	For
	Continue
	Break
	Return
	Discard
)

type ConstType int

const (
	ConstTypeNone ConstType = iota
	ConstTypeBool
	ConstTypeInt
	ConstTypeFloat
)

type Expr struct {
	Type        ExprType
	Exprs       []Expr
	Const       constant.Value
	ConstType   ConstType
	BuiltinFunc BuiltinFunc
	Swizzling   string
	Index       int
	Op          Op
}

type ExprType int

const (
	Blank ExprType = iota
	NumberExpr
	UniformVariable
	TextureVariable
	LocalVariable
	StructMember
	BuiltinFuncExpr
	SwizzlingExpr
	FunctionExpr
	Unary
	Binary
	Selection
	Call
	FieldSelector
	Index
)

type Op string

const (
	Add                Op = "+"
	Sub                Op = "-"
	NotOp              Op = "!"
	Mul                Op = "*"
	Div                Op = "/"
	ModOp              Op = "%"
	LeftShift          Op = "<<"
	RightShift         Op = ">>"
	LessThanOp         Op = "<"
	LessThanEqualOp    Op = "<="
	GreaterThanOp      Op = ">"
	GreaterThanEqualOp Op = ">="
	EqualOp            Op = "=="
	NotEqualOp         Op = "!="
	And                Op = "&"
	Xor                Op = "^"
	Or                 Op = "|"
	AndAnd             Op = "&&"
	OrOr               Op = "||"
)

func OpFromToken(t token.Token) (Op, bool) {
	switch t {
	case token.ADD:
		return Add, true
	case token.SUB:
		return Sub, true
	case token.NOT:
		return NotOp, true
	case token.MUL:
		return Mul, true
	case token.QUO:
		return Div, true
	case token.REM:
		return ModOp, true
	case token.SHL:
		return LeftShift, true
	case token.SHR:
		return RightShift, true
	case token.LSS:
		return LessThanOp, true
	case token.LEQ:
		return LessThanEqualOp, true
	case token.GTR:
		return GreaterThanOp, true
	case token.GEQ:
		return GreaterThanEqualOp, true
	case token.EQL:
		return EqualOp, true
	case token.NEQ:
		return NotEqualOp, true
	case token.AND:
		return And, true
	case token.XOR:
		return Xor, true
	case token.OR:
		return Or, true
	case token.LAND:
		return AndAnd, true
	case token.LOR:
		return OrOr, true
	}
	return "", false
}

type BuiltinFunc string

const (
	Len         BuiltinFunc = "len"
	Cap         BuiltinFunc = "cap"
	BoolF       BuiltinFunc = "bool"
	IntF        BuiltinFunc = "int"
	FloatF      BuiltinFunc = "float"
	Vec2F       BuiltinFunc = "vec2"
	Vec3F       BuiltinFunc = "vec3"
	Vec4F       BuiltinFunc = "vec4"
	Mat2F       BuiltinFunc = "mat2"
	Mat3F       BuiltinFunc = "mat3"
	Mat4F       BuiltinFunc = "mat4"
	Radians     BuiltinFunc = "radians"
	Degrees     BuiltinFunc = "degrees"
	Sin         BuiltinFunc = "sin"
	Cos         BuiltinFunc = "cos"
	Tan         BuiltinFunc = "tan"
	Asin        BuiltinFunc = "asin"
	Acos        BuiltinFunc = "acos"
	Atan        BuiltinFunc = "atan"
	Atan2       BuiltinFunc = "atan2"
	Pow         BuiltinFunc = "pow"
	Exp         BuiltinFunc = "exp"
	Log         BuiltinFunc = "log"
	Exp2        BuiltinFunc = "exp2"
	Log2        BuiltinFunc = "log2"
	Sqrt        BuiltinFunc = "sqrt"
	Inversesqrt BuiltinFunc = "inversesqrt"
	Abs         BuiltinFunc = "abs"
	Sign        BuiltinFunc = "sign"
	Floor       BuiltinFunc = "floor"
	Ceil        BuiltinFunc = "ceil"
	Fract       BuiltinFunc = "fract"
	Mod         BuiltinFunc = "mod"
	Min         BuiltinFunc = "min"
	Max         BuiltinFunc = "max"
	Clamp       BuiltinFunc = "clamp"
	Mix         BuiltinFunc = "mix"
	Step        BuiltinFunc = "step"
	Smoothstep  BuiltinFunc = "smoothstep"
	Length      BuiltinFunc = "length"
	Distance    BuiltinFunc = "distance"
	Dot         BuiltinFunc = "dot"
	Cross       BuiltinFunc = "cross"
	Normalize   BuiltinFunc = "normalize"
	Faceforward BuiltinFunc = "faceforward"
	Reflect     BuiltinFunc = "reflect"
	Transpose   BuiltinFunc = "transpose"
	Texture2DF  BuiltinFunc = "texture2D"
	Dfdx        BuiltinFunc = "dfdx"
	Dfdy        BuiltinFunc = "dfdy"
	Fwidth      BuiltinFunc = "fwidth"
)

func ParseBuiltinFunc(str string) (BuiltinFunc, bool) {
	switch BuiltinFunc(str) {
	case Len,
		Cap,
		BoolF,
		IntF,
		FloatF,
		Vec2F,
		Vec3F,
		Vec4F,
		Mat2F,
		Mat3F,
		Mat4F,
		Sin,
		Cos,
		Tan,
		Asin,
		Acos,
		Atan,
		Atan2,
		Pow,
		Exp,
		Log,
		Exp2,
		Log2,
		Sqrt,
		Inversesqrt,
		Abs,
		Sign,
		Floor,
		Ceil,
		Fract,
		Mod,
		Min,
		Max,
		Clamp,
		Mix,
		Step,
		Smoothstep,
		Length,
		Distance,
		Dot,
		Cross,
		Normalize,
		Faceforward,
		Reflect,
		Transpose,
		Texture2DF,
		Dfdx,
		Dfdy,
		Fwidth:
		return BuiltinFunc(str), true
	}
	return "", false
}

func IsValidSwizzling(s string) bool {
	if len(s) < 1 || 4 < len(s) {
		return false
	}

	const (
		xyzw = "xyzw"
		rgba = "rgba"
		strq = "strq"
	)

	switch {
	case strings.IndexByte(xyzw, s[0]) >= 0:
		for _, c := range s {
			if strings.IndexRune(xyzw, c) == -1 {
				return false
			}
		}
		return true
	case strings.IndexByte(rgba, s[0]) >= 0:
		for _, c := range s {
			if strings.IndexRune(rgba, c) == -1 {
				return false
			}
		}
		return true
	case strings.IndexByte(strq, s[0]) >= 0:
		for _, c := range s {
			if strings.IndexRune(strq, c) == -1 {
				return false
			}
		}
		return true
	}
	return false
}

func (p *Program) ReferredFuncIndicesInVertexShader() []int {
	return p.referredFuncIndicesInBlockEntryPoint(p.VertexFunc.Block)
}

func (p *Program) ReferredFuncIndicesInFragmentShader() []int {
	return p.referredFuncIndicesInBlockEntryPoint(p.FragmentFunc.Block)
}

func (p *Program) referredFuncIndicesInBlockEntryPoint(b *Block) []int {
	indexToFunc := map[int]*Func{}
	for _, f := range p.Funcs {
		f := f
		indexToFunc[f.Index] = &f
	}
	visited := map[int]struct{}{}
	return referredFuncIndicesInBlock(b, indexToFunc, visited)
}

func referredFuncIndicesInBlock(b *Block, indexToFunc map[int]*Func, visited map[int]struct{}) []int {
	if b == nil {
		return nil
	}

	var fs []int

	for _, s := range b.Stmts {
		for _, e := range s.Exprs {
			fs = append(fs, referredFuncIndicesInExpr(&e, indexToFunc, visited)...)
		}
		for _, bb := range s.Blocks {
			fs = append(fs, referredFuncIndicesInBlock(bb, indexToFunc, visited)...)
		}
	}
	return fs
}

func referredFuncIndicesInExpr(e *Expr, indexToFunc map[int]*Func, visited map[int]struct{}) []int {
	var fs []int

	if e.Type == FunctionExpr {
		if _, ok := visited[e.Index]; !ok {
			fs = append(fs, e.Index)
			visited[e.Index] = struct{}{}
			fs = append(fs, referredFuncIndicesInBlock(indexToFunc[e.Index].Block, indexToFunc, visited)...)
		}
	}
	for _, ee := range e.Exprs {
		fs = append(fs, referredFuncIndicesInExpr(&ee, indexToFunc, visited)...)
	}
	return fs
}
