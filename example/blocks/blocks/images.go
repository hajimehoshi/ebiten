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

type Images struct {
	imagePaths        chan namePath
	renderTargetSizes chan nameSize
	images            map[string]*ebiten.Image
	renderTargets     map[string]*ebiten.RenderTarget
	sync.RWMutex
}

func NewImages() *Images {
	images := &Images{
		imagePaths:        make(chan namePath),
		renderTargetSizes: make(chan nameSize),
		images:            map[string]*ebiten.Image{},
		renderTargets:     map[string]*ebiten.RenderTarget{},
	}
	go func() {
		for {
			images.loopMain()
		}
	}()
	return images
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

func (i *Images) loopMain() {
	select {
	case p := <-i.imagePaths:
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
			i.Lock()
			defer i.Unlock()
			i.images[name] = id
		}()
	case s := <-i.renderTargetSizes:
		name := s.name
		size := s.size
		go func() {
			id, err := ebiten.NewRenderTarget(size.Width, size.Height, ebiten.FilterNearest)
			if err != nil {
				panic(err)
			}
			i.Lock()
			defer i.Unlock()
			i.renderTargets[name] = id
		}()
	}
}

func (i *Images) RequestImage(name string, path string) {
	i.imagePaths <- namePath{name, path}
}

func (i *Images) RequestRenderTarget(name string, size Size) {
	i.renderTargetSizes <- nameSize{name, size}
}

func (i *Images) Has(name string) bool {
	i.RLock()
	defer i.RUnlock()
	_, ok := i.images[name]
	if ok {
		return true
	}
	_, ok = i.renderTargets[name]
	return ok
}

func (i *Images) GetImage(name string) *ebiten.Image {
	i.RLock()
	defer i.RUnlock()
	return i.images[name]
}

func (i *Images) GetRenderTarget(name string) *ebiten.RenderTarget {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargets[name]
}
