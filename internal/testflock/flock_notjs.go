// Copyright 2018 The Ebiten Authors
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

// +build !js

// Package testflock provides a lock for testing.
//
// There is a CI service like TravisCI where multiple OpenGL processes
// don't work well at the same time, and tests with an OpenGL main routine
// should be protected by a file lock (#575).
package testflock

import (
	"os"
	"path/filepath"

	"github.com/theckman/go-flock"
)

var theLock = flock.NewFlock(filepath.Join(os.TempDir(), "ebitentest"))

func Lock() {
	if err := theLock.Lock(); err != nil {
		panic(err)
	}
}

func Unlock() {
	if err := theLock.Unlock(); err != nil {
		panic(err)
	}
}
