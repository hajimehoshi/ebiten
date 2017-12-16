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

// +build darwin freebsd linux windows
// +build !js
// +build !android
// +build !ios

package ebitenutil

import (
	"os"
	"path/filepath"
)

// OpenFile opens a file and returns a stream for its data.
//
// How to solve path depends on your environment. This varies on your desktop or web browser.
// Note that this doesn't work on mobiles.
func OpenFile(path string) (ReadSeekCloser, error) {
	return os.Open(path)
}

// JoinStringsIntoFilePath joins any number of path elements into a single path,
// adding a Separator if necessary.
//
// This is basically same as filepath.Join, but the behavior is different on JavaScript.
// For browsers, path.Join is called instead of filepath.Join.
func JoinStringsIntoFilePath(elems ...string) string {
	return filepath.Join(elems...)
}
