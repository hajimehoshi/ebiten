// Copyright 2015 Hajime Hoshi
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
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	gamepadIDsBuf  []ebiten.GamepadID
	gamepadIDs     map[ebiten.GamepadID]struct{}
	axes           map[ebiten.GamepadID][]string
	pressedButtons map[ebiten.GamepadID][]string
}

func (g *Game) Update() error {
	if g.gamepadIDs == nil {
		g.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	}

	// Log the gamepad connection events.
	g.gamepadIDsBuf = inpututil.AppendJustConnectedGamepadIDs(g.gamepadIDsBuf[:0])
	for _, id := range g.gamepadIDsBuf {
		log.Printf("gamepad connected: id: %d, SDL ID: %s", id, ebiten.GamepadSDLID(id))
		g.gamepadIDs[id] = struct{}{}
	}
	for id := range g.gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			log.Printf("gamepad disconnected: id: %d", id)
			delete(g.gamepadIDs, id)
		}
	}

	g.axes = map[ebiten.GamepadID][]string{}
	g.pressedButtons = map[ebiten.GamepadID][]string{}
	for id := range g.gamepadIDs {
		maxAxis := ebiten.GamepadAxisCount(id)
		for a := 0; a < maxAxis; a++ {
			v := ebiten.GamepadAxisValue(id, a)
			g.axes[id] = append(g.axes[id], fmt.Sprintf("%d:%+0.2f", a, v))
		}

		maxButton := ebiten.GamepadButton(ebiten.GamepadButtonCount(id))
		for b := ebiten.GamepadButton(0); b < maxButton; b++ {
			if ebiten.IsGamepadButtonPressed(id, b) {
				g.pressedButtons[id] = append(g.pressedButtons[id], strconv.Itoa(int(b)))
			}

			// Log button events.
			if inpututil.IsGamepadButtonJustPressed(id, b) {
				log.Printf("button pressed: id: %d, button: %d", id, b)
			}
			if inpututil.IsGamepadButtonJustReleased(id, b) {
				log.Printf("button released: id: %d, button: %d", id, b)
			}
		}

		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			for b := ebiten.StandardGamepadButton(0); b <= ebiten.StandardGamepadButtonMax; b++ {
				// Log button events.
				if inpututil.IsStandardGamepadButtonJustPressed(id, b) {
					var strong float64
					var weak float64
					switch b {
					case ebiten.StandardGamepadButtonLeftTop,
						ebiten.StandardGamepadButtonLeftLeft,
						ebiten.StandardGamepadButtonLeftRight,
						ebiten.StandardGamepadButtonLeftBottom:
						weak = 0.5
					case ebiten.StandardGamepadButtonRightTop,
						ebiten.StandardGamepadButtonRightLeft,
						ebiten.StandardGamepadButtonRightRight,
						ebiten.StandardGamepadButtonRightBottom:
						strong = 0.5
					}
					if strong > 0 || weak > 0 {
						op := &ebiten.VibrateGamepadOptions{
							Duration:        200 * time.Millisecond,
							StrongMagnitude: strong,
							WeakMagnitude:   weak,
						}
						ebiten.VibrateGamepad(id, op)
					}
					log.Printf("standard button pressed: id: %d, button: %d", id, b)
				}
				if inpututil.IsStandardGamepadButtonJustReleased(id, b) {
					log.Printf("standard button released: id: %d, button: %d", id, b)
				}
			}
		}
	}
	return nil
}

var standardButtonToString = map[ebiten.StandardGamepadButton]string{
	ebiten.StandardGamepadButtonRightBottom:      "RB",
	ebiten.StandardGamepadButtonRightRight:       "RR",
	ebiten.StandardGamepadButtonRightLeft:        "RL",
	ebiten.StandardGamepadButtonRightTop:         "RT",
	ebiten.StandardGamepadButtonFrontTopLeft:     "FTL",
	ebiten.StandardGamepadButtonFrontTopRight:    "FTR",
	ebiten.StandardGamepadButtonFrontBottomLeft:  "FBL",
	ebiten.StandardGamepadButtonFrontBottomRight: "FBR",
	ebiten.StandardGamepadButtonCenterLeft:       "CL",
	ebiten.StandardGamepadButtonCenterRight:      "CR",
	ebiten.StandardGamepadButtonLeftStick:        "LS",
	ebiten.StandardGamepadButtonRightStick:       "RS",
	ebiten.StandardGamepadButtonLeftBottom:       "LB",
	ebiten.StandardGamepadButtonLeftRight:        "LR",
	ebiten.StandardGamepadButtonLeftLeft:         "LL",
	ebiten.StandardGamepadButtonLeftTop:          "LT",
	ebiten.StandardGamepadButtonCenterCenter:     "CC",
}

func standardMap(id ebiten.GamepadID) string {
	m := `       [FBL ]                    [FBR ]
       [FTL ]                    [FTR ]

       [LT  ]       [CC  ]       [RT  ]
    [LL  ][LR  ] [CL  ][CR  ] [RL  ][RR  ]
       [LB  ]                    [RB  ]
             [LS  ]       [RS  ]
`

	for b, str := range standardButtonToString {
		placeholder := "[" + str + strings.Repeat(" ", 4-len(str)) + "]"
		v := ebiten.StandardGamepadButtonValue(id, b)
		switch {
		case !ebiten.IsStandardGamepadButtonAvailable(id, b):
			m = strings.Replace(m, placeholder, "  --  ", 1)
		case ebiten.IsStandardGamepadButtonPressed(id, b):
			m = strings.Replace(m, placeholder, fmt.Sprintf("[%0.2f]", v), 1)
		default:
			m = strings.Replace(m, placeholder, fmt.Sprintf(" %0.2f ", v), 1)
		}
	}

	// TODO: Use ebiten.IsStandardGamepadAxisAvailable
	m += fmt.Sprintf("    Left Stick:  X: %+0.2f, Y: %+0.2f\n    Right Stick: X: %+0.2f, Y: %+0.2f",
		ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal),
		ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical),
		ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickHorizontal),
		ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickVertical))
	return m
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the current gamepad status.
	str := ""
	if len(g.gamepadIDs) > 0 {
		ids := make([]ebiten.GamepadID, 0, len(g.gamepadIDs))
		for id := range g.gamepadIDs {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(a, b int) bool {
			return ids[a] < ids[b]
		})
		for _, id := range ids {
			var standard string
			if ebiten.IsStandardGamepadLayoutAvailable(id) {
				standard = " (Standard Layout)"
			}
			str += fmt.Sprintf("Gamepad (ID: %d, SDL ID: %s)%s:\n", id, ebiten.GamepadSDLID(id), standard)
			str += fmt.Sprintf("  Name:    %s\n", ebiten.GamepadName(id))
			str += fmt.Sprintf("  Axes:    %s\n", strings.Join(g.axes[id], ", "))
			str += fmt.Sprintf("  Buttons: %s\n", strings.Join(g.pressedButtons[id], ", "))
			if ebiten.IsStandardGamepadLayoutAvailable(id) {
				str += "\n"
				str += standardMap(id) + "\n"
			}
			str += "\n"
		}
	} else {
		str = "Please connect your gamepad."
	}
	ebitenutil.DebugPrint(screen, str)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Gamepad (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
