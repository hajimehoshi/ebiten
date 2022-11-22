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
type MouseButton = ui.MouseButton

// MouseButtons
const (
	MouseButtonLeft   MouseButton = ui.MouseButtonLeft
	MouseButtonRight  MouseButton = ui.MouseButtonRight
	MouseButtonMiddle MouseButton = ui.MouseButtonMiddle
	MouseButton4      MouseButton = ui.MouseButton4
	MouseButton5      MouseButton = ui.MouseButton5
	MouseButton6      MouseButton = ui.MouseButton6
	MouseButton7      MouseButton = ui.MouseButton7
	MouseButton8      MouseButton = ui.MouseButton8
	MouseButtonMax    MouseButton = MouseButton8
)
