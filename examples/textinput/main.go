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
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/bitmapfont/v3"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var fontFace = text.NewStdFace(bitmapfont.FaceEA)

const (
	screenWidth  = 640
	screenHeight = 480
)

type TextField struct {
	bounds         image.Rectangle
	multilines     bool
	text           string
	selectionStart int
	selectionEnd   int
	focused        bool

	ch    chan textinput.State
	end   func()
	state textinput.State
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

	t.selectionStart = idx
	t.selectionEnd = idx
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
	for i, r := range t.text {
		var x0, x1 int
		currentAdvance := text.Advance(t.text[lineStart:i], fontFace)
		if lineStart < i {
			x0 = int((prevAdvance + currentAdvance) / 2)
		}
		if r == '\n' {
			x1 = int(math.MaxInt32)
		} else if i < len(t.text) {
			nextI := i + 1
			for !utf8.ValidString(t.text[i:nextI]) {
				nextI++
			}
			nextAdvance := text.Advance(t.text[lineStart:nextI], fontFace)
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

	return len(t.text), true
}

func (t *TextField) Focus() {
	t.focused = true
}

func (t *TextField) Blur() {
	t.focused = false
}

func (t *TextField) Update() {
	if !t.focused {
		if t.end != nil {
			t.end()
			t.ch = nil
			t.end = nil
			t.state = textinput.State{}
		}
		return
	}

	var processed bool

	// Text inputting can happen multiple times in one tick (1/60[s] by default).
	// Handle all of them.
	for {
		if t.ch == nil {
			x, y := t.bounds.Min.X, t.bounds.Min.Y
			cx, cy := t.cursorPos()
			px, py := textFieldPadding()
			x += cx + px
			y += cy + py + int(fontFace.Metrics().HAscent)
			t.ch, t.end = textinput.Start(x, y)
			// Start returns nil for non-supported envrionments.
			if t.ch == nil {
				return
			}
		}

	readchar:
		for {
			select {
			case state, ok := <-t.ch:
				processed = true
				if !ok {
					t.ch = nil
					t.end = nil
					t.state = textinput.State{}
					break readchar
				}
				if state.Committed {
					t.text = t.text[:t.selectionStart] + state.Text + t.text[t.selectionEnd:]
					t.selectionStart += len(state.Text)
					t.selectionEnd = t.selectionStart
					t.state = textinput.State{}
					continue
				}
				t.state = state
			default:
				break readchar
			}
		}

		if t.ch == nil {
			continue
		}

		break
	}

	if processed {
		return
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if t.multilines {
			t.text = t.text[:t.selectionStart] + "\n" + t.text[t.selectionEnd:]
			t.selectionStart += 1
			t.selectionEnd = t.selectionStart
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
		if t.selectionStart > 0 {
			// TODO: Remove a grapheme instead of a code point.
			_, l := utf8.DecodeLastRuneInString(t.text[:t.selectionStart])
			t.text = t.text[:t.selectionStart-l] + t.text[t.selectionEnd:]
			t.selectionStart -= l
		}
		t.selectionEnd = t.selectionStart
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft):
		if t.selectionStart > 0 {
			// TODO: Remove a grapheme instead of a code point.
			_, l := utf8.DecodeLastRuneInString(t.text[:t.selectionStart])
			t.selectionStart -= l
		}
		t.selectionEnd = t.selectionStart
	case inpututil.IsKeyJustPressed(ebiten.KeyRight):
		if t.selectionEnd < len(t.text) {
			// TODO: Remove a grapheme instead of a code point.
			_, l := utf8.DecodeRuneInString(t.text[t.selectionEnd:])
			t.selectionEnd += l
		}
		t.selectionStart = t.selectionEnd
	}

	if !t.multilines {
		orig := t.text
		new := strings.ReplaceAll(orig, "\n", "")
		if new != orig {
			t.selectionStart -= strings.Count(orig[:t.selectionStart], "\n")
			t.selectionEnd -= strings.Count(orig[:t.selectionEnd], "\n")
		}
	}
}

func (t *TextField) cursorPos() (int, int) {
	var nlCount int
	lastNLPos := -1
	for i, r := range t.text[:t.selectionStart] {
		if r == '\n' {
			nlCount++
			lastNLPos = i
		}
	}

	txt := t.text[lastNLPos+1 : t.selectionStart]
	if t.state.Text != "" {
		txt += t.state.Text[:t.state.CompositionSelectionStartInBytes]
	}
	x := int(text.Advance(txt, fontFace))
	y := nlCount * int(fontFace.Metrics().HLineGap+fontFace.Metrics().HAscent+fontFace.Metrics().HDescent)
	return x, y
}

func (t *TextField) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, float32(t.bounds.Min.X), float32(t.bounds.Min.Y), float32(t.bounds.Dx()), float32(t.bounds.Dy()), color.White, false)
	var clr color.Color = color.Black
	if t.focused {
		clr = color.RGBA{0, 0, 0xff, 0xff}
	}
	vector.StrokeRect(screen, float32(t.bounds.Min.X), float32(t.bounds.Min.Y), float32(t.bounds.Dx()), float32(t.bounds.Dy()), 1, clr, false)

	px, py := textFieldPadding()
	if t.focused && t.selectionStart >= 0 {
		x, y := t.bounds.Min.X, t.bounds.Min.Y
		cx, cy := t.cursorPos()
		x += px + cx
		y += py + cy
		h := int(fontFace.Metrics().HLineGap + fontFace.Metrics().HAscent + fontFace.Metrics().HDescent)
		vector.StrokeLine(screen, float32(x), float32(y), float32(x), float32(y+h), 1, color.Black, false)
	}

	shownText := t.text
	if t.focused && t.state.Text != "" {
		shownText = t.text[:t.selectionStart] + t.state.Text + t.text[t.selectionEnd:]
	}

	tx := t.bounds.Min.X + px
	ty := t.bounds.Min.Y + py
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(tx), float64(ty))
	op.ColorScale.ScaleWithColor(color.Black)
	op.LineSpacingInPixels = fontFace.Metrics().HLineGap + fontFace.Metrics().HAscent + fontFace.Metrics().HDescent
	text.Draw(screen, shownText, fontFace, op)
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

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
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
		tf.Update()
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
	if runtime.GOOS != "darwin" && runtime.GOOS != "js" {
		log.Printf("github.com/hajimehoshi/ebiten/v2/exp/textinput is not supported in this environment (GOOS=%s) yet", runtime.GOOS)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text Input (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
