package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/ui"
	_ "image/png"
)

type Size struct {
	Width  int
	Height int
}

// TODO: Should they be global??
var texturePaths = map[string]string{}
var renderTargetSizes = map[string]Size{}

const ScreenWidth = 256
const ScreenHeight = 240

type GameState struct {
	SceneManager *SceneManager
	Input        *Input
}

type Textures interface {
	RequestTexture(name string, path string)
	RequestRenderTarget(name string, size Size)
	Has(name string) bool
	GetTexture(name string) graphics.TextureId
	GetRenderTarget(name string) graphics.RenderTargetId
}

type Game struct {
	sceneManager *SceneManager
	input        *Input
	textures     Textures
}

func NewGame(textures Textures) *Game {
	game := &Game{
		sceneManager: NewSceneManager(NewTitleScene()),
		input:        NewInput(),
		textures:     textures,
	}
	for name, path := range texturePaths {
		game.textures.RequestTexture(name, path)
	}
	for name, size := range renderTargetSizes {
		game.textures.RequestRenderTarget(name, size)
	}
	return game
}

func (game *Game) HandleEvent(e interface{}) {
	switch e := e.(type) {
	case ui.KeyStateUpdatedEvent:
		game.input.UpdateKeys(e.Keys)
	case ui.MouseStateUpdatedEvent:
	}
}

func (game *Game) isInitialized() bool {
	for name, _ := range texturePaths {
		if !game.textures.Has(name) {
			return false
		}
	}
	for name, _ := range renderTargetSizes {
		if !game.textures.Has(name) {
			return false
		}
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
	game.sceneManager.Draw(context, game.textures)
}
