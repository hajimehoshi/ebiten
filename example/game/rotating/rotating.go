package rotating

import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
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
	ebitenTextureID graphics.TextureID
	x               int
	geometryMatrix  matrix.Geometry
}

func New() *Rotating {
	return &Rotating{}
}

func (game *Rotating) Init(tf graphics.TextureFactory) {
	file, err := os.Open("images/ebiten.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	if game.ebitenTextureID, err = tf.NewTextureFromImage(img); err != nil {
		panic(err)
	}
}

func (game *Rotating) Update(context ebiten.GameContext) {
	game.x++

	game.geometryMatrix = matrix.IdentityGeometry()
	tx, ty := float64(ebitenTextureWidth), float64(ebitenTextureHeight)
	game.geometryMatrix.Translate(-tx/2, -ty/2)
	game.geometryMatrix.Rotate(float64(game.x) * 2 * math.Pi / float64(ebiten.FPS*10))
	game.geometryMatrix.Translate(tx/2, ty/2)
	centerX := float64(context.ScreenWidth()) / 2
	centerY := float64(context.ScreenHeight()) / 2
	game.geometryMatrix.Translate(centerX-tx/2, centerY-ty/2)
}

func (game *Rotating) Draw(g graphics.Context) {
	g.Fill(128, 128, 255)
	g.DrawTexture(game.ebitenTextureID,
		game.geometryMatrix,
		matrix.IdentityColor())
}
