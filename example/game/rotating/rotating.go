package rotating

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
)

type Rotating struct {
	ebitenTexture graphics.Texture
	x             int
}

func New() *Rotating {
	return &Rotating{}
}

func (game *Rotating) ScreenWidth() int {
	return 256
}

func (game *Rotating) ScreenHeight() int {
	return 240
}

func (game *Rotating) Fps() int {
	return 60
}

func (game *Rotating) Init(tf graphics.TextureFactory) {
	file, err := os.Open("ebiten.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	if game.ebitenTexture, err = tf.NewTextureFromImage(img); err != nil {
		panic(err)
	}
}

func (game *Rotating) Update() {
	game.x++
}

func (game *Rotating) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})

	geometryMatrix := matrix.IdentityGeometry()
	tx, ty := float64(game.ebitenTexture.Width), float64(game.ebitenTexture.Height)
	geometryMatrix.Translate(-tx/2, -ty/2)
	geometryMatrix.Rotate(float64(game.x) * 2 * math.Pi / float64(game.Fps()*10))
	geometryMatrix.Translate(tx/2, ty/2)
	centerX := float64(game.ScreenWidth()) / 2
	centerY := float64(game.ScreenHeight()) / 2
	geometryMatrix.Translate(centerX-tx/2, centerY-ty/2)

	g.DrawTexture(game.ebitenTexture.ID,
		geometryMatrix,
		matrix.IdentityColor())
}
