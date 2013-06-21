package game

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	"math/rand"
	"os"
	"time"
)

type Sprite struct {
	texture graphics.Texture
	ch      chan bool
	x       int
	y       int
	vx      int
	vy      int
}

func NewSprite(screenWidth, screenHeight int,
	texture graphics.Texture) *Sprite {
	maxX := screenWidth - texture.Width
	maxY := screenHeight - texture.Height
	sprite := &Sprite{
		texture: texture,
		ch:      make(chan bool),
		x:       rand.Intn(maxX),
		y:       rand.Intn(maxY),
		vx:      rand.Intn(2)*2 - 1,
		vy:      rand.Intn(2)*2 - 1,
	}
	go sprite.update(screenWidth, screenHeight)
	return sprite
}

func (sprite *Sprite) update(screenWidth, screenHeight int) {
	maxX := screenWidth - sprite.texture.Width
	maxY := screenHeight - sprite.texture.Height
	for {
		<-sprite.ch
		sprite.x += sprite.vx
		sprite.y += sprite.vy
		if sprite.x < 0 || maxX <= sprite.x {
			sprite.vx = -sprite.vx
		}
		if sprite.y < 0 || maxY <= sprite.y {
			sprite.vy = -sprite.vy
		}
		sprite.ch <- true
	}
}

func (sprite *Sprite) Update() {
	sprite.ch <- true
	<-sprite.ch
}

func (sprite *Sprite) Draw(g graphics.GraphicsContext) {
	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Translate(float64(sprite.x), float64(sprite.y))

	g.DrawTexture(sprite.texture.ID,
		graphics.Rectangle{0, 0, sprite.texture.Width, sprite.texture.Height},
		geometryMatrix,
		matrix.IdentityColor())
}

type Sprites struct {
	ebitenTexture graphics.Texture
	sprites       []*Sprite
}

func NewSprites() *Sprites {
	return &Sprites{}
}

func (game *Sprites) ScreenWidth() int {
	return 256
}

func (game *Sprites) ScreenHeight() int {
	return 240
}

func (game *Sprites) Init(tf graphics.TextureFactory) {
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
	game.sprites = []*Sprite{}
	for i := 0; i < 200; i++ {
		sprite := NewSprite(
			game.ScreenWidth(),
			game.ScreenHeight(),
			game.ebitenTexture)
		game.sprites = append(game.sprites, sprite)
	}
}

func (game *Sprites) Update() {
	for _, sprite := range game.sprites {
		sprite.Update()
	}
}

func (game *Sprites) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
	g.Fill(&color.RGBA{R: 128, G: 128, B: 255, A: 255})
	for _, sprite := range game.sprites {
		sprite.Draw(g)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
