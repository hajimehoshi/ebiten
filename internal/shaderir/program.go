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
	"bytes"
	"encoding/hex"
	"go/constant"
	"go/token"
	"hash/fnv"
	"sort"
	"strings"
)

type Unit int

const (
	Texels Unit = iota
	Pixels
)

type SourceHash [16]byte

func CalcSourceHash(source []byte) SourceHash {
	h := fnv.New128a()
	_, _ = h.Write(bytes.TrimSpace(source))

	var hash SourceHash
	h.Sum(hash[:0])
	return hash
}

func (s SourceHash) String() string {
	return hex.EncodeToString(s[:])
}

type Program struct {
	UniformNames []string
	Uniforms     []Type
	TextureCount int
	Attributes   []Type
	Varyings     []Type
	Funcs        []Func
	VertexFunc   VertexFunc
	FragmentFunc FragmentFunc
	Unit         Unit

	SourceHash SourceHash

	uniformFactors []uint32
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
// If index == len(attributes), the param is an out-param and represents the position in vec4 (gl_Position in GLSL)
// If len(attributes) + 1 <= index < len(attributes) + len(varyings) + 1, the params are out-params and represent
// varying variables.
type VertexFunc struct {
	Block *Block
}

// FragmentFunc takes pseudo params, and the number is len(varyings) + 2.
// If index == 0, the param represents the coordinate of the fragment (gl_FragCoord in GLSL).
// If 0 < index <= len(varyings), the param represents (index-1)th varying variable.
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

type Expr struct {
	Type        ExprType
	Exprs       []Expr
	Const       constant.Value
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

type Op int

const (
	Add Op = iota
	Sub
	NotOp
	ComponentWiseMul
	MatrixMul
	Div
	ModOp
	LeftShift
	RightShift
	LessThanOp
	LessThanEqualOp
	GreaterThanOp
	GreaterThanEqualOp
	EqualOp
	NotEqualOp
	VectorEqualOp
	VectorNotEqualOp
	And
	Xor
	Or
	AndAnd
	OrOr
)

func OpFromToken(t token.Token, lhs, rhs Type) (Op, bool) {
	switch t {
	case token.ADD:
		return Add, true
	case token.SUB:
		return Sub, true
	case token.NOT:
		return NotOp, true
	case token.MUL:
		if lhs.IsMatrix() || rhs.IsMatrix() {
			return MatrixMul, true
		}
		return ComponentWiseMul, true
	case token.QUO:
		return Div, true
	case token.QUO_ASSIGN:
		// QUO_ASSIGN indicates an integer division.
		// https://pkg.go.dev/go/constant/#BinaryOp
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
		if lhs.IsFloatVector() || lhs.IsIntVector() || rhs.IsFloatVector() || rhs.IsIntVector() {
			return VectorEqualOp, true
		}
		return EqualOp, true
	case token.NEQ:
		if lhs.IsFloatVector() || lhs.IsIntVector() || rhs.IsFloatVector() || rhs.IsIntVector() {
			return VectorNotEqualOp, true
		}
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
	return 0, false
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
	IVec2F      BuiltinFunc = "ivec2"
	IVec3F      BuiltinFunc = "ivec3"
	IVec4F      BuiltinFunc = "ivec4"
	Mat2F       BuiltinFunc = "mat2"
	Mat3F       BuiltinFunc = "mat3"
	Mat4F       BuiltinFunc = "mat4"
	Radians     BuiltinFunc = "radians" // This function is not used yet (#2253)
	Degrees     BuiltinFunc = "degrees" // This function is not used yet (#2253)
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
	Refract     BuiltinFunc = "refract"
	Transpose   BuiltinFunc = "transpose"
	Dfdx        BuiltinFunc = "dfdx"
	Dfdy        BuiltinFunc = "dfdy"
	Fwidth      BuiltinFunc = "fwidth"
	DiscardF    BuiltinFunc = "discard"
	TexelAt     BuiltinFunc = "__texelAt"
	FrontFacing BuiltinFunc = "frontfacing"
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
		IVec2F,
		IVec3F,
		IVec4F,
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
		Refract,
		Transpose,
		Dfdx,
		Dfdy,
		Fwidth,
		DiscardF,
		TexelAt,
		FrontFacing:
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
			if !strings.ContainsRune(xyzw, c) {
				return false
			}
		}
		return true
	case strings.IndexByte(rgba, s[0]) >= 0:
		for _, c := range s {
			if !strings.ContainsRune(rgba, c) {
				return false
			}
		}
		return true
	case strings.IndexByte(strq, s[0]) >= 0:
		for _, c := range s {
			if !strings.ContainsRune(strq, c) {
				return false
			}
		}
		return true
	}
	return false
}

func (p *Program) ReachableFuncsFromBlock(block *Block) []*Func {
	indexToFunc := map[int]*Func{}
	for _, f := range p.Funcs {
		f := f
		indexToFunc[f.Index] = &f
	}

	visited := map[int]struct{}{}
	var indices []int
	var f func(expr *Expr)
	f = func(expr *Expr) {
		if expr.Type != FunctionExpr {
			return
		}
		if _, ok := visited[expr.Index]; ok {
			return
		}
		indices = append(indices, expr.Index)
		visited[expr.Index] = struct{}{}
		walkExprs(f, indexToFunc[expr.Index].Block)
	}
	walkExprs(f, block)

	sort.Ints(indices)

	funcs := make([]*Func, 0, len(indices))
	for _, i := range indices {
		funcs = append(funcs, indexToFunc[i])
	}
	return funcs
}

func walkExprs(f func(expr *Expr), block *Block) {
	if block == nil {
		return
	}
	for _, s := range block.Stmts {
		for _, e := range s.Exprs {
			walkExprsInExpr(f, &e)
		}
		for _, b := range s.Blocks {
			walkExprs(f, b)
		}
	}
}

func walkExprsInExpr(f func(expr *Expr), expr *Expr) {
	if expr == nil {
		return
	}
	f(expr)
	for _, e := range expr.Exprs {
		walkExprsInExpr(f, &e)
	}
}

func (p *Program) appendReachableUniformVariablesFromBlock(indices []int, block *Block) []int {
	indexToFunc := map[int]*Func{}
	for _, f := range p.Funcs {
		f := f
		indexToFunc[f.Index] = &f
	}

	visitedFuncs := map[int]struct{}{}
	indicesSet := map[int]struct{}{}
	var f func(expr *Expr)
	f = func(expr *Expr) {
		switch expr.Type {
		case UniformVariable:
			if _, ok := indicesSet[expr.Index]; ok {
				return
			}
			indicesSet[expr.Index] = struct{}{}
			indices = append(indices, expr.Index)
		case FunctionExpr:
			if _, ok := visitedFuncs[expr.Index]; ok {
				return
			}
			visitedFuncs[expr.Index] = struct{}{}
			walkExprs(f, indexToFunc[expr.Index].Block)
		}
	}
	walkExprs(f, block)

	return indices
}

// FilterUniformVariables replaces uniform variables with 0 when they are not used.
// By minimizing uniform variables, more commands can be merged in the graphicscommand package.
func (p *Program) FilterUniformVariables(uniforms []uint32) {
	if p.uniformFactors == nil {
		indices := p.appendReachableUniformVariablesFromBlock(nil, p.VertexFunc.Block)
		indices = p.appendReachableUniformVariablesFromBlock(indices, p.FragmentFunc.Block)
		reachableUniforms := make([]bool, len(p.Uniforms))
		for _, idx := range indices {
			reachableUniforms[idx] = true
		}
		p.uniformFactors = make([]uint32, len(uniforms))
		var idx int
		for i, typ := range p.Uniforms {
			c := typ.DwordCount()
			if reachableUniforms[i] {
				for i := idx; i < idx+c; i++ {
					p.uniformFactors[i] = 1
				}
			}
			idx += c
		}
	}

	for i, factor := range p.uniformFactors {
		uniforms[i] *= factor
	}
}
