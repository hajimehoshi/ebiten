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

package opengl

// framebuffer is a wrapper of OpenGL's framebuffer.
type framebuffer struct {
	driver    *Driver
	native    framebufferNative
	proMatrix []float32
	width     int
	height    int
}

// newFramebufferFromTexture creates a framebuffer from the given texture.
func newFramebufferFromTexture(context *context, texture textureNative, width, height int) (*framebuffer, error) {
	native, err := context.newFramebuffer(texture)
	if err != nil {
		return nil, err
	}
	return &framebuffer{
		native: native,
		width:  width,
		height: height,
	}, nil
}

// newScreenFramebuffer creates a framebuffer for the screen.
func newScreenFramebuffer(context *context, width, height int) *framebuffer {
	return &framebuffer{
		native: context.getScreenFramebuffer(),
		width:  width,
		height: height,
	}
}

// projectionMatrix returns a projection matrix of the framebuffer.
//
// A projection matrix converts the coodinates on the framebuffer
// (0, 0) - (viewport width, viewport height)
// to the normalized device coodinates (-1, -1) - (1, 1) with adjustment.
func (f *framebuffer) projectionMatrix() []float32 {
	if f.proMatrix != nil {
		return f.proMatrix
	}
	f.proMatrix = orthoProjectionMatrix(0, f.width, 0, f.height)
	return f.proMatrix
}

func (f *framebuffer) delete(context *context) {
	if f.native != context.getScreenFramebuffer() {
		context.deleteFramebuffer(f.native)
	}
}
