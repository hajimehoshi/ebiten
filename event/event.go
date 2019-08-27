// event is a package that models events that occur during the execution of a program
package event

// Event is an interface that custom events should implement. 
// It is empty for now because there are no general methods 
// required of events yet.
type Event interface {
}

// KeyCharacter is an event that occurs when a character is actually typed on 
// the keyboard. This may be provided by an input method.
type KeyCharacter struct { 
	// Code of the key pressed or released
	Code int
	// Key board modifiers
	Modifiers int
	// Character typed
	Character rune
}

// KeyDown is an event that occurs when a key is pressed on the keyboard.
type KeyDown struct {
	// Code of the key pressed or released
	Code int
	// Key board modifiers
	Modifiers int
}

// KeyUp is an event that occurs when a key is released on the keyboard.
// The data is the same as for a KeyDown event
type KeyUp KeyDown

// GamepadAxis is for event where an axis on a game pad changes.
type GamepadAxis struct { 	
	// Which gamepad caused the event
	ID int
	// Axis of the game pad that changed, if any
	Axis int
	// Position of the axis after the change
	Position float32
}

//GamepadButtonDown is a game pad button press event.
type GamepadButtonDown struct { 
	// Which gamepad caused the event
	ID int
    // Button that was pressed, if any.
	Button int
	// Position of the button after the change.
	Position float32
}

//GamepadButtonDown is a game pad button release event.
//The data is identical to a GamePadButtonDown event.
type GamepadButtonUp GamepadButtonDown

//GamepadAttach happens when a new game pad is attached.
type GamepadAttach struct { 
	// Which gamepad caused the event
	ID int
	// Amount of axes the game pad has.
	Axes int
	// Amount of buttons it has.
	Buttons int 
}

//GamepadDetach happens when a game pad is detached.
type GamepadDetach struct { 
	// Which gamepad caused the event
	ID int
}

//MouseAxes is a mouse axis event.
type MouseAxis struct {
	// X position of the event
	X float32
	// Y position of the event
	Y float32
	// Wheel is the position of the mouse wheel
	Wheel float32
	// DeltaX is the change in X since last event
	DeltaX float32
	// DeltaY is the change in Y since last event
	DeltaY float32
	// DeltaWheelis the change in the wheel position since last event
	DeltaWheel float32
}

//MouseButtonDown is a mouse button press event.
type MouseButtonDown struct {
	// X position of the event
	X float32
	// Y position of the event
	Y float32
	// Wheel is the position of the mouse wheel
	Wheel float32
	// DeltaX is the change in X since last event
	DeltaX float32
	// DeltaY is the change in Y since last event
	DeltaY float32
	// DeltaWheelis the change in the wheel position since last event
	DeltaWheel float32
	// Button that was pressed.
	Button int
	// Pressure applied on the mouse click
	Pressure float32
}

//MouseButtonDown is a mouse button Release event.
// The data is identical to a MouseButtonDown event.
type MouseButtonUp MouseButtonDown

// MouseEnter occurs when the mouse enters the view window.
type MouseEnter struct {
	// X position of the event
	X float32
	// Y position of the event
	Y float32
	// Wheel is the position of the mouse wheel
	Wheel float32
	// DeltaX is the change in X since last mouse event
	DeltaX float32
	// DeltaY is the change in Y since last mouse event
	DeltaY float32
	// DeltaWheelis the change in the wheel position since last mouse event
	DeltaWheel float32	
}

// MouseLeave occurs when the mouse leaves the view window.
// The data is identical to MouseEnter
type MouseLeave MouseEnter

// ViewUpdate occurs when the application is ready to update 
// the next frame on the view port.
type ViewUpdate struct {
	// No data neccesary, for now.
}

// ViewSize ocurs when the size of the application's view port changes.
type ViewSize struct {
	// Actual, real width of the view
	Width float32
	// Actual, real height of the view
	Height float32
	// X position of the view on the physical screen
	X float32
	// Y position of the view on the physical screen
	Y float32
}

// TouchBegin occurs when a touch begins.
type TouchBegin struct {
	// Touch ID that caused the touch event
	ID int
	// X position of the event
	X float32
	// Y position of the event
	Y float32
	// Change in X since last touch event
	DeltaX float32
	// Change in Y since last touch event
	DeltaY float32
	// Pressure of applied touch
	Pressure float32
	// Is the touch event primary or not
	Primary bool
}

// TouchEnd occuurs when a touch ends.
// The data is the same as for a TouchBegin event.
type TouchEnd TouchBegin

 // TouchMoved occurs when a touch moved, or in other words, is dragged.
 // The data is the same as for a TouchBegin event.
type TouchMoved TouchBegin

// TouchCancel occurs when a touch is canceled.
type TouchCancel struct { 
	// Touch ID of the touch that is now canceled.
	ID int
}

