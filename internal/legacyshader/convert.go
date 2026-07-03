// Copyright 2026 The Ebitengine Authors
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

package legacyshader

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
)

// texelHelperPrefix prefixes the helper functions emulating the texel-unit behavior of the builtin
// functions on top of their pixel-unit definitions.
const texelHelperPrefix = "__legacyshader_"

// texelBuiltinNames is the set of the builtin function names whose behavior depends on the unit.
var texelBuiltinNames = func() map[string]struct{} {
	names := map[string]struct{}{
		"imageDstOrigin":          {},
		"imageDstSize":            {},
		"imageDstRegionOnTexture": {},
		"imageSrcRegionOnTexture": {},
	}
	for i := range graphics.ShaderSrcImageCount {
		for _, f := range []string{"Origin", "Size", "UnsafeAt", "At"} {
			names[fmt.Sprintf("imageSrc%d%s", i, f)] = struct{}{}
		}
	}
	return names
}()

// texelHelpersSuffix is the Kage source defining the texel-unit helper functions. A helper wraps the
// pixel-unit builtin function of the same base name, converting the argument or the results between
// texels and pixels with the texture-size uniforms.
var texelHelpersSuffix = func() string {
	var b strings.Builder
	b.WriteString(`
func __legacyshader_imageDstOrigin() vec2 {
	return imageDstOrigin() / __imageDstTextureSize
}

func __legacyshader_imageDstSize() vec2 {
	return imageDstSize() / __imageDstTextureSize
}

func __legacyshader_imageDstRegionOnTexture() (vec2, vec2) {
	return __legacyshader_imageDstOrigin(), __legacyshader_imageDstSize()
}

func __legacyshader_imageSrcRegionOnTexture() (vec2, vec2) {
	return __legacyshader_imageSrc0Origin(), __legacyshader_imageSrc0Size()
}
`)
	for i := range graphics.ShaderSrcImageCount {
		// A source texture size can be zero when no source image is given. Guard against a division by
		// zero with max so that a zero region stays zero, matching the case without a source image.
		b.WriteString(fmt.Sprintf(`
func __legacyshader_imageSrc%[1]dOrigin() vec2 {
	return imageSrc%[1]dOrigin() / max(__imageSrcTextureSizes[%[1]d], vec2(1))
}

func __legacyshader_imageSrc%[1]dSize() vec2 {
	return imageSrc%[1]dSize() / max(__imageSrcTextureSizes[%[1]d], vec2(1))
}

func __legacyshader_imageSrc%[1]dUnsafeAt(pos vec2) vec4 {
	// The argument is in texels of the 0th texture. Convert it to pixels of the 0th texture.
	return imageSrc%[1]dUnsafeAt(pos * __imageSrcTextureSizes[0])
}

func __legacyshader_imageSrc%[1]dAt(pos vec2) vec4 {
	// The argument is in texels of the 0th texture. Convert it to pixels of the 0th texture.
	return imageSrc%[1]dAt(pos * __imageSrcTextureSizes[0])
}
`, i))
	}
	return b.String()
}()

// convertToPixels converts a texel-unit fragment shader source to an equivalent pixel-unit source.
//
// The conversion redirects the unit-dependent builtin calls to the texel-unit helper functions, converts
// the fragment entry point's source position from pixels to texels, and appends the helper definitions.
// Comments, including the //kage:unit directive, are dropped, and a //kage:unit pixels directive is
// prepended.
func convertToPixels(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fun := call.Fun
		for {
			p, ok := fun.(*ast.ParenExpr)
			if !ok {
				break
			}
			fun = p.X
		}
		id, ok := fun.(*ast.Ident)
		if !ok {
			return true
		}
		if _, ok := texelBuiltinNames[id.Name]; ok {
			id.Name = texelHelperPrefix + id.Name
		}
		return true
	})

	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Name.Name != "Fragment" || fd.Body == nil || fd.Type.Params == nil {
			continue
		}
		// The second parameter is the source position. An entry point's parameter cannot be assigned,
		// so rename the parameter and redeclare a local variable with the original name holding the
		// converted position. The local variable is always used by the appended blank assignment, in
		// case the original parameter is unused.
		var idx int
		var srcPosIdent *ast.Ident
		for _, field := range fd.Type.Params.List {
			if len(field.Names) == 0 {
				idx++
				continue
			}
			for _, n := range field.Names {
				if idx == 1 {
					srcPosIdent = n
				}
				idx++
			}
		}
		if srcPosIdent == nil || srcPosIdent.Name == "_" {
			continue
		}
		orig := srcPosIdent.Name
		srcPosIdent.Name = texelHelperPrefix + "srcPos"
		fd.Body.List = append([]ast.Stmt{
			srcPosConversionStmt(orig, srcPosIdent.Name),
			blankUseStmt(orig),
		}, fd.Body.List...)
	}

	var buf bytes.Buffer
	buf.WriteString("//kage:unit pixels\n\n")
	if err := printer.Fprint(&buf, fset, f); err != nil {
		return nil, err
	}
	buf.WriteString(texelHelpersSuffix)
	return buf.Bytes(), nil
}

// srcPosConversionStmt returns the statement `name := param / max(__imageSrcTextureSizes[0], vec2(1))`,
// converting the fragment entry point's source position from pixels to texels. The texture size is
// guarded against zero (no source image) to avoid a division by zero.
func srcPosConversionStmt(name, param string) ast.Stmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent(name)},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.BinaryExpr{
				X:  ast.NewIdent(param),
				Op: token.QUO,
				Y: &ast.CallExpr{
					Fun: ast.NewIdent("max"),
					Args: []ast.Expr{
						&ast.IndexExpr{
							X:     ast.NewIdent("__imageSrcTextureSizes"),
							Index: &ast.BasicLit{Kind: token.INT, Value: "0"},
						},
						&ast.CallExpr{
							Fun:  ast.NewIdent("vec2"),
							Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}},
						},
					},
				},
			},
		},
	}
}

// blankUseStmt returns the statement `_ = name`.
func blankUseStmt(name string) ast.Stmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("_")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{ast.NewIdent(name)},
	}
}
