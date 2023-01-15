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

func newGraphicsDriver(creator graphicsDriverCreator, graphicsLibrary GraphicsLibrary) (graphicsdriver.Graphics, error) {
	if graphicsLibrary == GraphicsLibraryAuto {
		envName := "EBITENGINE_GRAPHICS_LIBRARY"
		env := os.Getenv(envName)
		if env == "" {
			// For backward compatibility, read the EBITEN_ version.
			envName = "EBITEN_GRAPHICS_LIBRARY"
			env = os.Getenv(envName)
		}

		switch env {
		case "", "auto":
			// Keep the automatic choosing.
		case "opengl":
			graphicsLibrary = GraphicsLibraryOpenGL
		case "directx":
			graphicsLibrary = GraphicsLibraryDirectX
		case "metal":
			graphicsLibrary = GraphicsLibraryMetal
		default:
			return nil, fmt.Errorf("ui: an unsupported graphics library is specified by the environment variable: %s", env)
		}
	}

	switch graphicsLibrary {
	case GraphicsLibraryAuto:
		g, lib, err := creator.newAuto()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: no graphics library is available")
		}
		theGlobalState.setGraphicsLibrary(lib)
		return g, nil
	case GraphicsLibraryOpenGL:
		g, err := creator.newOpenGL()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: %s is specified but OpenGL is not available", graphicsLibrary)
		}
		theGlobalState.setGraphicsLibrary(GraphicsLibraryOpenGL)
		return g, nil
	case GraphicsLibraryDirectX:
		g, err := creator.newDirectX()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: %s is specified but DirectX is not available.", graphicsLibrary)
		}
		theGlobalState.setGraphicsLibrary(GraphicsLibraryDirectX)
		return g, nil
	case GraphicsLibraryMetal:
		g, err := creator.newMetal()
		if err != nil {
			return nil, err
		}
		if g == nil {
			return nil, fmt.Errorf("ui: %s is specified but Metal is not available", graphicsLibrary)
		}
		theGlobalState.setGraphicsLibrary(GraphicsLibraryMetal)
		return g, nil
	default:
		return nil, fmt.Errorf("ui: an unsupported graphics library is specified: %d", graphicsLibrary)
	}
}

func GraphicsDriverForTesting() graphicsdriver.Graphics {
	return theUI.graphicsDriver
}

type GraphicsLibrary int

const (
	GraphicsLibraryAuto GraphicsLibrary = iota
	GraphicsLibraryOpenGL
	GraphicsLibraryDirectX
	GraphicsLibraryMetal
	GraphicsLibraryUnknown
)

func (g GraphicsLibrary) String() string {
	switch g {
	case GraphicsLibraryAuto:
		return "Auto"
	case GraphicsLibraryOpenGL:
		return "OpenGL"
	case GraphicsLibraryDirectX:
		return "DirectX"
	case GraphicsLibraryMetal:
		return "Metal"
	case GraphicsLibraryUnknown:
		return "Unknown"
	default:
		return fmt.Sprintf("GraphicsLibrary(%d)", g)
	}
}
