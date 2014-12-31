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

import (
	"errors"
	"fmt"
	"github.com/go-gl/gl"
)

type Filter int

const (
	filterNearest Filter = gl.NEAREST
	filterLinear         = gl.LINEAR
)

type Context struct {
	Nearest Filter
	Linear  Filter
}

type Texture gl.Texture

func (t Texture) Pixels(width, height int) ([]uint8, error) {
	// TODO: Use glGetTexLevelParameteri and GL_TEXTURE_WIDTH?
	pixels := make([]uint8, 4*width*height)
	gl.Texture(t).Bind(gl.TEXTURE_2D)
	gl.GetTexImage(gl.TEXTURE_2D, 0, gl.RGBA, gl.UNSIGNED_BYTE, pixels)
	if e := gl.GetError(); e != gl.NO_ERROR {
		// TODO: Use glu.ErrorString
		return nil, errors.New(fmt.Sprintf("gl error: %d", e))
	}
	return pixels, nil
}

func (t Texture) Delete() {
	gl.Texture(t).Delete()
}

type Framebuffer gl.Framebuffer

func (f Framebuffer) SetAsViewport(width, height int) error {
	gl.Flush()
	gl.Framebuffer(f).Bind()
	err := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if err != gl.FRAMEBUFFER_COMPLETE {
		if gl.GetError() != 0 {
			return errors.New(fmt.Sprintf("glBindFramebuffer failed: %d", gl.GetError()))
		}
		return errors.New("glBindFramebuffer failed: the context is different?")
	}
	gl.Viewport(0, 0, width, height)
	return nil
}

func (f Framebuffer) Delete() {
	gl.Framebuffer(f).Delete()
}

func NewContext() *Context {
	c := &Context{
		Nearest: filterNearest,
		Linear:  filterLinear,
	}
	c.init()
	return c
}

func (c *Context) init() {
	gl.Init()
	gl.Enable(gl.TEXTURE_2D)
	// Textures' pixel formats are alpha premultiplied.
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
}

func (c *Context) NewTexture(width, height int, pixels []uint8, filter Filter) (Texture, error) {
	t := gl.GenTexture()
	if t < 0 {
		return 0, errors.New("glGenTexture failed")
	}
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	t.Bind(gl.TEXTURE_2D)
	defer gl.Texture(0).Bind(gl.TEXTURE_2D)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int(filter))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int(filter))

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, pixels)

	return Texture(t), nil
}

func (c *Context) NewFramebuffer(texture Texture) (Framebuffer, error) {
	f := gl.GenFramebuffer()
	f.Bind()

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, gl.Texture(texture), 0)
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return 0, errors.New("creating framebuffer failed")
	}

	return Framebuffer(f), nil
}
