// Copyright 2017 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"strings"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	uiFontSize          = 12
	lineSpacingInPixels = 16
)

var (
	uiImage      *ebiten.Image
	uiFaceSource *text.GoTextFaceSource
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.UI_png))
	if err != nil {
		log.Fatal(err)
	}
	uiImage = ebiten.NewImageFromImage(img)
}

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	uiFaceSource = s
}

type imageType int

const (
	imageTypeButton imageType = iota
	imageTypeButtonPressed
	imageTypeTextBox
	imageTypeVScrollBarBack
	imageTypeVScrollBarFront
	imageTypeCheckBox
	imageTypeCheckBoxPressed
	imageTypeCheckBoxMark
)

var imageSrcRects = map[imageType]image.Rectangle{
	imageTypeButton:          image.Rect(0, 0, 16, 16),
	imageTypeButtonPressed:   image.Rect(16, 0, 32, 16),
	imageTypeTextBox:         image.Rect(0, 16, 16, 32),
	imageTypeVScrollBarBack:  image.Rect(16, 16, 24, 32),
	imageTypeVScrollBarFront: image.Rect(24, 16, 32, 32),
	imageTypeCheckBox:        image.Rect(0, 32, 16, 48),
	imageTypeCheckBoxPressed: image.Rect(16, 32, 32, 48),
	imageTypeCheckBoxMark:    image.Rect(32, 32, 48, 48),
}

const (
	screenWidth  = 640
	screenHeight = 480
)

type Input struct {
	mouseButtonState int
}

func drawNinePatches(dst *ebiten.Image, dstRect image.Rectangle, srcRect image.Rectangle) {
	srcX := srcRect.Min.X
	srcY := srcRect.Min.Y
	srcW := srcRect.Dx()
	srcH := srcRect.Dy()

	dstX := dstRect.Min.X
	dstY := dstRect.Min.Y
	dstW := dstRect.Dx()
	dstH := dstRect.Dy()

	op := &ebiten.DrawImageOptions{}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			op.GeoM.Reset()

			sx := srcX
			sy := srcY
			sw := srcW / 4
			sh := srcH / 4
			dx := 0
			dy := 0
			dw := sw
			dh := sh
			switch i {
			case 1:
				sx = srcX + srcW/4
				sw = srcW / 2
				dx = srcW / 4
				dw = dstW - 2*srcW/4
			case 2:
				sx = srcX + 3*srcW/4
				dx = dstW - srcW/4
			}
			switch j {
			case 1:
				sy = srcY + srcH/4
				sh = srcH / 2
				dy = srcH / 4
				dh = dstH - 2*srcH/4
			case 2:
				sy = srcY + 3*srcH/4
				dy = dstH - srcH/4
			}

			op.GeoM.Scale(float64(dw)/float64(sw), float64(dh)/float64(sh))
			op.GeoM.Translate(float64(dx), float64(dy))
			op.GeoM.Translate(float64(dstX), float64(dstY))
			dst.DrawImage(uiImage.SubImage(image.Rect(sx, sy, sx+sw, sy+sh)).(*ebiten.Image), op)
		}
	}
}

type Button struct {
	Rect image.Rectangle
	Text string

	mouseDown bool

	onPressed func(b *Button)
}

func (b *Button) Update() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if b.Rect.Min.X <= x && x < b.Rect.Max.X && b.Rect.Min.Y <= y && y < b.Rect.Max.Y {
			b.mouseDown = true
		} else {
			b.mouseDown = false
		}
	} else {
		if b.mouseDown {
			if b.onPressed != nil {
				b.onPressed(b)
			}
		}
		b.mouseDown = false
	}
}

func (b *Button) Draw(dst *ebiten.Image) {
	t := imageTypeButton
	if b.mouseDown {
		t = imageTypeButtonPressed
	}
	drawNinePatches(dst, b.Rect, imageSrcRects[t])

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(b.Rect.Min.X+b.Rect.Max.X)/2, float64(b.Rect.Min.Y+b.Rect.Max.Y)/2)
	op.ColorScale.ScaleWithColor(color.Black)
	op.LineSpacingInPixels = lineSpacingInPixels
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(dst, b.Text, &text.GoTextFace{
		Source: uiFaceSource,
		Size:   uiFontSize,
	}, op)
}

func (b *Button) SetOnPressed(f func(b *Button)) {
	b.onPressed = f
}

const VScrollBarWidth = 16

type VScrollBar struct {
	X      int
	Y      int
	Height int

	thumbRate           float64
	thumbOffset         int
	dragging            bool
	draggingStartOffset int
	draggingStartY      int
	contentOffset       int
}

func (v *VScrollBar) thumbSize() int {
	const minThumbSize = VScrollBarWidth

	r := v.thumbRate
	if r > 1 {
		r = 1
	}
	s := int(float64(v.Height) * r)
	if s < minThumbSize {
		return minThumbSize
	}
	return s
}

func (v *VScrollBar) thumbRect() image.Rectangle {
	if v.thumbRate >= 1 {
		return image.Rectangle{}
	}

	s := v.thumbSize()
	return image.Rect(v.X, v.Y+v.thumbOffset, v.X+VScrollBarWidth, v.Y+v.thumbOffset+s)
}

func (v *VScrollBar) maxThumbOffset() int {
	return v.Height - v.thumbSize()
}

func (v *VScrollBar) ContentOffset() int {
	return v.contentOffset
}

func (v *VScrollBar) Update(contentHeight int) {
	v.thumbRate = float64(v.Height) / float64(contentHeight)

	if !v.dragging && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		tr := v.thumbRect()
		if tr.Min.X <= x && x < tr.Max.X && tr.Min.Y <= y && y < tr.Max.Y {
			v.dragging = true
			v.draggingStartOffset = v.thumbOffset
			v.draggingStartY = y
		}
	}
	if v.dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			_, y := ebiten.CursorPosition()
			v.thumbOffset = v.draggingStartOffset + (y - v.draggingStartY)
			if v.thumbOffset < 0 {
				v.thumbOffset = 0
			}
			if v.thumbOffset > v.maxThumbOffset() {
				v.thumbOffset = v.maxThumbOffset()
			}
		} else {
			v.dragging = false
		}
	}

	v.contentOffset = 0
	if v.thumbRate < 1 {
		v.contentOffset = int(float64(contentHeight) * float64(v.thumbOffset) / float64(v.Height))
	}
}

func (v *VScrollBar) Draw(dst *ebiten.Image) {
	sd := image.Rect(v.X, v.Y, v.X+VScrollBarWidth, v.Y+v.Height)
	drawNinePatches(dst, sd, imageSrcRects[imageTypeVScrollBarBack])

	if v.thumbRate < 1 {
		drawNinePatches(dst, v.thumbRect(), imageSrcRects[imageTypeVScrollBarFront])
	}
}

const (
	textBoxPaddingLeft = 8
	textBoxPaddingTop  = 4
)

type TextBox struct {
	Rect image.Rectangle
	Text string

	vScrollBar *VScrollBar
	offsetX    int
	offsetY    int
}

func (t *TextBox) AppendLine(line string) {
	if t.Text == "" {
		t.Text = line
	} else {
		t.Text += "\n" + line
	}
}

func (t *TextBox) Update() {
	if t.vScrollBar == nil {
		t.vScrollBar = &VScrollBar{}
	}
	t.vScrollBar.X = t.Rect.Max.X - VScrollBarWidth
	t.vScrollBar.Y = t.Rect.Min.Y
	t.vScrollBar.Height = t.Rect.Dy()

	_, h := t.contentSize()
	t.vScrollBar.Update(h)

	t.offsetX = 0
	t.offsetY = t.vScrollBar.ContentOffset()
}

func (t *TextBox) contentSize() (int, int) {
	h := len(strings.Split(t.Text, "\n"))*lineSpacingInPixels + textBoxPaddingTop
	return t.Rect.Dx(), h
}

func (t *TextBox) viewSize() (int, int) {
	return t.Rect.Dx() - VScrollBarWidth - textBoxPaddingLeft, t.Rect.Dy()
}

func (t *TextBox) contentOffset() (int, int) {
	return t.offsetX, t.offsetY
}

func (t *TextBox) Draw(dst *ebiten.Image) {
	drawNinePatches(dst, t.Rect, imageSrcRects[imageTypeTextBox])

	textOp := &text.DrawOptions{}
	x := -float64(t.offsetX) + textBoxPaddingLeft
	y := -float64(t.offsetY) + textBoxPaddingTop
	textOp.GeoM.Translate(x, y)
	textOp.GeoM.Translate(float64(t.Rect.Min.X), float64(t.Rect.Min.Y))
	textOp.ColorScale.ScaleWithColor(color.Black)
	textOp.LineSpacingInPixels = lineSpacingInPixels
	text.Draw(dst.SubImage(t.Rect).(*ebiten.Image), t.Text, &text.GoTextFace{
		Source: uiFaceSource,
		Size:   uiFontSize,
	}, textOp)

	t.vScrollBar.Draw(dst)
}

const (
	checkboxWidth       = 16
	checkboxHeight      = 16
	checkboxPaddingLeft = 8
)

type CheckBox struct {
	X    int
	Y    int
	Text string

	checked   bool
	mouseDown bool

	onCheckChanged func(c *CheckBox)
}

func (c *CheckBox) width() int {
	w := text.Advance(c.Text, &text.GoTextFace{
		Source: uiFaceSource,
		Size:   uiFontSize,
	})
	return checkboxWidth + checkboxPaddingLeft + int(w)
}

func (c *CheckBox) Update() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if c.X <= x && x < c.X+c.width() && c.Y <= y && y < c.Y+checkboxHeight {
			c.mouseDown = true
		} else {
			c.mouseDown = false
		}
	} else {
		if c.mouseDown {
			c.checked = !c.checked
			if c.onCheckChanged != nil {
				c.onCheckChanged(c)
			}
		}
		c.mouseDown = false
	}
}

func (c *CheckBox) Draw(dst *ebiten.Image) {
	t := imageTypeCheckBox
	if c.mouseDown {
		t = imageTypeCheckBoxPressed
	}
	r := image.Rect(c.X, c.Y, c.X+checkboxWidth, c.Y+checkboxHeight)
	drawNinePatches(dst, r, imageSrcRects[t])
	if c.checked {
		drawNinePatches(dst, r, imageSrcRects[imageTypeCheckBoxMark])
	}

	x := c.X + checkboxWidth + checkboxPaddingLeft
	y := c.Y + checkboxHeight/2
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(color.Black)
	op.LineSpacingInPixels = lineSpacingInPixels
	op.PrimaryAlign = text.AlignStart
	op.SecondaryAlign = text.AlignCenter
	text.Draw(dst, c.Text, &text.GoTextFace{
		Source: uiFaceSource,
		Size:   uiFontSize,
	}, op)
}

func (c *CheckBox) Checked() bool {
	return c.checked
}

func (c *CheckBox) SetOnCheckChanged(f func(c *CheckBox)) {
	c.onCheckChanged = f
}

type Game struct {
	button1    *Button
	button2    *Button
	checkBox   *CheckBox
	textBoxLog *TextBox
}

func NewGame() *Game {
	g := &Game{}
	g.button1 = &Button{
		Rect: image.Rect(16, 16, 144, 48),
		Text: "Button 1",
	}
	g.button2 = &Button{
		Rect: image.Rect(160, 16, 288, 48),
		Text: "Button 2",
	}
	g.checkBox = &CheckBox{
		X:    16,
		Y:    64,
		Text: "Check Box!",
	}
	g.textBoxLog = &TextBox{
		Rect: image.Rect(16, 96, 624, 464),
	}

	g.button1.SetOnPressed(func(b *Button) {
		g.textBoxLog.AppendLine("Button 1 Pressed")
	})
	g.button2.SetOnPressed(func(b *Button) {
		g.textBoxLog.AppendLine("Button 2 Pressed")
	})
	g.checkBox.SetOnCheckChanged(func(c *CheckBox) {
		msg := "Check box check changed"
		if c.Checked() {
			msg += " (Checked)"
		} else {
			msg += " (Unchecked)"
		}
		g.textBoxLog.AppendLine(msg)
	})
	return g
}

func (g *Game) Update() error {
	g.button1.Update()
	g.button2.Update()
	g.checkBox.Update()
	g.textBoxLog.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xeb, 0xeb, 0xeb, 0xff})
	g.button1.Draw(screen)
	g.button2.Draw(screen)
	g.checkBox.Draw(screen)
	g.textBoxLog.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("UI (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
