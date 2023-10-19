package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	obstacleImg *ebiten.Image
)

type Obstacle struct {
	frame Frame
	img   *ebiten.Image
}

func NewObstacle() (*Obstacle, error) {
	img, _, errImg := ebitenutil.NewImageFromFile("D:\\Program Files\\JetBrains\\GoLand 2023.1.3\\workspaces\\runningman\\game\\img.png")
	if errImg != nil {
		panic(errImg)
	}
	obstacleImg = ebiten.NewImageFromImage(img)
	w := obstacleImg.Bounds().Dx()
	h := obstacleImg.Bounds().Dy()
	f := &Frame{
		x:      screenWidth / 2,
		y:      screenHeight / 2,
		width:  w,
		height: h,
	}
	o := &Obstacle{
		frame: *f,
		img:   obstacleImg,
	}
	return o, errImg
}

func (o *Obstacle) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(o.frame.x), float64(o.frame.y))
	screen.DrawImage(o.img, op)
}
