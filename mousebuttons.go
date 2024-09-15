// Copyright 2015 Hajime Hoshi
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// A MouseButton represents a mouse button.
type MouseButton int

// MouseButtons
const (
	MouseButtonLeft   MouseButton = MouseButton0
	MouseButtonMiddle MouseButton = MouseButton1
	MouseButtonRight  MouseButton = MouseButton2

	MouseButton0   MouseButton = MouseButton(ui.MouseButton0)
	MouseButton1   MouseButton = MouseButton(ui.MouseButton1)
	MouseButton2   MouseButton = MouseButton(ui.MouseButton2)
	MouseButton3   MouseButton = MouseButton(ui.MouseButton3)
	MouseButton4   MouseButton = MouseButton(ui.MouseButton4)
	MouseButtonMax MouseButton = MouseButton4
)
