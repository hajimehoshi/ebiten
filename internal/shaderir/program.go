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

type Program struct {
	Uniforms     []Type
	Attributes   []Type
	Varyings     []Type
	Funcs        []Func
	VertexFunc   VertexFunc
	FragmentFunc FragmentFunc

	structNames map[string]string
	structTypes []Type
}

// TODO: How to avoid the name with existing functions?

type Func struct {
	Name        string
	InParams    []Type
	InOutParams []Type
	OutParams   []Type
	Return      Type
	Block       Block
}

// VertexFunc takes pseudo params, and the number if len(attributes) + len(varyings) + 1.
// If 0 <= index < len(attributes), the params are in-params and treated as attribute variables.
// If len(attributes) <= index < len(attributes) + len(varyings), the params are out-params and treated as varying
// variables.
// The last param represents the position in vec4 (gl_Position in GLSL).
type VertexFunc struct {
	Block Block
}

// FragmentFunc takes pseudo in-params, and the number is len(varyings) + 1.
// The last param represents the coordinate of the fragment (gl_FragCoord in GLSL)
type FragmentFunc struct {
	Block Block
}

type Block struct {
	LocalVars []Type
	Stmts     []Stmt
}

type Stmt struct {
	Type     StmtType
	Exprs    []Expr
	Blocks   []Block
	ForInit  int
	ForEnd   int
	ForOp    Op
	ForDelta int
}

type StmtType int

const (
	ExprStmt StmtType = iota
	BlockStmt
	Assign
	If
	For
	Continue
	Break
	Return
	Discard
)

type Expr struct {
	Type     ExprType
	Exprs    []Expr
	Variable Variable
	Int      int32
	Float    float32
	Ident    string
	Op       Op
}

type ExprType int

const (
	IntExpr ExprType = iota
	FloatExpr
	VarName
	Ident
	Unary
	Binary
	Selection
	Call
	FieldSelector
	Index
)

type Variable struct {
	Type  VariableType
	Index int
}

type VariableType int

const (
	Uniform VariableType = iota
	Local
)

type Op string

const (
	Add          Op = "+"
	Sub          Op = "-"
	Neg          Op = "!"
	Mul          Op = "*"
	Div          Op = "/"
	Mod          Op = "%"
	LeftShift    Op = "<<"
	RightShift   Op = ">>"
	LessThan     Op = "<"
	LessEqual    Op = "<="
	GreaterThan  Op = ">"
	GreaterEqual Op = ">="
	Equal        Op = "=="
	NotEqual     Op = "!="
	And          Op = "&"
	Xor          Op = "^"
	Or           Op = "|"
	AndAnd       Op = "&&"
	OrOr         Op = "||"
)
