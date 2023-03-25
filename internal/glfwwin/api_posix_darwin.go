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

package glfwwin

import (
	"github.com/ebitengine/purego"
)

var (
	libSystem, _ = purego.Dlopen("libSystem.dylib", purego.RTLD_GLOBAL)
)

type pthread_key uint32

var pthread_key_create func(key *pthread_key, destructor uintptr) int32

var pthread_getspecific func(key pthread_key) uintptr

var pthread_key_delete func(key pthread_key) int32

func init() {
	purego.RegisterLibFunc(&pthread_getspecific, libSystem, "pthread_getspecific")
	purego.RegisterLibFunc(&pthread_key_create, libSystem, "pthread_key_create")
	purego.RegisterLibFunc(&pthread_key_delete, libSystem, "pthread_key_delete")
}
