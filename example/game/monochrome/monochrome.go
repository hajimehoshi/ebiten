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
	ch            chan bool
	colorMatrix   matrix.Color
}

func New() *Monochrome {
	return &Monochrome{
		ch: make(chan bool),
		colorMatrix: matrix.IdentityColor(),
	}
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

	go game.update()
}

func mean(a, b matrix.Color, k float64) matrix.Color {
	dim := a.Dim()
	result := matrix.Color{}
	for i := 0; i < dim - 1; i++ {
		for j := 0; j < dim; j++ {
			result.Elements[i][j] =
				a.Elements[i][j] * (1 - k) +
				b.Elements[i][j] * k
		}
	}
	return result
}

func (game *Monochrome) update() {
	colorI := matrix.IdentityColor()
	colorMonochrome := matrix.Monochrome()
	for {
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			rate := float64(i) / float64(game.Fps())
			game.colorMatrix = mean(colorI, colorMonochrome, rate)
			game.ch <- true
		}
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			game.colorMatrix = colorMonochrome
			game.ch <- true
		}
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			rate := float64(i) / float64(game.Fps())
			game.colorMatrix = mean(colorMonochrome, colorI, rate)
			game.ch <- true
		}
		for i := 0; i < game.Fps(); i++ {
			<-game.ch
			game.colorMatrix = colorI
			game.ch <- true
		}
	}
}

func (game *Monochrome) Update() {
	game.ch <- true
	<-game.ch
}

func (game *Monochrome) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})

	geometryMatrix := matrix.IdentityGeometry()
	tx := game.ScreenWidth() / 2 - game.ebitenTexture.Width / 2
	ty := game.ScreenHeight() / 2 - game.ebitenTexture.Height / 2
	geometryMatrix.Translate(float64(tx), float64(ty))
	g.DrawTexture(game.ebitenTexture.ID,
		geometryMatrix, game.colorMatrix)
}
