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
	"strings"

	"github.com/ebitengine/purego"
)

var (
	opengl uintptr
)

func (c *defaultContext) init() error {
	lib, errGLES := purego.Dlopen("OpenGLES.framework/OpenGLES", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if errGLES == nil {
		c.isES = true
		opengl = lib
		return nil
	}

	lib, errGL := purego.Dlopen("OpenGL.framework/OpenGL", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if errGL == nil {
		opengl = lib
		return nil
	}

	// TODO: Use multiple %w-s as of Go 1.20
	return fmt.Errorf("gl: failed to load: OpenGL.framework: %v, OpenGLES.framework: %v", errGL, errGLES)
}

func (c *defaultContext) getProcAddress(name string) uintptr {
	if c.isES {
		name = strings.TrimSuffix(name, "EXT")
	}
	proc, err := purego.Dlsym(opengl, name)
	if err != nil {
		// The proc is not found.
		return 0
	}
	return proc
}
