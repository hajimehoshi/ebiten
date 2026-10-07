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

package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// unixSyscallCheckAnalyzer is an analyzer to check uses of Syscall, Syscall6, RawSyscall, and RawSyscall6 in golang.org/x/sys/unix.
// On linux/ppc64 and linux/ppc64le, they are plain Go functions, so a uintptr(unsafe.Pointer(...)) argument does not keep its referent
// alive and in place during the call. The syscall package's functions with the same names do on every architecture.
var unixSyscallCheckAnalyzer = &analysis.Analyzer{
	Name: "unixsyscallcheck",
	Doc:  "check uses of Syscall, Syscall6, RawSyscall, and RawSyscall6 in golang.org/x/sys/unix",
	Run:  runUnixSyscallCheck,
}

func runUnixSyscallCheck(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			fn, ok := pass.TypesInfo.Uses[id].(*types.Func)
			if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "golang.org/x/sys/unix" {
				return true
			}
			switch name := fn.Name(); name {
			case "Syscall", "Syscall6", "RawSyscall", "RawSyscall6":
				pass.Reportf(id.Pos(), "use syscall.%[1]s instead of unix.%[1]s: unix.%[1]s does not keep pointer arguments alive on linux/ppc64 and linux/ppc64le", name)
			}
			return true
		})
	}
	return nil, nil
}
