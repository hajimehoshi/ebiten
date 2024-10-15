// Copyright 2024 The Ebitengine Authors
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

package debug

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	ebitengineFileDir     string
	ebitengineFileDirOnce sync.Once
)

// FirstCaller returns the file and line number of the first caller outside of Ebitengine.
func FirstCaller() (file string, line int, ok bool) {
	ebitengineFileDirOnce.Do(func() {
		cmd := exec.Command("go", "list", "-f", "{{.Dir}}", "github.com/hajimehoshi/ebiten/v2")
		out, err := cmd.Output()
		if errors.Is(err, exec.ErrNotFound) {
			return
		}
		if err != nil {
			panic(fmt.Sprintf("debug: go list -f {{.Dir}} failed: %v", err))
		}
		ebitengineFileDir = filepath.ToSlash(strings.TrimSpace(string(out)))
	})

	// Go command is not found.
	if ebitengineFileDir == "" {
		return "", 0, false
	}

	// Relying on a caller stacktrace is very fragile, but this is fine as this is only for debugging.
	var ebitenPackageReached bool
	for i := 0; ; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// The file should be with a slash, but just in case, convert it.
		file = filepath.ToSlash(file)

		if !ebitenPackageReached {
			if path.Dir(file) == ebitengineFileDir {
				ebitenPackageReached = true
			}
			continue
		}

		if path.Dir(file) == ebitengineFileDir || path.Dir(file) == ebitengineFileDir+"/colorm" || path.Dir(file) == ebitengineFileDir+"/text" || path.Dir(file) == ebitengineFileDir+"/text/v2" {
			continue
		}
		return file, line, true
	}

	return "", 0, false
}
