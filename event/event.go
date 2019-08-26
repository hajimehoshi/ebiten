package event

// Events implement this interface
type Event interface {
	// Marker method, to avoid non-event accidentaly being used as events.
	IsEvent()
}

// BasicEvent is a basic event
type Basic struct { }

// Implement the event marker method
func (b *Basic) IsEvent() {
}

// Keyboard related events
type Key struct {
	// Basic event
	Basic
	// Code of the key pressed or released
	Code int
	// Key board modifiers
	Modifiers int
}

type KeyCharacter struct { 
	// Key, as this is a key event
	Key
	// Character typed
	Character rune
}

type KeyDown struct {
	// Key, as this is a key event
	Key
}

type KeyUp struct {
	// Key, as this is a key event
	Key
}

// Gamepad related events
type Gamepad struct {
	// Basic event
	Basic
	// Which gamepad caused the event
	ID int
}

// GamePadAxis is for event where an axis on a game pad changes.
type GamepadAxes struct { 	
	// Gamepad, because this  a game pad event
	Gamepad
	// Axis the game pad that changed, if any
	Axis int
	// Position of the axis after the change
	Position float32
}

// GamePadAxis is for event where a button on a game pad changes.
type GamepadButton struct { 
	// Gamepad, because this is a game pad event
        Gamepad
        // Button that was pressed, if any.
	Button int
	// Position of the button after the change.
	Position float32
}


type GamepadButtonDown struct { 
	// GamepadButton because it is a game pad button event
	GamepadButton
}

type GamepadButtonUp struct { 
	// GamepadButton because it is a game pad button event
	GamepadButton
}

type GamepadConfiguration struct { 
	// Gamepad because it is a game pad related
	Gamepad
}

// Mouse related events
type Mouse struct {
	// Basic event
	Basic

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


//MouseAxes is a mouse axis event
type MouseAxis struct {
	// Mouse because this is a mouse event
	Mouse
}

type MouseButton struct {
	// Mouse because this is a mouse event
	Mouse
	// Button that was pressed.
	Button int
	// Pressure applied on the mouse click
	Pressure float32
}

type MouseButtonDown struct {
	//MouseButton because this is a mouse button event
	MouseButton
}

type MouseButtonUp struct {
	//MouseButton because this is a mouse button event
	MouseButton
}

// When the mouse enters the view window
type MouseEnter struct {
	// Mouse because this is a mouse event
	Mouse
}

// When the mouse leaves the view window
type MouseLeave struct {
	// Mouse because this is a mouse event
	Mouse
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

// When the application is ready to update the next frame on the view port
type ViewUpdate struct {
	// View because this is a view event
	View
}

// When the size of the application's view port changes
type ViewSize struct {
	// View because this is a view event
	View
}


// Touch related events
type Touch struct {
	// Basic event
	Basic
	// Touch ID that caused the touch event
	ID int
	// X position of the event
	X float32
	// Y position of the event
	Y float32
	// Change in X since last event
	DeltaX float32
	// Change in Y since last event
	DeltaY float32
	// Pressure of applied touch
	Pressure float32

	// Is the touch event primary or not.
	Primary bool
}

// When a touch begins
type TouchBegin struct {
	// Touch because this is a touch event
	Touch
}

// When a touch ends
type TouchEnd struct {
	// Touch because this is a touch event
	Touch
}

 // When a touch moved, ie is dragged
type TouchMoved struct {
	Touch
}

// When a touch is canceled
type TouchCancel struct { 
	Touch
}


