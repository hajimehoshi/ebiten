/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"image"
)

// Filter represents the type of filter to be used when a texture or a render
// target is maginified or minified.
type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
)

// TextureID represents an ID of a texture.
type TextureID int

// IsNil returns true if the texture is nil.
func (i TextureID) IsNil() bool {
	return i == 0
}

// RenderTargetID represents an ID of a render target.
// A render target is essentially same as a texture, but it is assumed that the
// all alpha of a render target is maximum.
type RenderTargetID int

// IsNil returns true if the render target is nil.
func (i RenderTargetID) IsNil() bool {
	return i == 0
}

// NewRenderTargetID returns an ID of a newly created render target.
func NewRenderTargetID(width, height int, filter Filter) (RenderTargetID, error) {
	if currentGameContext == nil {
		panic("graphics.NewRenderTarget: currentGameContext is not set.")
	}
	return currentGameContext.NewRenderTargetID(width, height, filter)
}

// NewRenderTargetID returns an ID of a newly created texture.
func NewTextureID(img image.Image, filter Filter) (TextureID, error) {
	if currentGameContext == nil {
		panic("graphics.NewTexture: currentGameContext is not set")
	}
	return currentGameContext.NewTextureID(img, filter)
}
