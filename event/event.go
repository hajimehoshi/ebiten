// Copyright 2019 The Ebiten Authors
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

// Package event is a package that models events that occur during
// the execution of a program.
package event

// Event is an interface that custom events should implement.
// It is empty for now because there are no general methods
// required of events yet.
type Event interface {
}

// KeyCharacter is an event that occurs when a character is actually typed on
// the keyboard. This may be provided by an input method.
type KeyCharacter struct {
	// Code of the key typed.
	// TODO: this should change later from an int to an enumeration type.
	Code int
	// Modifiers are the modifiers pressed together with the key.
	// TODO: this should change later from an int to an enumeration type.
	Modifiers int
	// Character typed.
	Character rune
}

// KeyDown is an event that occurs when a key is pressed on the keyboard.
type KeyDown struct {
	// Code of the key pressed or released.
	// TODO: this should change later from an int to an enumeration type.
	Code int
	// Modifiers are the modifiers pressed together with the key.
	// TODO: this should change later from an int to an enumeration type.
	Modifiers int
}

// KeyUp is an event that occurs when a key is released on the keyboard.
// The data is the same as for a KeyDown event.
type KeyUp KeyDown

// GamepadAxis is for event where an axis on a gamepad changes.
type GamepadAxis struct {
	// ID represents which gamepad caused the event.
	ID int
	// Axis of the gamepad that changed.
	Axis int
	// Position of the axis after the change.
	// It varies between -1.0 and 1.0.
	Position float32
}

// GamepadButtonDown is a gamepad button press event.
type GamepadButtonDown struct {
	// ID represents which gamepad caused the event.
	ID int
	// Button that was pressed, if any.
	Button int
	// Pressure applied to the gamepad button.
	// It varies between 0.0 for not pressed, and 1.0 for completely pressed.
	Pressure float32
}

// GamepadButtonDown is a gamepad button release event.
// The data is identical to a GamePadButtonDown event.
type GamepadButtonUp GamepadButtonDown

// GamepadAttach happens when a new gamepad is attached.
type GamepadAttach struct {
	// ID represents which gamepad caused the event.
	ID int
	// Axes represents the amount of axes the gamepad has.
	Axes int
	// Buttons represents the amount of buttons the gamepad has.
	Buttons int
}

// GamepadDetach happens when a gamepad is detached.
type GamepadDetach struct {
	// ID represents which gamepad caused the event.
	ID int
}

// MouseAxes is a mouse axis event.
type MouseAxis struct {
	// X position of the mouse pointer.
	X float32
	// Y position of the mouse pointer.
	Y float32
	// DeltaX is the change in X since the last mouse event.
	DeltaX float32
	// DeltaY is the change in Y since the last mouse event.
	DeltaY float32
}

// MouseWheel is a mouse wheel event.
type MouseWheel struct {
	// Vertical is the vertical position of the mouse wheel.
	Vertical float32
	// Horizontal is the horizontal position of the mouse wheel.
	Horizontal float32
	// DeltaVertical is the change in Vertical since the last MouseWheel event.
	DeltaVertical float32
	// DeltaHorizontal is the change in Horizontal since the last MouseWheel event.
	DeltaHorizontal float32
}

// MouseButtonDown is a mouse button press event.
type MouseButtonDown struct {
	// X position of the mouse pointer.
	X float32
	// Y position of the mouse pointer.
	Y float32
	// DeltaX is the change in X since the last mouse event.
	DeltaX float32
	// DeltaY is the change in Y since the last mouse event.
	DeltaY float32
	// Button that was pressed.
	// TODO: this should change later from an int to an enumeration type
	Button int
	// Pressure applied on the mouse click.
	// It varies between 0.0 for not pressed, and 1.0 for completely pressed.
	Pressure float32
}

// MouseButtonDown is a mouse button Release event.
// The data is identical to a MouseButtonDown event.
type MouseButtonUp MouseButtonDown

// MouseEnter occurs when the mouse enters the view window.
type MouseEnter struct {
	// X position of the mouse pointer.
	X float32
	// Y position of the mouse pointer.
	Y float32
	// DeltaX is the change in X since last mouse event.
	DeltaX float32
	// DeltaY is the change in Y since last mouse event.
	DeltaY float32
}

// MouseLeave occurs when the mouse leaves the view window.
// The data is identical to MouseEnter
type MouseLeave MouseEnter

// ViewUpdate occurs when the application is ready to update
// the next frame on the view port.
type ViewUpdate struct {
	// No data neccesary, for now.
}

// ViewSize occurs when the size of the application's view port changes.
type ViewSize struct {
	// Width is the actual, real width of the view.
	Width float32
	// Height is the actual, real height of the view.
	Height float32
	// X position of the view on the physical screen.
	X float32
	// Y position of the view on the physical screen.
	Y float32
}

// TouchBegin occurs when a touch begins.
type TouchBegin struct {
	// ID of the touch that caused the touch event.
	ID int
	// X position of the event.
	X float32
	// Y position of the event.
	Y float32
	// Pressure of applied touch.
	Pressure float32
	// Primary represents whether the touch event is the primary touch or not.
	Primary bool
}

// TouchMoved occurs when a touch moved, or in other words, is dragged.
type TouchMoved struct {
	// ID of the touch that caused the touch event.
	ID int
	// X position of the event.
	X float32
	// Y position of the event.
	Y float32
	// DeltaX is the change in X since last touch event.
	DeltaX float32
	// Deltay is the change in Y since last touch event.
	DeltaY float32
	// Pressure of applied touch.
	Pressure float32
	// Primary represents whether the touch event is the primary touch or not.
	Primary bool
}

// TouchEnd occurs when a touch ends.
// The data is the same as for a TouchMoved event.
type TouchEnd TouchMoved

// TouchCancel occurs when a touch is canceled.
type TouchCancel struct {
	// ID of the touch that caused the touch event.
	ID int
}
