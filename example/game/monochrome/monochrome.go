package monochrome

import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
	_ "image/png"
	"os"
)

type Monochrome struct {
	ebitenTexture  graphics.Texture
	ch             chan bool
	colorMatrix    matrix.Color
	geometryMatrix matrix.Geometry
}

func New() *Monochrome {
	return &Monochrome{
		ch:             make(chan bool),
		colorMatrix:    matrix.IdentityColor(),
		geometryMatrix: matrix.IdentityGeometry(),
	}
}

func (game *Monochrome) Init(tf graphics.TextureFactory) {
	file, err := os.Open("images/ebiten.png")
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
	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			result.Elements[i][j] =
				a.Elements[i][j]*(1-k) +
					b.Elements[i][j]*k
		}
	}
	return result
}

func (game *Monochrome) update() {
	colorI := matrix.IdentityColor()
	colorMonochrome := matrix.Monochrome()
	for {
		for i := 0; i < ebiten.FPS; i++ {
			<-game.ch
			rate := float64(i) / float64(ebiten.FPS)
			game.colorMatrix = mean(colorI, colorMonochrome, rate)
			game.ch <- true
		}
		for i := 0; i < ebiten.FPS; i++ {
			<-game.ch
			game.colorMatrix = colorMonochrome
			game.ch <- true
		}
		for i := 0; i < ebiten.FPS; i++ {
			<-game.ch
			rate := float64(i) / float64(ebiten.FPS)
			game.colorMatrix = mean(colorMonochrome, colorI, rate)
			game.ch <- true
		}
		for i := 0; i < ebiten.FPS; i++ {
			<-game.ch
			game.colorMatrix = colorI
			game.ch <- true
		}
	}
}

func (game *Monochrome) Update(context ebiten.GameContext) {
	game.ch <- true
	<-game.ch

	game.geometryMatrix = matrix.IdentityGeometry()
	tx := context.ScreenWidth()/2 - game.ebitenTexture.Width()/2
	ty := context.ScreenHeight()/2 - game.ebitenTexture.Height()/2
	game.geometryMatrix.Translate(float64(tx), float64(ty))
}

func (game *Monochrome) Draw(g graphics.Context) {
	g.Fill(128, 128, 255)

	g.DrawTexture(game.ebitenTexture.ID(),
		game.geometryMatrix, game.colorMatrix)
}
