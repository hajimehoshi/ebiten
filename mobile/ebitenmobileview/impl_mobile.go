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

package ebitenmobileview

// #cgo ios LDFLAGS: -framework UIKit -framework GLKit -framework QuartzCore -framework OpenGLES
//
// #include <stdint.h>
import "C"

import (
	"errors"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/uidriver/mobile"
)

var (
	chError <-chan error
	running bool
)

func update() error {
	if chError == nil {
		return errors.New("ebitenmobileview: chError must not be nil: Start is not called yet?")
	}
	if !running {
		return errors.New("ebitenmobileview: start must be called ahead of update")
	}

	select {
	case err := <-chError:
		return err
	default:
	}

	mobile.Get().Render()
	return nil
}

func start(f func(*ebiten.Image) error, width, height int, scale float64) {
	running = true
	// The last argument 'title' is not used on mobile platforms, so just pass an empty string.
	chError = ebiten.RunWithoutMainLoop(f, width, height, scale, "")
}

func setScreenSize(width, height int, scale float64) {
	ebiten.SetScreenSize(width, height)
	ebiten.SetScreenScale(scale)
}
