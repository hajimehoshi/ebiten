// Copyright 2022 The Ebiten Authors
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

package graphicscommand

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type Drawer interface {
	Draw(screenScale float64, offsetX, offsetY float64, needsClearingScreen bool, framebufferYDirection graphicsdriver.YDirection, screenClearedEveryFrame, filterEnabled bool) error
}

func Draw(drawer Drawer, screenScale float64, offsetX, offsetY float64, screenClearedEveryFrame, filterEnabled bool) error {
	return drawer.Draw(screenScale, offsetX, offsetY, graphicsDriver().NeedsClearingScreen(), graphicsDriver().FramebufferYDirection(), screenClearedEveryFrame, filterEnabled)
}

// TODO: Reduce these 'getter' global functions if possible.

func NeedsInvertY() bool {
	return graphicsDriver().FramebufferYDirection() != graphicsDriver().NDCYDirection()
}

func NeedsRestoring() bool {
	return graphicsDriver().NeedsRestoring()
}

func IsGL() bool {
	return graphicsDriver().IsGL()
}

func SetVsyncEnabled(enabled bool) {
	graphicsDriver().SetVsyncEnabled(enabled)
}

func SetTransparent(transparent bool) {
	graphicsDriver().SetTransparent(transparent)
}

func SetFullscreen(fullscreen bool) {
	graphicsDriver().SetFullscreen(fullscreen)
}

func SetWindow(window uintptr) {
	if g, ok := graphicsDriver().(interface{ SetWindow(uintptr) }); ok {
		g.SetWindow(window)
	}
}
