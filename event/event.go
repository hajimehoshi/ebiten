package event

import "time"

// Kind is the detailed type of the event
type Kind int

// Kind constants
const (
	// When a gamepad axis changes in value
	KindGamepadAxis = Kind(iota)
	// When a gamepad button is pressed
	KindGamepadButtonDown
	// When a gamepad button is released
	KindGamepadButtonUp
	// When a gamepad is added or removed to the system.
	KindGamepadConfiguration

	// When a keyboard key is pressed
	KindKeyDown
	// When a keyboard key is released
	KindKeyUp
	// When a character of text is typed on the keyboard.
	KindKeyCharacter

	// When a gamepad axis changes in value.
	KindMouseAxes
	// When a mouse button is pressed
	KindMouseButtonDown
	// When a mouse button is released
	KindMouseButtonUp
	// When the mouse enters the display window
	KindMouseEnter
	// When the mouse leaves the display
	KindMouseLeave

	// When the application is ready to update the next frame on the view port
	KindViewUpdate
	// When the size of the application's view port changes
	KindViewSize

	// When a touch begins
	KindTouchBegin
	// When a touch ends
	KindTouchEnd
	// When a touch moved, ie is dragged
	KindTouchMove
	// When a touch is canceled
	KindTouchCancel
)

// Source is the source, or origin of an event
type Source interface {
	Name() string
}

// Events implement this interface
type Event interface {
	// Source of the event
	Source() Source
	// Time the event was sent, in seconds after the program started
	Time() time.Time
	// Detailed type of the event
	Kind() Kind
}

// Basic is a basic event
type Basic struct {
	Any struct {
		Source
		time.Time
		Kind
	}
}

// Source returns the source of the basic event
func (b *Basic) Source() Source {
	return b.Any.Source
}

// Time stamp returns the tie stamp of the basic event
func (b *Basic) Time() time.Time {
	return b.Any.Time
}

// Kind returns the kind of the event.
func (b *Basic) Kind() Kind {
	return b.Any.Kind
}

// Code of a key press
type KeyCode int

// Modifiers of a key press
type KeyModifiers int

// Character of a key press, for KindKeyCharacter in particular
type KeyCharacter rune

// Keyboard related events
type Key struct {
	// Basic event
	Basic
	// Code of the key pressed or released
	KeyCode
	// Character typed
	KeyCharacter
	// Key board modifiers
	KeyModifiers
}

// Gamepad axis index
type GamepadAxis int

// Gamepad or mouse button index
type GamepadButton int

// Which gamepad or mouse index caused the event
type GamepadID int

// Position for a gamepad event
type GamepadPosition float32

// Gamepad related events
type Gamepad struct {
	// Basic event
	Basic
	// Which gamepad caused the event
	GamepadID
	// Which gamepad axis changed, if any
	GamepadAxis
	// which button was pressed, if any.
	GamepadButton
	// Position of the axis or button after the change
	GamepadPosition
}

// Mouse button index
type MouseButton int

// Change in position for a mouse
type MouseDelta float32

// Pressure applied for a mouse
type MousePressure float32

// Position for a mouse event
type MousePosition float32

// Mouse related events
type Mouse struct {
	// Basic event
	Basic

	// X position of the event
	X MousePosition
	// Y position of the event
	Y MousePosition
	// Z position of the event
	Z MousePosition
	// W position of the event
	W MousePosition
	// Change in X since last event
	DX MouseDelta
	// Change in Y since last event
	DY MouseDelta
	// Change in Z since last event
	DZ MouseDelta
	// Change in W since last event
	DW MouseDelta

	// Which button was pressed, if any.
	MouseButton
	// Pressure applied on the mouse click
	MousePressure
}

// View port related events.
type View struct {
	// Basic event
	Basic
	// Actual, real width of the display
	Width float32
	// Actual, real height of the display
	Height float32
	// X position of the display on the screen
	X float32
	// Y position of the display on the screen
	Y float32
}

// Touch ID button index
type TouchID int

// Change in position for a touch event
type TouchDelta float32

// Pressure applied for a touch event
type TouchPressure float32

// Position for a touch event
type TouchPosition float32

// Touch related events
type Touch struct {
	// Basic event
	Basic
	// Touch ID that caused the touch event
	TouchID
	// X position of the event
	X TouchPosition
	// Y position of the event
	Y  TouchPosition
	DX TouchDelta
	// Change in Y since last event
	DY TouchDelta

	// Is the touch event primary or not.
	Primary bool
}

// NewChannel returns a channel on which you can listen for incoming events
func NewChannel() chan<- Event {
	// TODO
	return nil
}
