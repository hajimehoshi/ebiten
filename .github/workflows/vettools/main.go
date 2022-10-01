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

package main

import (
	"os"

	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/copylock"
)

func main() {
	const filename = ".errcheck_excludes"
	if _, err := os.Stat(filename); err == nil {
		errcheck.Analyzer.Flags.Set("exclude", filename)
	}
	multichecker.Main(atomic.Analyzer,
		atomicalign.Analyzer,
		copylock.Analyzer,
		errcheck.Analyzer,
		imageImportCheckAnalyzer)
}
