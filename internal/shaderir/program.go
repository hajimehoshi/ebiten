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

type Func struct {
	Index       int
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

// FragmentFunc takes pseudo params, and the number is len(varyings) + 2.
// If index == len(varyings), the param represents the coordinate of the fragment (gl_FragCoord in GLSL).
// If index == len(varyings)+1, the param is an out-param representing the color of the pixel (gl_FragColor in GLSL).
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
	Type        ExprType
	Exprs       []Expr
	Int         int32
	Float       float32
	BuiltinFunc BuiltinFunc
	Swizzling   string
	Index       int
	Op          Op
}

type ExprType int

const (
	IntExpr ExprType = iota
	FloatExpr
	UniformVariable
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
	Neg                Op = "!"
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

type BuiltinFunc string

const (
	Vec2F            BuiltinFunc = "vec2"
	Vec3F            BuiltinFunc = "vec3"
	Vec4F            BuiltinFunc = "vec4"
	Mat2F            BuiltinFunc = "mat2"
	Mat3F            BuiltinFunc = "mat3"
	Mat4F            BuiltinFunc = "mat4"
	Radians          BuiltinFunc = "radians"
	Degrees          BuiltinFunc = "degrees"
	Sin              BuiltinFunc = "sin"
	Cos              BuiltinFunc = "cos"
	Tan              BuiltinFunc = "tan"
	Asin             BuiltinFunc = "asin"
	Acos             BuiltinFunc = "acos"
	Atan             BuiltinFunc = "atan"
	Pow              BuiltinFunc = "pow"
	Exp              BuiltinFunc = "exp"
	Log              BuiltinFunc = "log"
	Exp2             BuiltinFunc = "exp2"
	Log2             BuiltinFunc = "log2"
	Sqrt             BuiltinFunc = "sqrt"
	Inversesqrt      BuiltinFunc = "inversesqrt"
	Abs              BuiltinFunc = "abs"
	Sign             BuiltinFunc = "sign"
	Floor            BuiltinFunc = "floor"
	Ceil             BuiltinFunc = "ceil"
	Fract            BuiltinFunc = "fract"
	Mod              BuiltinFunc = "mod"
	Min              BuiltinFunc = "min"
	Max              BuiltinFunc = "max"
	Clamp            BuiltinFunc = "clamp"
	Mix              BuiltinFunc = "mix"
	Step             BuiltinFunc = "step"
	Smoothstep       BuiltinFunc = "smoothstep"
	Length           BuiltinFunc = "length"
	Distance         BuiltinFunc = "distance"
	Dot              BuiltinFunc = "dot"
	Cross            BuiltinFunc = "cross"
	Normalize        BuiltinFunc = "normalize"
	Faceforward      BuiltinFunc = "faceforward"
	Reflect          BuiltinFunc = "reflect"
	MatrixCompMult   BuiltinFunc = "matrixCompMult"
	OuterProduct     BuiltinFunc = "outerProduct"
	Transpose        BuiltinFunc = "transpose"
	LessThan         BuiltinFunc = "lessThan"
	LessThanEqual    BuiltinFunc = "lessThanEqual"
	GreaterThan      BuiltinFunc = "greaterThan"
	GreaterThanEqual BuiltinFunc = "greaterThanEqual"
	Equal            BuiltinFunc = "equal"
	NotEqual         BuiltinFunc = "notEqual"
	Any              BuiltinFunc = "any"
	All              BuiltinFunc = "all"
	Not              BuiltinFunc = "not"
	Texture2DF       BuiltinFunc = "texture2D"
)

func ParseBuiltinFunc(str string) (BuiltinFunc, bool) {
	switch BuiltinFunc(str) {
	case Vec2F,
		Vec3F,
		Vec4F,
		Mat2F,
		Mat3F,
		Mat4F,
		Radians,
		Degrees,
		Sin,
		Cos,
		Tan,
		Asin,
		Acos,
		Atan,
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
		MatrixCompMult,
		OuterProduct,
		Transpose,
		LessThan,
		LessThanEqual,
		GreaterThan,
		GreaterThanEqual,
		Equal,
		NotEqual,
		Any,
		All,
		Not,
		Texture2DF:
		return BuiltinFunc(str), true
	}
	return "", false
}
