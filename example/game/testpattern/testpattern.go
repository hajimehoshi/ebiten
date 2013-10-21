package testpattern

import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
	_ "image/png"
	"math"
	"os"
)

type TestPattern struct {
	textureId     graphics.TextureId
	textureWidth  int
	textureHeight int
	geos          []matrix.Geometry
}

func New() *TestPattern {
	return &TestPattern{}
}

func (game *TestPattern) InitTextures(tf graphics.TextureFactory) {
	file, err := os.Open("images/test_pattern.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	if game.textureId, err = tf.NewTextureFromImage(img); err != nil {
		panic(err)
	}
	size := img.Bounds().Size()
	game.textureWidth = size.X
	game.textureHeight = size.Y
}

func (game *TestPattern) Update(context ebiten.GameContext) {
	geo := matrix.IdentityGeometry()
	geo.Translate(13, 13)
	game.geos = append(game.geos, geo)

	geo = matrix.IdentityGeometry()
	geo.Translate(float64(13+game.textureWidth), 13)
	game.geos = append(game.geos, geo)

	geo = matrix.IdentityGeometry()
	geo.Translate(float64(13+game.textureWidth*2), 13)
	game.geos = append(game.geos, geo)

	geo = matrix.IdentityGeometry()
	geo.Translate(float64(13+game.textureWidth*3), 13)
	game.geos = append(game.geos, geo)

	geo = matrix.IdentityGeometry()
	geo.Scale(2, 2)
	geo.Translate(13, float64(13+game.textureHeight))
	game.geos = append(game.geos, geo)

	geo = matrix.IdentityGeometry()
	geo.Rotate(math.Pi)
	geo.Scale(2, 2)
	geo.Translate(float64(game.textureWidth*2),
		float64(game.textureHeight*2))
	geo.Translate(float64(13+game.textureWidth*2),
		float64(13+game.textureHeight))
	game.geos = append(game.geos, geo)
}

func (game *TestPattern) Draw(g graphics.Context) {
	for _, geo := range game.geos {
		g.DrawTexture(game.textureId, geo, matrix.IdentityColor())
	}
}
