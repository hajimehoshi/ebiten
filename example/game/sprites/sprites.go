package sprites

import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
	"math/rand"
	"os"
	"time"
)

const (
	ebitenTextureWidth = 57
	ebitenTextureHeight = 26
)

type Sprite struct {
	width  int
	height int
	ch     chan bool
	x      int
	y      int
	vx     int
	vy     int
}

func newSprite(screenWidth, screenHeight, width, height int) *Sprite {
	maxX := screenWidth - width
	maxY := screenHeight - height
	sprite := &Sprite{
		width:  width,
		height: height,
		ch:     make(chan bool),
		x:      rand.Intn(maxX),
		y:      rand.Intn(maxY),
		vx:     rand.Intn(2)*2 - 1,
		vy:     rand.Intn(2)*2 - 1,
	}
	go sprite.update(screenWidth, screenHeight)
	return sprite
}

func (sprite *Sprite) update(screenWidth, screenHeight int) {
	maxX := screenWidth - sprite.width
	maxY := screenHeight - sprite.height
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

type Sprites struct {
	ebitenTexture graphics.Texture
	sprites       []*Sprite
}

func New() *Sprites {
	return &Sprites{}
}

func (game *Sprites) Init(tf graphics.TextureFactory) {
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
}

func (game *Sprites) Update(context ebiten.GameContext) {
	if game.sprites == nil {
		game.sprites = []*Sprite{}
		for i := 0; i < 100; i++ {
			sprite := newSprite(
				context.ScreenWidth(),
				context.ScreenHeight(),
				ebitenTextureWidth,
				ebitenTextureHeight)
			game.sprites = append(game.sprites, sprite)
		}
	}

	for _, sprite := range game.sprites {
		sprite.Update()
	}
}

func (game *Sprites) Draw(g graphics.Context) {
	g.Fill(128, 128, 255)

	// Draw the sprites
	locations := make([]graphics.TexturePart, 0, len(game.sprites))
	texture := game.ebitenTexture
	for _, sprite := range game.sprites {
		location := graphics.TexturePart{
			LocationX: sprite.x,
			LocationY: sprite.y,
			Source: graphics.Rect{
				0, 0, ebitenTextureWidth, ebitenTextureHeight,
			},
		}
		locations = append(locations, location)
	}
	geometryMatrix := matrix.IdentityGeometry()
	g.DrawTextureParts(texture.ID(), locations,
		geometryMatrix, matrix.IdentityColor())
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
