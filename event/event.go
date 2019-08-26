package event

// Events implement this interface. It is empty for now
// because there are no general methods required of events yet.
type Event interface {
}



// key is data for keyboard related events
type key struct {
	// Code of the key pressed or released
	Code int
	// Key board modifiers
	Modifiers int
}

type KeyCharacter struct { 
	// Key, as this is a key event
	key
	// Character typed
	Character rune
}

func NewKeyCharacter(code, modifiers int, character rune) KeyCharacter {
    return KeyCharacter{key:key{Code: code, Modifiers: modifiers}, Character: character}
} 

type KeyDown struct {
	// Key, as this is a key event
	key
}

func NewKeyDown(code, modifiers int) KeyDown {
    return KeyDown{key:key{Code: code, Modifiers: modifiers}}
} 

type KeyUp struct {
	// Key, as this is a key event
	key
}

func NewKeyUp(code, modifiers int) KeyUp {
    return KeyUp{key:key{Code: code, Modifiers: modifiers}}
} 

// Gamepad related event data
type gamepad struct {
	// Which gamepad caused the event
	ID int
}

// GamePadAxis is for event where an axis on a game pad changes.
type GamepadAxis struct { 	
	// gamepad, because this  a game pad event
	gamepad
	// Axis the game pad that changed, if any
	Axis int
	// Position of the axis after the change
	Position float32
}

func NewGamePadAxis(id, axis int, position float32) GamepadAxis {
    return GamepadAxis{gamepad:gamepad{ID: id}, Axis: axis, Position: position}
} 

// GamePadAxis is data for event where a button on a game pad changes.
type gamepadButton struct { 
	// gamepad, because this is a game pad event
    gamepad
    // Button that was pressed, if any.
	Button int
	// Position of the button after the change.
	Position float32
}

func newGamepadButton(id, button int, position float32) gamepadButton {
    return gamepadButton{gamepad:gamepad{ID: id}, Button: button, Position: position}
}

type GamepadButtonDown struct { 
	// GamepadButton because it is a game pad button event
	gamepadButton
}

func NewGamepadButtonDown(id, button int, position float32) GamepadButtonDown {
    return GamepadButtonDown{gamepadButton:newGamepadButton(id, button, position)}
} 


type GamepadButtonUp struct { 
	// GamepadButton because it is a game pad button event
	gamepadButton
}

func NewGamepadButtonUp(id, button int, position float32) GamepadButtonUp {
    return GamepadButtonUp{gamepadButton:newGamepadButton(id, button, position)}
} 

type GamepadConfiguration struct { 
	// gamepad because it is a game pad related
	gamepad
}

func NewGamepadConfiguration(id int) GamepadConfiguration {
    return GamepadConfiguration{gamepad:gamepad{ID: id}}
} 


// mouse related event data
type mouse struct {
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

func newMouse(x, y, wheel, deltax, deltay, deltawheel float32) mouse {
    return mouse{X:x, Y:y, Wheel: wheel, 
        DeltaX: deltax, DeltaY: deltay, DeltaWheel: deltawheel}
}

//MouseAxes is a mouse axis event
type MouseAxis struct {
	// Mouse because this is a mouse event
	mouse
}

func NewMouseAxis(x, y, wheel, deltax, deltay, deltawheel float32) MouseAxis {
    return MouseAxis{ mouse: newMouse(x, y, wheel, deltax, deltay, deltawheel) }
}

type mouseButton struct {
    // Mouse event data
    mouse
	// Button that was pressed.
	Button int
	// Pressure applied on the mouse click
	Pressure float32
}

func newMouseButton(x, y, wheel, deltax, deltay, deltawheel float32, button int, pressure float32) mouseButton {
    return mouseButton{ 
        mouse: newMouse(x, y, wheel, deltax, deltay, deltawheel), 
        Button: button,
        Pressure: pressure,
    }
}


type MouseButtonDown struct {
	//MouseButton because this is a mouse button event
	mouseButton
}

func NewMouseButtonDown(x, y, wheel, deltax, deltay, deltawheel float32, button int, pressure float32) MouseButtonDown {
    return MouseButtonDown{mouseButton: 
        newMouseButton(x, y, wheel, deltax, deltay, deltawheel, button, pressure),
    }
}


type MouseButtonUp struct {
	//MouseButton because this is a mouse button event
	mouseButton
}

func NewMouseButtonUp(x, y, wheel, deltax, deltay, deltawheel float32, button int, pressure float32) MouseButtonDown {
    return MouseButtonDown{mouseButton: 
        newMouseButton(x, y, wheel, deltax, deltay, deltawheel, button, pressure),
    }
}

// When the mouse enters the view window
type MouseEnter struct {
	// Mouse because this is a mouse event
	mouse
}

func NewMouseEnter(x, y, wheel, deltax, deltay, deltawheel float32) MouseEnter {
    return MouseEnter{ mouse: newMouse(x, y, wheel, deltax, deltay, deltawheel) }
}


// When the mouse leaves the view window
type MouseLeave struct {
	// Mouse because this is a mouse event
	mouse
}

func NewMouseLeave(x, y, wheel, deltax, deltay, deltawheel float32) MouseLeave {
    return MouseLeave{ mouse: newMouse(x, y, wheel, deltax, deltay, deltawheel) }
}


// view port related events.
type view struct {
	// Actual, real width of the display
	Width float32
	// Actual, real height of the display
	Height float32
	// X position of the display on the screen
	X float32
	// Y position of the display on the screen
	Y float32
}

func newView(width, height, x, y float32) view {
    return view{ Width: width, Height:height, X: x, Y: y}
}

// When the application is ready to update the next frame on the view port
type ViewUpdate struct {
	// View because this is a view event
	view
}

func NewViewUpdate(width, height, x, y float32) ViewUpdate {
    return ViewUpdate{ view: view{ Width: width, Height:height, X: x, Y: y}}
}

// When the size of the application's view port changes
type ViewSize struct {
	// View because this is a view event
	view
}

func NewViewSize(width, height, x, y float32) ViewSize {
    return ViewSize{ view: view{ Width: width, Height:height, X: x, Y: y}}
}

// touch related event data
type touch struct {
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

func newTouch(id int, x, y, deltax, deltay, pressure float32, primary bool) touch {
    return touch{ID: id, X:x, Y:y, DeltaX: deltax, DeltaY: deltay, 
        Pressure: pressure, Primary: primary}
}


// When a touch begins
type TouchBegin struct {
	// Touch because this is a touch event
	touch
}

func NewTouchBegin(id int, x, y, deltax, deltay, pressure float32, primary bool) TouchBegin {
    return TouchBegin{touch: newTouch(id, x, y, deltax, deltay, pressure, primary)}
}

// When a touch ends
type TouchEnd struct {
	// Touch because this is a touch event
	touch
}

func NewTouchEnd(id int, x, y, deltax, deltay, pressure float32, primary bool) TouchEnd {
    return TouchEnd{touch: newTouch(id, x, y, deltax, deltay, pressure, primary)}
}

 // When a touch moved, ie is dragged
type TouchMoved struct {
	touch
}

func NewTouchMoved(id int, x, y, deltax, deltay, pressure float32, primary bool) TouchMoved {
    return TouchMoved{touch: newTouch(id, x, y, deltax, deltay, pressure, primary)}
}

// When a touch is canceled
type TouchCancel struct { 
	touch
}

func NewTouchCancel(id int, x, y, deltax, deltay, pressure float32, primary bool) TouchCancel {
    return TouchCancel{touch: newTouch(id, x, y, deltax, deltay, pressure, primary)}
}

