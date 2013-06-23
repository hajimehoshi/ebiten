package monochrome

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	_ "image/png"
	"os"
)

type Monochrome struct {
	ebitenTexture graphics.Texture
}

func New() *Monochrome {
	return &Monochrome{}
}

func (game *Monochrome) ScreenWidth() int {
	return 256
}

func (game *Monochrome) ScreenHeight() int {
	return 240
}

func (game *Monochrome) Fps() int {
	return 60
}

func (game *Monochrome) Init(tf graphics.TextureFactory) {
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

func (game *Monochrome) Update() {
}

func (game *Monochrome) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})

	geometryMatrix := matrix.IdentityGeometry()
	tx := game.ScreenWidth() / 2 - game.ebitenTexture.Width / 2
	ty := game.ScreenHeight() / 2 - game.ebitenTexture.Height / 2
	geometryMatrix.Translate(float64(tx), float64(ty))
	g.DrawTexture(game.ebitenTexture.ID,
		geometryMatrix, matrix.Monochrome())
}
