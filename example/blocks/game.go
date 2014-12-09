package blocks

import (
	"github.com/hajimehoshi/ebiten"
	"sync"
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

type Game struct {
	sceneManager *SceneManager
	ebiten       *Input
	textures     *Textures
}

func NewGame() *Game {
	game := &Game{
		sceneManager: NewSceneManager(NewTitleScene()),
		ebiten:       NewInput(),
		textures:     NewTextures(),
	}
	return game
}

func (game *Game) isInitialized() bool {
	for name := range texturePaths {
		if !game.textures.Has(name) {
			return false
		}
	}
	for name := range renderTargetSizes {
		if !game.textures.Has(name) {
			return false
		}
	}
	return true
}

var once sync.Once

func (game *Game) Update() error {
	once.Do(func() {
		for name, path := range texturePaths {
			game.textures.RequestTexture(name, path)
		}
		for name, size := range renderTargetSizes {
			game.textures.RequestRenderTarget(name, size)
		}
	})
	if !game.isInitialized() {
		return nil
	}
	game.ebiten.Update()
	game.sceneManager.Update(&GameState{
		SceneManager: game.sceneManager,
		Input:        game.ebiten,
	})
	return nil
}

func (game *Game) Draw(context ebiten.GraphicsContext) error {
	if !game.isInitialized() {
		return nil
	}
	game.sceneManager.Draw(context, game.textures)
	return nil
}
