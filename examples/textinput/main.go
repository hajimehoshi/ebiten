// Copyright 2023 The Ebitengine Authors
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
	"image"
	"image/color"
	"log"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/bitmapfont/v4"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var fontFace = text.NewGoXFace(bitmapfont.FaceEA)

const (
	screenWidth  = 640
	screenHeight = 480
)

type TextField struct {
	bounds     image.Rectangle
	multilines bool
	field      textinput.Field
}

func NewTextField(bounds image.Rectangle, multilines bool) *TextField {
	return &TextField{
		bounds:     bounds,
		multilines: multilines,
	}
}

func (t *TextField) Contains(x, y int) bool {
	return image.Pt(x, y).In(t.bounds)
}

func (t *TextField) SetSelectionStartByCursorPosition(x, y int) bool {
	idx, ok := t.textIndexByCursorPosition(x, y)
	if !ok {
		return false
	}
	t.field.SetSelection(idx, idx)
	return true
}

func (t *TextField) textIndexByCursorPosition(x, y int) (int, bool) {
	if !t.Contains(x, y) {
		return 0, false
	}

	x -= t.bounds.Min.X
	y -= t.bounds.Min.Y
	px, py := textFieldPadding()
	x -= px
	y -= py
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	lineSpacingInPixels := int(fontFace.Metrics().HLineGap + fontFace.Metrics().HAscent + fontFace.Metrics().HDescent)
	var nlCount int
	var lineStart int
	var prevAdvance float64
	txt := t.field.Text()
	for i, r := range txt {
		var x0, x1 int
		currentAdvance := text.Advance(txt[lineStart:i], fontFace)
		if lineStart < i {
			x0 = int((prevAdvance + currentAdvance) / 2)
		}
		if r == '\n' {
			x1 = int(math.MaxInt32)
		} else if i < len(txt) {
			nextI := i + 1
			for !utf8.ValidString(txt[i:nextI]) {
				nextI++
			}
			nextAdvance := text.Advance(txt[lineStart:nextI], fontFace)
			x1 = int((currentAdvance + nextAdvance) / 2)
		} else {
			x1 = int(currentAdvance)
		}
		if x0 <= x && x < x1 && nlCount*lineSpacingInPixels <= y && y < (nlCount+1)*lineSpacingInPixels {
			return i, true
		}
		prevAdvance = currentAdvance

		if r == '\n' {
			nlCount++
			lineStart = i + 1
			prevAdvance = 0
		}
	}

	return len(txt), true
}

func (t *TextField) Focus() {
	t.field.Focus()
}

func (t *TextField) Blur() {
	t.field.Blur()
}

func (t *TextField) Update() error {
	if !t.field.IsFocused() {
		return nil
	}

	x, y := t.bounds.Min.X, t.bounds.Min.Y
	cx, cy := t.cursorPos()
	px, py := textFieldPadding()
	x0 := x + cx + px
	x1 := x0 + 1
	y0 := y + cy + py
	y1 := y0 + int(fontFace.Metrics().HLineGap+fontFace.Metrics().HAscent+fontFace.Metrics().HDescent)
	handled, err := t.field.HandleInputWithBounds(image.Rect(x0, y0, x1, y1))
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if t.multilines {
			text := t.field.Text()
			selectionStart, selectionEnd := t.field.Selection()
			text = text[:selectionStart] + "\n" + text[selectionEnd:]
			selectionStart += len("\n")
			selectionEnd = selectionStart
			t.field.SetTextAndSelection(text, selectionStart, selectionEnd)
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
		text := t.field.Text()
		selectionStart, selectionEnd := t.field.Selection()
		if selectionStart != selectionEnd {
			text = text[:selectionStart] + text[selectionEnd:]
		} else if selectionStart > 0 {
			// TODO: Remove a grapheme instead of a code point.
			_, l := utf8.DecodeLastRuneInString(text[:selectionStart])
			text = text[:selectionStart-l] + text[selectionEnd:]
			selectionStart -= l
		}
		selectionEnd = selectionStart
		t.field.SetTextAndSelection(text, selectionStart, selectionEnd)
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft):
		text := t.field.Text()
		selectionStart, _ := t.field.Selection()
		if selectionStart > 0 {
			// TODO: Remove a grapheme instead of a code point.
			_, l := utf8.DecodeLastRuneInString(text[:selectionStart])
			selectionStart -= l
		}
		t.field.SetTextAndSelection(text, selectionStart, selectionStart)
	case inpututil.IsKeyJustPressed(ebiten.KeyRight):
		text := t.field.Text()
		_, selectionEnd := t.field.Selection()
		if selectionEnd < len(text) {
			// TODO: Remove a grapheme instead of a code point.
			_, l := utf8.DecodeRuneInString(text[selectionEnd:])
			selectionEnd += l
		}
		t.field.SetTextAndSelection(text, selectionEnd, selectionEnd)
	}

	if !t.multilines {
		orig := t.field.Text()
		new := strings.ReplaceAll(orig, "\n", "")
		if new != orig {
			selectionStart, selectionEnd := t.field.Selection()
			selectionStart -= strings.Count(orig[:selectionStart], "\n")
			selectionEnd -= strings.Count(orig[:selectionEnd], "\n")
			t.field.SetSelection(selectionStart, selectionEnd)
		}
	}

	return nil
}

func (t *TextField) cursorPos() (int, int) {
	var nlCount int
	lastNLPos := -1
	txt := t.field.TextForRendering()
	selectionStart, _ := t.field.Selection()
	if s, _, ok := t.field.CompositionSelection(); ok {
		selectionStart += s
	}
	txt = txt[:selectionStart]
	for i, r := range txt {
		if r == '\n' {
			nlCount++
			lastNLPos = i
		}
	}

	txt = txt[lastNLPos+1:]
	x := int(text.Advance(txt, fontFace))
	y := nlCount * int(fontFace.Metrics().HLineGap+fontFace.Metrics().HAscent+fontFace.Metrics().HDescent)
	return x, y
}

func (t *TextField) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, float32(t.bounds.Min.X), float32(t.bounds.Min.Y), float32(t.bounds.Dx()), float32(t.bounds.Dy()), color.White, false)
	var clr color.Color = color.Black
	if t.field.IsFocused() {
		clr = color.RGBA{0, 0, 0xff, 0xff}
	}
	vector.StrokeRect(screen, float32(t.bounds.Min.X), float32(t.bounds.Min.Y), float32(t.bounds.Dx()), float32(t.bounds.Dy()), 1, clr, false)

	px, py := textFieldPadding()
	selectionStart, _ := t.field.Selection()
	if t.field.IsFocused() && selectionStart >= 0 {
		x, y := t.bounds.Min.X, t.bounds.Min.Y
		cx, cy := t.cursorPos()
		x += px + cx
		y += py + cy
		h := int(fontFace.Metrics().HLineGap + fontFace.Metrics().HAscent + fontFace.Metrics().HDescent)
		vector.StrokeLine(screen, float32(x), float32(y), float32(x), float32(y+h), 1, color.Black, false)
	}

	tx := t.bounds.Min.X + px
	ty := t.bounds.Min.Y + py
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(tx), float64(ty))
	op.ColorScale.ScaleWithColor(color.Black)
	op.LineSpacing = fontFace.Metrics().HLineGap + fontFace.Metrics().HAscent + fontFace.Metrics().HDescent
	text.Draw(screen, t.field.TextForRendering(), fontFace, op)
}

const textFieldHeight = 24

func textFieldPadding() (int, int) {
	m := fontFace.Metrics()
	return 4, (textFieldHeight - int(m.HLineGap+m.HAscent+m.HDescent)) / 2
}

type Game struct {
	textFields []*TextField
}

func (g *Game) Update() error {
	if g.textFields == nil {
		g.textFields = append(g.textFields, NewTextField(image.Rect(16, 16, screenWidth-16, 16+textFieldHeight), false))
		g.textFields = append(g.textFields, NewTextField(image.Rect(16, 48, screenWidth-16, 48+textFieldHeight), false))
		g.textFields = append(g.textFields, NewTextField(image.Rect(16, 80, screenWidth-16, screenHeight-16), true))
	}

	ids := inpututil.AppendJustPressedTouchIDs(nil)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(ids) > 0 {
		var x, y int
		if len(ids) > 0 {
			x, y = ebiten.TouchPosition(ids[0])
		} else {
			x, y = ebiten.CursorPosition()
		}
		for _, tf := range g.textFields {
			if tf.Contains(x, y) {
				tf.Focus()
				tf.SetSelectionStartByCursorPosition(x, y)
			} else {
				tf.Blur()
			}
		}
	}

	for _, tf := range g.textFields {
		if err := tf.Update(); err != nil {
			return err
		}
	}

	x, y := ebiten.CursorPosition()
	var inTextField bool
	for _, tf := range g.textFields {
		if tf.Contains(x, y) {
			inTextField = true
			break
		}
	}
	if inTextField {
		ebiten.SetCursorShape(ebiten.CursorShapeText)
	} else {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xcc, 0xcc, 0xcc, 0xff})
	for _, tf := range g.textFields {
		tf.Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text Input (Ebitengine Demo)")
	op := &ebiten.RunGameOptions{}
	op.ApplePressAndHoldEnabled = true
	if err := ebiten.RunGameWithOptions(&Game{}, op); err != nil {
		log.Fatal(err)
	}
}
