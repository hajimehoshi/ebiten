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

//go:build ebitenexternaldll
// +build ebitenexternaldll

package glfw

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
)

func loadDLL() (*dll, error) {
	name := "glfw_windows_" + runtime.GOARCH + ".dll"
	d := windows.NewLazyDLL(name)
	if err := d.Load(); err != nil {
		return nil, fmt.Errorf("glfw: failed to load %s: %w", name, err)
	}
	return &dll{
		d: d,
	}, nil
}
