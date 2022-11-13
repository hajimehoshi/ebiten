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
	"github.com/ebitengine/purego"
)

var (
	opengl uintptr
	isES   bool
)

func init() {
	opengl = purego.Dlopen("/System/Library/Frameworks/OpenGLES.framework/Versions/Current/OpenGLES", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if opengl != 0 {
		isES = true
		return
	}
	opengl = purego.Dlopen("/System/Library/Frameworks/OpenGL.framework/Versions/Current/OpenGL", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
}

func getProcAddress(name string) uintptr {
	return purego.Dlsym(opengl, name)
}
