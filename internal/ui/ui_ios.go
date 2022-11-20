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

package ui

import (
	"fmt"

	"golang.org/x/mobile/gl"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverCreatorImpl struct {
	gomobileContext gl.Context
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	m, err1 := g.newMetal()
	if err1 == nil {
		return m, GraphicsLibraryMetal, nil
	}
	o, err2 := g.newOpenGL()
	if err2 == nil {
		return o, GraphicsLibraryMetal, nil
	}
	return nil, GraphicsLibraryUnknown, fmt.Errorf("ui: failed to choose graphics drivers: Metal: %v, OpenGL: %v", err1, err2)
}

func (g *graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics(g.gomobileContext)
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, nil
}

func (g *graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	if g.gomobileContext != nil {
		return nil, fmt.Errorf("ui: Metal is not available with gomobile-build")
	}
	return metal.NewGraphics()
}

func SetUIView(uiview uintptr) error {
	return theUI.setUIView(uiview)
}

func IsGL() (bool, error) {
	return theUI.isGL()
}

func (u *userInterfaceImpl) setUIView(uiview uintptr) error {
	select {
	case err := <-u.errCh:
		return err
	case <-u.graphicsDriverInitCh:
	}

	// This function should be called only when the graphics library is Metal.
	if g, ok := u.graphicsDriver.(interface{ SetUIView(uintptr) }); ok {
		g.SetUIView(uiview)
	}
	return nil
}

func (u *userInterfaceImpl) isGL() (bool, error) {
	select {
	case err := <-u.errCh:
		return false, err
	case <-u.graphicsDriverInitCh:
	}

	return u.graphicsDriver.IsGL(), nil
}
