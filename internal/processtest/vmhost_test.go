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

package processtest_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestVMHostThreadMode(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("VM hosts require a desktop")
	}
	for _, tag := range []string{"", "ebitenginesinglethread", "ebitensinglethread"} {
		t.Run("tag="+tag, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "host.exe")
			build := exec.Command("go", "build", "-buildvcs=false", "-tags", tag, "-o", binary, "./testdata/vmhostthreadmode")
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			modes := []string{"before", "during"}
			if tag != "" {
				modes = []string{"tag-before"}
			}
			for _, mode := range modes {
				t.Run(mode, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					cmd := exec.CommandContext(ctx, binary, mode)
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Errorf("fixture: %v\n%s", err, output)
					}
				})
			}
		})
	}
}
