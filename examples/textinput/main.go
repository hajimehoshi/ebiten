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

	text           string
	selectionStart int
	selectionEnd   int
	focused        bool

	composer            textinput.Composer
	composition         string
	compositionSelStart int

	// imeTextStart is the absolute byte offset in t.text where the
	// TextBeforeCaret passed to the IME begins. Used to translate the
	// IME's replacement range into an absolute range.
	imeTextStart int
}

func NewTextField(bounds image.Rectangle, multilines bool) *TextField {
	t := &TextField{
		bounds:     bounds,
		multilines: multilines,
	}
	t.composer.OnNewSession = t.onNewIMESession
	t.composer.OnComposition = t.onIMEComposition
	t.composer.OnCommit = t.onIMECommit
	return t
}

func (t *TextField) onNewIMESession() *textinput.SessionOptions {
	before, after := t.lineAroundSelection()
	t.imeTextStart = t.selectionStart - len(before)
	return &textinput.SessionOptions{
		CaretBounds:     t.caretBounds(),
		TextBeforeCaret: before,
		TextAfterCaret:  after,
	}
}

func (t *TextField) onIMEComposition(c textinput.Composition) {
	t.composition = c.Text
	t.compositionSelStart = c.SelectionStartInBytes
}

// onIMECommit applies a committed IME state: it removes the IME's
// requested replacement range (translated from IME-text-relative to
// absolute offsets), then inserts the committed text at the caret.
func (t *TextField) onIMECommit(c textinput.Commit) {
	if c.ReplacementEndInBytes > c.ReplacementStartInBytes {
		absStart := t.imeTextStart + c.ReplacementStartInBytes
		absEnd := t.imeTextStart + c.ReplacementEndInBytes
		// Adjust the selection so it tracks across the removal.
		shift := func(p int) int {
			switch {
			case p <= absStart:
				return p
			case p >= absEnd:
				return p - (absEnd - absStart)
			default:
				return absStart
			}
		}
		t.text = t.text[:absStart] + t.text[absEnd:]
		t.selectionStart = shift(t.selectionStart)
		t.selectionEnd = shift(t.selectionEnd)
	}
	t.replaceSelection(c.Text)
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
	txt := t.text
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
	t.focused = true
}

func (t *TextField) Blur() {
	t.focused = false
}

func (t *TextField) IsFocused() bool {
	return t.focused
}

// lineAroundSelection returns the bytes of the current line on either side of
// the selection start. They form CaretContext for the IME.
func (t *TextField) lineAroundSelection() (before, after string) {
	lineStart := strings.LastIndexByte(t.text[:t.selectionStart], '\n') + 1
	rel := strings.IndexByte(t.text[t.selectionStart:], '\n')
	var lineEnd int
	if rel < 0 {
		lineEnd = len(t.text)
	} else {
		lineEnd = t.selectionStart + rel
	}
	return t.text[lineStart:t.selectionStart], t.text[t.selectionStart:lineEnd]
}

func (t *TextField) replaceSelection(s string) {
	t.text = t.text[:t.selectionStart] + s + t.text[t.selectionEnd:]
	t.selectionStart += len(s)
	t.selectionEnd = t.selectionStart
}

func (t *TextField) caretBounds() image.Rectangle {
	cx, cy := t.cursorPos()
	px, py := textFieldPadding()
	x0 := t.bounds.Min.X + cx + px
	y0 := t.bounds.Min.Y + cy + py
	h := int(fontFace.Metrics().HLineGap + fontFace.Metrics().HAscent + fontFace.Metrics().HDescent)
	return image.Rect(x0, y0, x0+1, y0+h)
}

func (t *TextField) Update() error {
	if !t.focused {
		t.composer.Cancel()
		return nil
	}

	handled, err := t.composer.Update()
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if t.multilines {
			t.replaceSelection("\n")
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
		if t.selectionStart != t.selectionEnd {
			t.replaceSelection("")
		} else if t.selectionStart > 0 {
			// TODO: Remove a grapheme instead of a code point.
			_, l := utf8.DecodeLastRuneInString(t.text[:t.selectionStart])
			t.text = t.text[:t.selectionStart-l] + t.text[t.selectionStart:]
			t.selectionStart -= l
			t.selectionEnd = t.selectionStart
		}
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

	return nil
}

// textForRendering returns the committed text with the active composition
// spliced in at the caret.
func (t *TextField) textForRendering() string {
	if t.composition == "" {
		return t.text
	}
	return t.text[:t.selectionStart] + t.composition + t.text[t.selectionStart:]
}

func (t *TextField) cursorPos() (int, int) {
	txt := t.textForRendering()
	caret := t.selectionStart + t.compositionSelStart
	txt = txt[:caret]
	var nlCount int
	lastNLPos := -1
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
	vector.FillRect(screen, float32(t.bounds.Min.X), float32(t.bounds.Min.Y), float32(t.bounds.Dx()), float32(t.bounds.Dy()), color.White, false)
	var clr color.Color = color.Black
	if t.focused {
		clr = color.RGBA{0, 0, 0xff, 0xff}
	}
	vector.StrokeRect(screen, float32(t.bounds.Min.X), float32(t.bounds.Min.Y), float32(t.bounds.Dx()), float32(t.bounds.Dy()), 1, clr, false)

	px, py := textFieldPadding()
	if t.focused {
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
	text.Draw(screen, t.textForRendering(), fontFace, op)
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
