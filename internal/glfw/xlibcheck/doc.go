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

// Package xlibcheck hosts a cgo test that cross-checks internal/glfw's Go
// mirrors of the Xlib structs against the real C headers. It lives in its own
// leaf package because the go tool rejects cgo in a test of a package that is
// imported elsewhere in the build ("use of cgo in test not supported"), which
// internal/glfw is.
//
// The whole check is guarded by the ebitenginexlibcheck build tag. Ordinary
// builds do not set it, so this package is empty for them and needs no X11
// development headers; only the dedicated CI step that sets the tag compiles
// the check. The same tag exposes the alias bridge to the unexported Xlib
// mirror types in internal/glfw, so internal/glfw keeps those types out of its
// ordinary (untagged) build.
package xlibcheck
