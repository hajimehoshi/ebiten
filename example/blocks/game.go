package blocks

import (
	"fmt"
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

var texturePaths = map[string]string{
	"background": "images/blocks/background.png",
	"font":       "images/blocks/font.png",
}

var renderTargetSizes = map[string]Size{
	"whole": Size{256, 254},
}

const ScreenWidth = 256
const ScreenHeight = 240

var drawInfo = struct {
	textures      map[string]graphics.TextureId
	renderTargets map[string]graphics.RenderTargetId
}{
	textures:      map[string]graphics.TextureId{},
	renderTargets: map[string]graphics.RenderTargetId{},
}

type Game struct {
	sceneManager *SceneManager
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
			textureFactory.CreateRenderTarget(tag, size.Width, size.Height)
		}()
	}
}

func NewGame(textureFactory graphics.TextureFactory) *Game {
	game := &Game{
		sceneManager: NewSceneManager(NewTitleScene()),
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
		fmt.Printf("%v\n", e.Keys)
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
	game.sceneManager.Update()
}

func (game *Game) Draw(context graphics.Context) {
	if !game.isInitialized() {
		return
	}
	game.sceneManager.Draw(context)
}

/*func (game *Game) drawText(g graphics.Context, text string, x, y int, clr color.Color) {
	const letterWidth = 6
	const letterHeight = 16

	parts := []graphics.TexturePart{}
	textX := 0
	textY := 0
	for _, c := range text {
		if c == '\n' {
			textX = 0
			textY += letterHeight
			continue
		}
		code := int(c)
		x := (code % 32) * letterWidth
		y := (code / 32) * letterHeight
		source := graphics.Rect{x, y, letterWidth, letterHeight}
		parts = append(parts, graphics.TexturePart{
			LocationX: textX,
			LocationY: textY,
			Source:    source,
		})
		textX += letterWidth
	}

	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Translate(float64(x), float64(y))
	colorMatrix := matrix.IdentityColor()
	colorMatrix.Scale(clr)
	g.DrawTextureParts(drawInfo.textures["text"], parts,
		geometryMatrix, colorMatrix)
}*/
