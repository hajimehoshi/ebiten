// Copyright 2022 The Ebitengine Authors
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

package main

import (
	"fmt"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// imageImportCheckAnalyzer is an analyzer to check whether unexpected `image/*` packages are imported.
// Importing `image/gif`, `image/jpeg`, and `image/png` registers their recorders at `init` functions, so
// it affects the result of `image.Decode`. Ebitengine should not have such side-effects.
var imageImportCheckAnalyzer = &analysis.Analyzer{
	Name:       "imageimportcheck",
	Doc:        "check importing image/gif, image/jpeg, and image/png packages",
	Run:        runImageImportCheck,
	ResultType: reflect.TypeOf(imageImportCheckResult{}),
}

type imageImportCheckResult struct {
	Errors []imageImportCheckError
}

type imageImportCheckError struct {
	Pos    token.Pos
	Import string
}

func runImageImportCheck(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	if strings.HasPrefix(pkgPath, "github.com/hajimehoshi/ebiten/v2/examples/") {
		return imageImportCheckResult{}, nil
	}
	if strings.HasSuffix(pkgPath, "_test") {
		return imageImportCheckResult{}, nil
	}

	// TODO: Remove this exception after v3 is released (#2336).
	if pkgPath == "github.com/hajimehoshi/ebiten/v2/ebitenutil" {
		return imageImportCheckResult{}, nil
	}

	var errs []imageImportCheckError
	for _, f := range pass.Files {
		for _, i := range f.Imports {
			path, err := strconv.Unquote(i.Path.Value)
			if err != nil {
				return imageImportCheckResult{}, err
			}
			if path == "image/gif" || path == "image/jpeg" || path == "image/png" {
				err := imageImportCheckError{
					Pos:    pass.Fset.File(f.Pos()).Pos(int(i.Pos())),
					Import: path,
				}
				errs = append(errs, err)
				pass.Report(analysis.Diagnostic{
					Pos:     err.Pos,
					Message: fmt.Sprintf("unexpected import %q", err.Import),
				})
			}
		}
	}
	return imageImportCheckResult{
		Errors: errs,
	}, nil
}
