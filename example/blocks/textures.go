package blocks

import (
	"github.com/hajimehoshi/ebiten/graphics"
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
	textures          map[string]graphics.TextureID
	renderTargets     map[string]graphics.RenderTargetID
	sync.RWMutex
}

func NewTextures() *Textures {
	textures := &Textures{
		texturePaths:      make(chan namePath),
		renderTargetSizes: make(chan nameSize),
		textures:          map[string]graphics.TextureID{},
		renderTargets:     map[string]graphics.RenderTargetID{},
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
			id, err := graphics.NewTextureID(img, graphics.FilterNearest)
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
			id, err := graphics.NewRenderTargetID(size.Width, size.Height, graphics.FilterNearest)
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

func (t *Textures) GetTexture(name string) graphics.TextureID {
	t.RLock()
	defer t.RUnlock()
	return t.textures[name]
}

func (t *Textures) GetRenderTarget(name string) graphics.RenderTargetID {
	t.RLock()
	defer t.RUnlock()
	return t.renderTargets[name]
}
