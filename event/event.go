package event

// Kind is the detailed type of the event
type Kind int

// Kind constants
const (
    // When a joystick axis changes in value
    KindJoystickAxis             = Kind(iota)
    // When a joystick button is pressed
    KindJoystickButtonDown   
    // When a joystick button is released
    KindJoystickButtonUp
    // When a joystick is added or removed to the system.
    KindJoystickConfiguration     

    // When a keyboard key is pressed
    KindKeyDown               
    // When a keyboard key is released
    KindKeyUp
    // When a character of text is typed on the keyboard. 
    KindKeyCharacter                 

    // When a joystick axis changes in value.
    KindMouseAxes
    // When a mouse button is pressed             
    KindMouseButtonDown
    // When a mouse button is released             
    KindMouseButtonUp
    // When the mouse enters the display window
    KindMouseEnter
    // When the mouse leaves the display
    KindMouseLeave
    
    // When the application is created
    KindApplicationCreate
    // When the application is destroyed
    KindApplicationDestroy
    // When the application becomes visible
    KindApplicationShow
    // When the application becomes hidden
    KindApplicationHide
    // When the application gains the focus
    KindApplicationFocus
    // When the application looses the focus.
    KindApplicationBlur

    // When the application is ready to paint the next frame on the display
    KindDisplayPaint
    // When the size of the application's display changes
    KindDisplaySize
    // When the orientation of the dipslay changes 
    KindDisplayOrientation

    // When a touch begins
    KindTouchBegin
    // When a touch ends            
    KindTouchEnd
    // When a touch moved, ie is dragged              
    KindTouchMove             
    // When a touch is canceled
    KindTouchCancel       
)

//Timestamp is the time the event was sent, in seconds after the program started
type Timestamp float64 

// Source is the source, or origin of an event
type Source interface {
    Name() string
}

// Events implement this interface
type Event interface {
    // Source of the event
    Source() Source
    // Time the event was sent, in seconds after the program started
    Timestamp() Timestamp
    // Detailed type of the event
    Kind() Kind
}

// Basic is a basic event 
type Basic struct {
    Any struct { 
        Source
        Timestamp
        Kind 
    }
}

// Source returns the source of the basic event
func (b * Basic) Source() Source {
    return b.Any.Source
}

// Time stamp returns the tie stamp of the basic event
func (b * Basic) Timestamp() Timestamp {
    return b.Any.Timestamp
}

// Kind returns the kind of the event.
func (b * Basic) Kind() Kind {
    return b.Any.Kind
}


// Code of a key press
type Code int
// Modifiers of a key press
type Modifiers int

// Character of a key press, for KindKeyCharacter in particular
type Character rune

// Keyboard related events
type Key struct {
    // Basic event
    Basic
    // Code of the key pressed or released
    Code
    // Character typed
    Character
    // Key board modifiers
    Modifiers
}

// Joystick or mouse axis index
type Axis int
// Joystick or mouse button index
type Button int
// Which joystick or mouse index caused the event
type Which int
// Position for a joystick or mouse axis event
type Position float32

// Joystick related events
type Joystick struct {
    // Basic event
    Basic
    // Which joystick caused the event
    Which
    // Which joystick axis changed, if any 
    Axis
    // which button was pressed, if any.
    Button
    // Position of the axis or button after the change
    Position 
}

// Change in position for a mouse event
type Delta float32
// Pressure applied for a moue or touch event
type Pressure float32

// Mouse related events 
type Mouse struct {
    // Basic event
    Basic
    
    // X position of the event
    X Position
    // Y position of the event
    Y Position
    // Z position of the event
    Z Position
    // W position of the event
    W Position
    // Change in X since last event
    DX Delta
    // Change in Y since last event
    DY Delta
    // Change in Z since last event
    DZ Delta
    // Change in W since last event
    DW Delta
    
    // Which button was pressed, if any.
    Button
    // Pressure applied on the mouse click    
    Pressure 
}

// Application related events
type Application struct {
    // Basic event
    Basic
}

// Orientation of a display
type Orientation int

const (
    OrientationUnknown = Orientation(iota)
    OrientationPortrait
    OrientationLandscape
)

// Display related events 
type Display struct {
    // Basic event
    Basic
    // Orientation of the display
    Orientation
    // Actual, real width of the display
    Width float32 
    // Actual, real height of the display
    Height float32
    // X position of the display on the screen
    X float32
    // Y position of the display on the screen
    Y float32
}    

// Touch related events 
type Touch struct {
    // Basic event
    Basic
    // Which "finger" caused the touch event
    Which
    // X position of the event
    X Position
    // Y position of the event
    Y Position
    DX Delta
    // Change in Y since last event
    DY Delta
    
    // Is the touch event primary or not. 
    Primary bool
}

// A channel to listen on for events
type Channel (<-Event)

// NewChannel returns a channel on which you can listen for incoming events
func NewChannel() (Channel) {
    // TODO
    return nil
}

