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

// +build android ios

package mobile

import (
	"errors"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

var (
	chError <-chan error
	running bool
)

func update() error {
	if chError == nil {
		return errors.New("mobile: chError must not be nil: Start is not called yet?")
	}
	if !running {
		return errors.New("mobile: start must be called ahead of update")
	}
	return ui.Render(chError)
}

func start(f func(*ebiten.Image) error, width, height int, scale float64, title string) {
	running = true
	chError = ebiten.RunWithoutMainLoop(f, width, height, scale, title)
}
