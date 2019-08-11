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

// +build darwin freebsd js linux windows
// +build !android
// +build !ios

package ebitenmobileview

import (
	"github.com/hajimehoshi/ebiten"
)

// Empty implementation of this package.
// Package mobile is buildable for non-mobile platforms so that godoc can show comments.

func update() error {
	return nil
}

func start(f func(*ebiten.Image) error, width, height int, scale float64, title string) {
}

func updateTouchesOnAndroid(action int, id int, x, y int) {
}

func updateTouchesOnIOSImpl(phase int, ptr int64, x, y int) {
}
