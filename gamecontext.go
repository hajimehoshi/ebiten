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

package ebiten

import (
	audio "github.com/hajimehoshi/ebiten/exp/audio/internal"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"image"
)

var glContext *opengl.Context

func init() {
	ui.Init()
	ui.ExecOnUIThread(func() {
		glContext = opengl.NewContext()
	})
	audio.Init()
}

// NewImage returns an empty image.
//
// NewImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
func NewImage(width, height int, filter Filter) (*Image, error) {
	var img *Image
	var err error
	useGLContext(func(c *opengl.Context) {
		var texture *graphics.Texture
		var framebuffer *graphics.Framebuffer
		texture, err = graphics.NewTexture(c, width, height, glFilter(c, filter))
		if err != nil {
			return
		}
		framebuffer, err = graphics.NewFramebufferFromTexture(c, texture)
		if err != nil {
			return
		}
		img = &Image{framebuffer: framebuffer, texture: texture}
	})
	if err != nil {
		return nil, err
	}
	if err := img.Clear(); err != nil {
		return nil, err
	}
	return img, nil
}

// NewImageFromImage creates a new image with the given image (img).
//
// NewImageFromImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
func NewImageFromImage(img image.Image, filter Filter) (*Image, error) {
	var eimg *Image
	var err error
	useGLContext(func(c *opengl.Context) {
		var texture *graphics.Texture
		var framebuffer *graphics.Framebuffer
		texture, err = graphics.NewTextureFromImage(c, img, glFilter(c, filter))
		if err != nil {
			return
		}
		framebuffer, err = graphics.NewFramebufferFromTexture(c, texture)
		if err != nil {
			return
		}
		eimg = &Image{framebuffer: framebuffer, texture: texture}
	})
	if err != nil {
		return nil, err
	}
	return eimg, nil
}
