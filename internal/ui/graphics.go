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
	"os"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type graphicsDriverCreator interface {
	newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error)
	newOpenGL() (graphicsdriver.Graphics, error)
	newDirectX() (graphicsdriver.Graphics, error)
	newMetal() (graphicsdriver.Graphics, error)
}

func newGraphicsDriver(creator graphicsDriverCreator) (graphicsdriver.Graphics, error) {
	envName := "EBITENGINE_GRAPHICS_LIBRARY"
	env := os.Getenv(envName)
	if env == "" {
		// For backward compatibility, read the EBITEN_ version.
		envName = "EBITEN_GRAPHICS_LIBRARY"
		env = os.Getenv(envName)
	}

	switch env {
	case "", "auto":
		g, lib, err := creator.newAuto()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: no graphics library is available")
		}
		theGlobalState.setGraphicsLibrary(lib)
		return g, nil
	case "opengl":
		g, err := creator.newOpenGL()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: %s=%s is specified but OpenGL is not available", envName, env)
		}
		theGlobalState.setGraphicsLibrary(GraphicsLibraryOpenGL)
		return g, nil
	case "directx":
		g, err := creator.newDirectX()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: %s=%s is specified but DirectX is not available.", envName, env)
		}
		theGlobalState.setGraphicsLibrary(GraphicsLibraryDirectX)
		return g, nil
	case "metal":
		g, err := creator.newMetal()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: %s=%s is specified but Metal is not available", envName, env)
		}
		theGlobalState.setGraphicsLibrary(GraphicsLibraryMetal)
		return g, nil
	default:
		return nil, fmt.Errorf("ui: an unsupported graphics library is specified: %s", env)
	}
}

func GraphicsDriverForTesting() graphicsdriver.Graphics {
	return theUI.graphicsDriver
}
