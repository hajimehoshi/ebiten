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

package gl

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	opengl32              = windows.NewLazySystemDLL("opengl32")
	procWglGetProcAddress = opengl32.NewProc("wglGetProcAddress")
)

func (c *defaultContext) init() error {
	return nil
}

func (c *defaultContext) getProcAddress(namea string) (uintptr, error) {
	cname, err := windows.BytePtrFromString(namea)
	if err != nil {
		return 0, err
	}

	r, _, err := procWglGetProcAddress.Call(uintptr(unsafe.Pointer(cname)))
	if r != 0 {
		return r, nil
	}
	if err != nil && err != windows.ERROR_SUCCESS && err != windows.ERROR_PROC_NOT_FOUND {
		return 0, fmt.Errorf("gl: wglGetProcAddress failed for %s: %w", namea, err)
	}

	p := opengl32.NewProc(namea)
	if err := p.Find(); err != nil {
		return 0, err
	}
	return p.Addr(), nil
}
