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

//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const oboeVersion = "1.6.0"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := clean(); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "oboe-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := prepareOboe(tmp); err != nil {
		return err
	}

	return nil
}

func clean() error {
	fmt.Printf("Cleaning *.cpp and *.h files\n")
	if err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != "." {
			return filepath.SkipDir
		}
		base := filepath.Base(path)
		if base == "README-oboe.md" {
			return os.Remove(base)
		}
		if base == "LICENSE-oboe" {
			return os.Remove(base)
		}
		if !strings.HasPrefix(base, "oboe_") {
			return nil
		}
		if !strings.HasSuffix(base, ".cpp") && !strings.HasSuffix(base, ".h") {
			return nil
		}
		return os.Remove(base)
	}); err != nil {
		return err
	}
	return nil
}

func prepareOboe(tmp string) error {
	fn := oboeVersion + ".tar.gz"
	if e, err := exists(fn); err != nil {
		return err
	} else if !e {
		url := "https://github.com/google/oboe/archive/refs/tags/" + fn
		fmt.Fprintf(os.Stderr, "%s not found: please download it from %s\n", fn, url)
		return nil
	}

	fmt.Printf("Copying %s to %s\n", fn, filepath.Join(tmp, fn))
	in, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(filepath.Join(tmp, fn))
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	fmt.Printf("Extracting %s\n", fn)
	cmd := exec.Command("tar", "-xzf", fn)
	cmd.Stderr = os.Stderr
	cmd.Dir = tmp
	if err := cmd.Run(); err != nil {
		return err
	}

	reInclude := regexp.MustCompile(`(?m)^#include\s+([<"])(.+)[>"]$`)

	fmt.Printf("Copying *.cpp and *.h files\n")
	for _, dir := range []string{"src", "include"} {
		dir := dir
		indir := filepath.Join(tmp, "oboe-"+oboeVersion, dir)
		if err := filepath.Walk(indir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".cpp") && !strings.HasSuffix(path, ".h") {
				return nil
			}

			f, err := filepath.Rel(indir, path)
			if err != nil {
				return err
			}
			ext := filepath.Ext(f)
			curTs := strings.Split(f[:len(f)-len(ext)], string(filepath.Separator))
			outfn := "oboe_" + strings.Join(curTs, "_") + "_android" + ext

			if _, err := os.Stat(outfn); err == nil {
				return fmt.Errorf("%s must not exist", outfn)
			}

			in, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Replace #include paths.
			in = reInclude.ReplaceAllFunc(in, func(inc []byte) []byte {
				m := reInclude.FindSubmatch(inc)
				f := string(m[2])

				searchDirs := []string{filepath.Dir(path)}
				if dir == "src" {
					searchDirs = append(searchDirs, filepath.Join(tmp, "oboe-"+oboeVersion, "src"))
				}
				searchDirs = append(searchDirs, filepath.Join(tmp, "oboe-"+oboeVersion, "include"))
				for _, searchDir := range searchDirs {
					path := filepath.Join(searchDir, f)
					e, err := exists(path)
					if err != nil {
						panic(err)
					}
					if !e {
						continue
					}

					f, err := filepath.Rel(filepath.Join(tmp, "oboe-"+oboeVersion), path)
					if err != nil {
						panic(err)
					}
					ext := filepath.Ext(f)
					ts := strings.Split(f[:len(f)-len(ext)], string(filepath.Separator))
					// The first token is 'src' or 'include'. Remove this.
					ts = ts[1:]
					newpath := "oboe_" + strings.Join(ts, "_") + "_android" + ext
					return []byte(`#include "` + newpath + `"`)
				}
				return inc
			})

			out, err := os.Create(outfn)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, bytes.NewReader(in)); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return err
		}
	}

	fmt.Printf("Copying README.md and LICENSE\n")
	for _, f := range []string{"README.md", "LICENSE"} {
		infn := filepath.Join(tmp, "oboe-"+oboeVersion, f)

		ext := filepath.Ext(f)
		outfn := f[:len(f)-len(ext)] + "-oboe" + ext

		in, err := os.Open(infn)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(outfn)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}
	}

	return nil
}

func exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
