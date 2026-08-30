// Copyright 2021 The Ebiten Authors
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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func isWSL() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}
	abs, err := filepath.Abs(".")
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(abs, `\\wsl$\`), nil
}

func TestPrograms(t *testing.T) {
	switch runtime.GOOS {
	case "android", "ios", "js":
		t.Skipf("process tests are not supported on %s", runtime.GOOS)
	}

	wsl, err := isWSL()
	if err != nil {
		t.Fatal(err)
	}
	if wsl {
		t.Skip("WSL doesn't support LockFileEx (#1864)")
	}

	dir := "testdata"
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	tmpdir := t.TempDir()

	type program struct {
		name string
		bin  string
	}
	programs := make([]program, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".go") {
			continue
		}
		programs = append(programs, program{name: n, bin: filepath.Join(tmpdir, n)})
	}

	errs := make([]error, len(programs))
	outs := make([][]byte, len(programs))

	build := func(i int) {
		if out, err := exec.Command("go", "build", "-o", programs[i].bin, filepath.Join(dir, programs[i].name)).CombinedOutput(); err != nil {
			errs[i] = err
			outs[i] = out
		}
	}

	// Build the first program serially so that the shared dependencies are
	// put into the build cache before the fan-out. Concurrent go build
	// invocations don't share in-flight compile actions, so a cold cache
	// would otherwise recompile the whole tree for each program.
	if len(programs) > 0 {
		build(0)
	}

	var eg errgroup.Group
	eg.SetLimit(runtime.NumCPU())
	for i := 1; i < len(programs); i++ {
		eg.Go(func() error {
			build(i)
			return nil
		})
	}
	_ = eg.Wait()

	// Run sub-tests one by one, not in parallel (#2571).
	var m sync.Mutex

	for i, p := range programs {
		t.Run(p.name, func(t *testing.T) {
			m.Lock()
			defer m.Unlock()

			if errs[i] != nil {
				t.Fatalf("%v\n%s", errs[i], outs[i])
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			cmd := exec.CommandContext(ctx, p.bin)
			stderr := &bytes.Buffer{}
			cmd.Stderr = stderr
			if err := cmd.Run(); err != nil {
				t.Errorf("%v\n%s", err, stderr)
			}
		})
	}
}
