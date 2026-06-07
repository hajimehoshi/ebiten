// main this program displays how you can use images in your game within a boundary.
// Boxes will bounce off the boundary wall of the game window size
package main

import (
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	ScreenWidth  = 640
	ScreenHeight = 480
	BoxSize      = 20
)

// rng used to generate random integers used in the program
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Box is the main struct used by this boundary boxes program
type Box struct {
	x      int
	y      int
	width  int
	height int
	speedX int
	speedY int
	//includes the ebiten Image used to draw the box
	image *ebiten.Image
}

type Game struct {
	boxes []Box
}

func (g *Game) Update() error {
	for i := range g.boxes {
		g.boxes[i].Move()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for i := range g.boxes {
		g.boxes[i].DrawBox(screen)
	}
}

func (g *Game) Layout(int, int) (sw int, sh int) {
	return ScreenWidth, ScreenHeight
}

func (b *Box) MoveX() {
	b.x += b.speedX
	if b.x > ScreenWidth-BoxSize || b.x < 0 {
		b.speedX *= -1 //reverses the position to "bounce" off the boundary wall
	}
}

func (b *Box) MoveY() {
	b.y += b.speedY
	if b.y > ScreenHeight-BoxSize || b.y < 0 {
		b.speedY *= -1 //reverses the position to "bounce" off the boundary wall
	}
}

func (b *Box) Move() {
	b.MoveX()
	b.MoveY()
}

func (b *Box) DrawBox(screen *ebiten.Image) {
	options := ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(b.x), float64(b.y))
	screen.DrawImage(b.image, &options)
}

func getRandomPosition(limit int) int {
	return rng.Intn(limit)
}

func getRandomSpeed() int {
	//speed can be between -3 and 3
	speed := rng.Intn(7) - 3
	if speed == 0 {
		speed = 1
	}
	return speed
}

func NewBoxImage(width, height int) *ebiten.Image {
	boxImage := ebiten.NewImage(width, height)

	//fill box with a random color
	boxImage.Fill(color.RGBA{
		R: uint8(rng.Intn(255)),
		G: uint8(rng.Intn(255)),
		B: uint8(rng.Intn(255)),
	})
	return boxImage
}

// NewBox initializes a box with random values
func NewBox() Box {
	b := Box{}
	b.x = getRandomPosition(ScreenWidth - BoxSize)
	b.y = getRandomPosition(ScreenHeight - BoxSize)
	b.width = BoxSize
	b.height = BoxSize
	b.speedX = getRandomSpeed()
	b.speedY = getRandomSpeed()
	b.image = NewBoxImage(b.width, b.height)
	return b
}

func main() {
	game := &Game{}
	//add 50 boxes, change this value if you want more boxes
	for i := 0; i < 50; i++ {
		box := NewBox()
		game.boxes = append(game.boxes, box)
	}
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Boundary boxes")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
