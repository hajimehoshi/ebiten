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

type graphicsDriverGetter interface {
	getAuto() graphicsdriver.Graphics
	getOpenGL() graphicsdriver.Graphics
	getDirectX() graphicsdriver.Graphics
	getMetal() graphicsdriver.Graphics
}

func chooseGraphicsDriver(getter graphicsDriverGetter) (graphicsdriver.Graphics, error) {
	const envName = "EBITEN_GRAPHICS_LIBRARY"

	switch env := os.Getenv(envName); env {
	case "", "auto":
		if g := getter.getAuto(); g != nil {
			return g, nil
		}
		return nil, fmt.Errorf("ui: no graphics library is available")
	case "opengl":
		if g := getter.getOpenGL(); g != nil {
			return g, nil
		}
		return nil, fmt.Errorf("ui: %s=%s is specified but OpenGL is not available", envName, env)
	case "directx":
		if g := getter.getDirectX(); g != nil {
			return g, nil
		}
		return nil, fmt.Errorf("ui: %s=%s is specified but DirectX is not available.", envName, env)
	case "metal":
		if g := getter.getMetal(); g != nil {
			return g, nil
		}
		return nil, fmt.Errorf("ui: %s=%s is specified but Metal is not available", envName, env)
	default:
		return nil, fmt.Errorf("ui: an unsupported graphics library is specified: %s", env)
	}
}

func GraphicsDriverForTesting() graphicsdriver.Graphics {
	return theUI.graphicsDriver
}
