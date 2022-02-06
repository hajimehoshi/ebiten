// Copyright 2019 The Ebiten Authors
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

package driver

import (
	"image"
)

type Window interface {
	IsDecorated() bool
	SetDecorated(decorated bool)

	IsResizable() bool
	SetResizable(resizable bool)

	Position() (int, int)
	SetPosition(x, y int)

	Size() (int, int)
	SetSize(width, height int)
	SizeLimits() (minw, minh, maxw, maxh int)
	SetSizeLimits(minw, minh, maxw, maxh int)

	IsFloating() bool
	SetFloating(floating bool)

	Maximize()
	IsMaximized() bool

	Minimize()
	IsMinimized() bool

	SetIcon(iconImages []image.Image)
	SetTitle(title string)
	Restore()

	IsBeingClosed() bool
	SetClosingHandled(handled bool)
	IsClosingHandled() bool
}
