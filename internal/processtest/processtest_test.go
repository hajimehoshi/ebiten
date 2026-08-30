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

	type buildResult struct {
		name string
		bin  string
		err  error
		out  []byte
	}
	results := make([]buildResult, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".go") {
			continue
		}
		results = append(results, buildResult{name: n, bin: filepath.Join(tmpdir, n)})
	}

	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if out, err := exec.Command("go", "build", "-o", results[i].bin, filepath.Join(dir, results[i].name)).CombinedOutput(); err != nil {
				results[i].err = err
				results[i].out = out
			}
		}(i)
	}
	wg.Wait()
	for _, r := range results {
		if r.err != nil {
			t.Fatalf("%v\n%s", r.err, r.out)
		}
	}

	// Run sub-tests one by one, not in parallel (#2571).
	var m sync.Mutex

	for _, r := range results {
		r := r
		t.Run(r.name, func(t *testing.T) {
			m.Lock()
			defer m.Unlock()

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			cmd := exec.CommandContext(ctx, r.bin)
			stderr := &bytes.Buffer{}
			cmd.Stderr = stderr
			if err := cmd.Run(); err != nil {
				t.Errorf("%v\n%s", err, stderr)
			}
		})
	}
}