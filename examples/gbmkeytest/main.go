// gbmkeytest is a minimal Ebiten program to exercise the no-window-system
// backends (gbm / fbdev) and their input on a handheld: it fills the screen
// with a slowly changing color to show frames are presenting, and lists the
// keyboard keys and gamepad buttons/axes currently active to show input is
// delivered without a window system.
package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type game struct {
	frame  int
	recent []string
}

func (g *game) Update() error {
	g.frame++
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		g.note("key " + strings.TrimPrefix(k.String(), "Key"))
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		for _, b := range inpututil.AppendJustPressedGamepadButtons(id, nil) {
			g.note(fmt.Sprintf("pad%d btn%d", id, b))
		}
	}
	return nil
}

func (g *game) note(s string) {
	g.recent = append(g.recent, fmt.Sprintf("#%d %s", g.frame, s))
	if len(g.recent) > 14 {
		g.recent = g.recent[len(g.recent)-14:]
	}
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: uint8(g.frame % 256), G: 40, B: uint8(255 - g.frame%256), A: 255})

	var lines []string
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	lines = append(lines, fmt.Sprintf("ebiten GBM/KMS input test  %dx%d  frame=%d", w, h, g.frame))

	// keyboard
	var held []string
	for _, k := range inpututil.AppendPressedKeys(nil) {
		held = append(held, strings.TrimPrefix(k.String(), "Key"))
	}
	lines = append(lines, "keys held: "+strings.Join(held, " "))

	// gamepads
	ids := ebiten.AppendGamepadIDs(nil)
	lines = append(lines, fmt.Sprintf("gamepads: %d", len(ids)))
	for _, id := range ids {
		var btns []string
		for b := 0; b < ebiten.GamepadButtonCount(id); b++ {
			if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(b)) {
				btns = append(btns, fmt.Sprintf("%d", b))
			}
		}
		var axes []string
		for a := 0; a < ebiten.GamepadAxisCount(id); a++ {
			if v := ebiten.GamepadAxisValue(id, ebiten.GamepadAxisType(a)); v > 0.5 || v < -0.5 {
				axes = append(axes, fmt.Sprintf("a%d=%.1f", a, v))
			}
		}
		lines = append(lines, fmt.Sprintf(" pad%d %q std=%v btns[%s] %s",
			id, ebiten.GamepadName(id), ebiten.IsStandardGamepadLayoutAvailable(id),
			strings.Join(btns, ","), strings.Join(axes, " ")))
	}

	lines = append(lines, "", "recent:")
	lines = append(lines, g.recent...)
	ebitenutil.DebugPrint(screen, strings.Join(lines, "\n"))
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowTitle("gbm input test")
	if err := ebiten.RunGame(&game{}); err != nil {
		panic(err)
	}
}
