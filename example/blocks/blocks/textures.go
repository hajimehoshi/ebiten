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

package blocks

import (
	"github.com/hajimehoshi/ebiten"
	"image"
	_ "image/png"
	"os"
	"sync"
)

type namePath struct {
	name string
	path string
}

type nameSize struct {
	name string
	size Size
}

type Textures struct {
	texturePaths      chan namePath
	renderTargetSizes chan nameSize
	textures          map[string]*ebiten.Image
	renderTargets     map[string]*ebiten.RenderTarget
	sync.RWMutex
}

func NewTextures() *Textures {
	textures := &Textures{
		texturePaths:      make(chan namePath),
		renderTargetSizes: make(chan nameSize),
		textures:          map[string]*ebiten.Image{},
		renderTargets:     map[string]*ebiten.RenderTarget{},
	}
	go func() {
		for {
			textures.loopMain()
		}
	}()
	return textures
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (t *Textures) loopMain() {
	select {
	case p := <-t.texturePaths:
		name := p.name
		path := p.path
		go func() {
			img, err := loadImage(path)
			if err != nil {
				panic(err)
			}
			id, err := ebiten.NewImage(img, ebiten.FilterNearest)
			if err != nil {
				panic(err)
			}
			t.Lock()
			defer t.Unlock()
			t.textures[name] = id
		}()
	case s := <-t.renderTargetSizes:
		name := s.name
		size := s.size
		go func() {
			id, err := ebiten.NewRenderTarget(size.Width, size.Height, ebiten.FilterNearest)
			if err != nil {
				panic(err)
			}
			t.Lock()
			defer t.Unlock()
			t.renderTargets[name] = id
		}()
	}
}

func (t *Textures) RequestTexture(name string, path string) {
	t.texturePaths <- namePath{name, path}
}

func (t *Textures) RequestRenderTarget(name string, size Size) {
	t.renderTargetSizes <- nameSize{name, size}
}

func (t *Textures) Has(name string) bool {
	t.RLock()
	defer t.RUnlock()
	_, ok := t.textures[name]
	if ok {
		return true
	}
	_, ok = t.renderTargets[name]
	return ok
}

func (t *Textures) GetTexture(name string) *ebiten.Image {
	t.RLock()
	defer t.RUnlock()
	return t.textures[name]
}

func (t *Textures) GetRenderTarget(name string) *ebiten.RenderTarget {
	t.RLock()
	defer t.RUnlock()
	return t.renderTargets[name]
}
