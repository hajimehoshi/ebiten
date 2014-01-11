package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image"
	_ "image/png"
	"os"
)

type Size struct {
	Width  int
	Height int
}

const ScreenWidth = 256
const ScreenHeight = 240

var texturePaths = map[string]string{}
var renderTargetSizes = map[string]Size{}

// TODO: Make this not a global variable.
var drawInfo = struct {
	textures      map[string]graphics.TextureId
	renderTargets map[string]graphics.RenderTargetId
}{
	textures:      map[string]graphics.TextureId{},
	renderTargets: map[string]graphics.RenderTargetId{},
}

type GameState struct {
	SceneManager *SceneManager
	Input        *Input
}

type Game struct {
	sceneManager *SceneManager
	input        *Input
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

func (game *Game) startLoadingTextures(textureFactory graphics.TextureFactory) {
	for tag, path := range texturePaths {
		tag := tag
		path := path
		go func() {
			img, err := loadImage(path)
			if err != nil {
				panic(err)
			}
			textureFactory.CreateTexture(tag, img, graphics.FilterNearest)
		}()
	}

	for tag, size := range renderTargetSizes {
		tag := tag
		size := size
		go func() {
			textureFactory.CreateRenderTarget(tag, size.Width, size.Height, graphics.FilterNearest)
		}()
	}
}

func NewGame(textureFactory graphics.TextureFactory) *Game {
	game := &Game{
		sceneManager: NewSceneManager(NewTitleScene()),
		input:        NewInput(),
	}
	game.startLoadingTextures(textureFactory)
	return game
}

func (game *Game) HandleEvent(e interface{}) {
	switch e := e.(type) {
	case graphics.TextureCreatedEvent:
		if e.Error != nil {
			panic(e.Error)
		}
		drawInfo.textures[e.Tag.(string)] = e.Id
	case graphics.RenderTargetCreatedEvent:
		if e.Error != nil {
			panic(e.Error)
		}
		drawInfo.renderTargets[e.Tag.(string)] = e.Id
	case ui.KeyStateUpdatedEvent:
		game.input.UpdateKeys(e.Keys)
	case ui.MouseStateUpdatedEvent:
	}
}

func (game *Game) isInitialized() bool {
	if len(drawInfo.textures) < len(texturePaths) {
		return false
	}
	if len(drawInfo.renderTargets) < len(renderTargetSizes) {
		return false
	}
	return true
}

func (game *Game) Update() {
	if !game.isInitialized() {
		return
	}
	game.input.Update()
	game.sceneManager.Update(&GameState{
		SceneManager: game.sceneManager,
		Input:        game.input,
	})
}

func (game *Game) Draw(context graphics.Context) {
	if !game.isInitialized() {
		return
	}
	game.sceneManager.Draw(context)
}
