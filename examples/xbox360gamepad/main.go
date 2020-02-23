package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

var (
	gamepadIDs = make(map[int]struct{})
)

// update is called once per frame.
func update(screen *ebiten.Image) error {
	for _, id := range inpututil.JustConnectedGamepadIDs() {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("Gamepad %d:%s - %s has been connected",
			id, ebiten.GamepadSDLID(id), ebiten.GamepadName(id)))
		gamepadIDs[id] = struct{}{}
	}

	for id := range gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			ebitenutil.DebugPrint(screen, fmt.Sprintf("Gamepad %d:%s - %s has been disconnected",
				id, ebiten.GamepadSDLID(id), ebiten.GamepadName(id)))
			delete(gamepadIDs, id)
		}
	}

	ids := ebiten.GamepadIDs()
	axes := make(map[int][]string)
	buttons := make(map[int][]string)

	for _, id := range ids {
		// Axes.
		input := ebiten.GamepadAxis(id, ebiten.GamepadAxisLeftX)
		axes[id] = append(axes[id], fmt.Sprintf("Gamepad axis left X: %f", input))

		input = ebiten.GamepadAxis(id, ebiten.GamepadAxisLeftY)
		axes[id] = append(axes[id], fmt.Sprintf("Gamepad axis left Y: %f", input))

		input = ebiten.GamepadAxis(id, ebiten.GamepadAxisLT)
		axes[id] = append(axes[id], fmt.Sprintf("Gamepad axis LT: %f", input))

		input = ebiten.GamepadAxis(id, ebiten.GamepadAxisRightX)
		axes[id] = append(axes[id], fmt.Sprintf("Gamepad axis right X: %f", input))

		input = ebiten.GamepadAxis(id, ebiten.GamepadAxisRightY)
		axes[id] = append(axes[id], fmt.Sprintf("Gamepad axis right Y: %f", input))

		input = ebiten.GamepadAxis(id, ebiten.GamepadAxisRT)
		axes[id] = append(axes[id], fmt.Sprintf("Gamepad axis RT: %f", input))

		// Buttons.
		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonA) {
			buttons[id] = append(buttons[id], "Button pressed: A")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonB) {
			buttons[id] = append(buttons[id], "Button pressed: B")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonX) {
			buttons[id] = append(buttons[id], "Button pressed: X")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonY) {
			buttons[id] = append(buttons[id], "Button pressed: Y")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonLB) {
			buttons[id] = append(buttons[id], "Button pressed: LB")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonRB) {
			buttons[id] = append(buttons[id], "Button pressed: RB")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonBack) {
			buttons[id] = append(buttons[id], "Button pressed: Back")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonStart) {
			buttons[id] = append(buttons[id], "Button pressed: Start")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonSystem) {
			buttons[id] = append(buttons[id], "Button pressed: System")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonLeftStick) {
			buttons[id] = append(buttons[id], "Button pressed: Left Stick")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonRightStick) {
			buttons[id] = append(buttons[id], "Button pressed: Right Stick")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonDpadUp) {
			buttons[id] = append(buttons[id], "Button pressed: Dpad Up")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonDpadRight) {
			buttons[id] = append(buttons[id], "Button pressed: Dpad Right")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonDpadDown) {
			buttons[id] = append(buttons[id], "Button pressed: Dpad Down")
		}

		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButtonDpadLeft) {
			buttons[id] = append(buttons[id], "Button pressed: Dpad Left")
		}
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	out := ""

	for _, id := range ids {
		out += fmt.Sprintf("Gamepad ID: %d SDL_ID: %s NAME: %s\n",
			id, ebiten.GamepadSDLID(id), ebiten.GamepadName(id))
		out += strings.Join(axes[id], "\n")
		out += "\n"
		out += strings.Join(buttons[id], "\n")
		out += "\n"
	}

	ebitenutil.DebugPrint(screen, out)

	return nil
}

func main() {
	err := ebiten.Run(update, screenWidth, screenHeight, 1, "Gamepad demo")
	handleError("Couldn't run the demo", err)
}

func handleError(message string, err error) {
	if err != nil {
		log.Fatalf("%s: %s\n", message, err)
	}
}
