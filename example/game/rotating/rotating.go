package rotating

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image"
	_ "image/png"
	"math"
	"os"
)

const (
	ebitenTextureWidth  = 57
	ebitenTextureHeight = 26
)

type Rotating struct {
	ebitenTextureId     graphics.TextureId
	x                   int
	geometryMatrix      matrix.Geometry
	screenSizeUpdatedCh chan ui.ScreenSizeUpdatedEvent
	screenWidth         int
	screenHeight        int
}

func New() *Rotating {
	return &Rotating{
		screenSizeUpdatedCh: make(chan ui.ScreenSizeUpdatedEvent),
	}
}

func (game *Rotating) OnScreenSizeUpdated(e ui.ScreenSizeUpdatedEvent) {
	go func() {
		e := e
		game.screenSizeUpdatedCh <- e
	}()
}

func (game *Rotating) InitTextures(tf graphics.TextureFactory) {
	file, err := os.Open("images/ebiten.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	if game.ebitenTextureId, err = tf.CreateTextureFromImage(img); err != nil {
		panic(err)
	}
}

func (game *Rotating) Update() {
events:
	for {
		select {
		case e := <-game.screenSizeUpdatedCh:
			game.screenWidth, game.screenHeight = e.Width, e.Height
		default:
			break events
		}
	}

	if game.screenWidth == 0 || game.screenHeight == 0 {
		return
	}

	const fps = 60

	game.x++

	game.geometryMatrix = matrix.IdentityGeometry()
	tx, ty := float64(ebitenTextureWidth), float64(ebitenTextureHeight)
	game.geometryMatrix.Translate(-tx/2, -ty/2)
	game.geometryMatrix.Rotate(float64(game.x) * 2 * math.Pi / float64(fps*10))
	game.geometryMatrix.Translate(tx/2, ty/2)
	centerX := float64(game.screenWidth) / 2
	centerY := float64(game.screenHeight) / 2
	game.geometryMatrix.Translate(centerX-tx/2, centerY-ty/2)
}

func (game *Rotating) Draw(g graphics.Canvas) {
	g.Fill(128, 128, 255)
	g.DrawTexture(game.ebitenTextureId,
		game.geometryMatrix,
		matrix.IdentityColor())
}
