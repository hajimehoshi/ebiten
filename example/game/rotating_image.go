package game

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	_ "image/png"
	"os"
)

type RotatingImage struct {
	ebitenTexture graphics.Texture
	x             int
}

func NewRotatingImage() *RotatingImage {
	return &RotatingImage{}
}

func (game *RotatingImage) ScreenWidth() int {
	return 256
}

func (game *RotatingImage) ScreenHeight() int {
	return 240
}

func (game *RotatingImage) Init(tf graphics.TextureFactory) {
	file, err := os.Open("ebiten.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	game.ebitenTexture = tf.NewTextureFromImage(img)
}

func (game *RotatingImage) Update() {
	game.x++
}

func (game *RotatingImage) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})

	geometryMatrix := matrix.IdentityGeometry()
	tx, ty := float64(game.ebitenTexture.Width), float64(game.ebitenTexture.Height)
	geometryMatrix.Translate(-tx/2, -ty/2)
	geometryMatrix.Rotate(float64(game.x) / 60)
	geometryMatrix.Translate(tx/2, ty/2)
	centerX := float64(game.ScreenWidth()) / 2
	centerY := float64(game.ScreenHeight()) / 2
	geometryMatrix.Translate(centerX-tx/2, centerY-ty/2)

	g.DrawTexture(game.ebitenTexture,
		geometryMatrix,
		matrix.IdentityColor())
}
