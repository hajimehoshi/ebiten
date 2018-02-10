// Copyright 2014 Hajime Hoshi
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

package ebitenutil

import (
	"io"

	"github.com/hajimehoshi/ebiten"
)

// RecordScreenAsGIF is deprecated as of version 1.6.0-alpha.
func RecordScreenAsGIF(update func(*ebiten.Image) error, out io.Writer, frameNum int) func(*ebiten.Image) error {
	panic("ebitenutil: RecordScreenAsGIF is no longer implemented")
}
