// Copyright 2016 Hajime Hoshi
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

//go:build !android && !ios && !js

package ebitenutil

import (
	"os"
	"path/filepath"
)

// OpenFile opens a file and returns a stream for its data.
//
// The path parts should be separated with slash '/' on any environments.
//
// OpenFile doesn't work on mobiles.
//
// Deprecated: as of v2.4. Use os.Open on desktops and http.Get on browsers instead.
func OpenFile(path string) (ReadSeekCloser, error) {
	return os.Open(filepath.FromSlash(path))
}
