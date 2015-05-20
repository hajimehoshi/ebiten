// Copyright 2015 Hajime Hoshi
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

package internal

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

func LicenseComment() (string, error) {
	_, path, _, _ := runtime.Caller(0)
	licensePath := filepath.Join(filepath.Dir(path), "..", "LICENSE")
	l, err := ioutil.ReadFile(licensePath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(l), "\n")
	license := "// " + strings.Join(lines[:len(lines)-1], "\n// ")
	return license, nil
}
