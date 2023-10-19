package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"image"
	_ "image/png"
)

var (
	runningImg *ebiten.Image
)

const (
	movingSpeed = 1
)

type Frame struct {
	x      int
	y      int
	width  int
	height int
}

type Man struct {
	frame Frame
	img   *ebiten.Image
}

const (
	forward = iota
	leftward
	rightward
	backward
)

const (
	stop = iota
)

func (m *Man) IfCollision(o *Obstacle) (dir, bool) {
	return CheckCollision(m.frame, o.frame)
}

func NewMan() (*Man, error) {
	img, _, errImg := ebitenutil.NewImageFromFile("D:\\Program Files\\JetBrains\\GoLand 2023.1.3\\workspaces\\runningman\\game\\img.png")
	if errImg != nil {
		panic(errImg)
	}

	runningImg = ebiten.NewImageFromImage(img)
	w := runningImg.Bounds().Dx()
	h := runningImg.Bounds().Dy()
	f := &Frame{
		x:      0,
		y:      0,
		width:  w,
		height: h,
	}
	m := &Man{
		frame: *f,
		img:   runningImg,
	}
	return m, errImg
}

func (m *Man) Update(i *Input, o *Obstacle) {
	dir, _ := m.IfCollision(o)
	switch i.state {
	case keyup:
		if dir == "down" {
			return
		}
		m.frame.y -= movingSpeed
	case keydown:
		if dir == "up" {
			return
		}
		m.frame.y += movingSpeed
	case keyleft:
		if dir == "right" {
			return
		}
		m.frame.x -= movingSpeed
	case keyright:
		if dir == "left" {
			return
		}
		m.frame.x += movingSpeed
	}
}

var (
	psy int
)

func (m *Man) Draw(screen *ebiten.Image, state int, counter int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(m.frame.x), float64(m.frame.y))
	w := m.img.Bounds().Dx() / 4
	h := m.img.Bounds().Dy() / 4
	sx := counter
	var sy int
	switch state {
	case keyup:
		sy = backward
	case keydown:
		sy = forward
	case keyleft:
		sy = leftward
	case keyright:
		sy = rightward
	case keynone:
		sy = psy
		sx = stop
	}
	screen.DrawImage(m.img.SubImage(image.Rect(sx*w, sy*h, (sx+1)*w, (sy+1)*h)).(*ebiten.Image), op)
	psy = sy
}
