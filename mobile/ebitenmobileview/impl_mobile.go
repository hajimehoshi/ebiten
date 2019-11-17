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
	"github.com/hajimehoshi/ebiten/internal/uidriver/mobile"
)

func update() error {
	if !theState.isRunning() {
		// start is not called yet, but as update can be called from another thread, it is OK. Just ignore
		// this.
		return nil
	}

	select {
	case err := <-theState.errorCh:
		return err
	default:
	}

	mobile.Get().Update()
	return nil
}
