// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfwwin

const (
	NoAPI       = 0
	OpenGLAPI   = 0x00030001
	OpenGLESAPI = 0x00030002

	NoRobustness        = 0
	NoResetNotification = 0x00031001
	LoseContextOnReset  = 0x00031002

	OpenGLAnyProfile    = 0
	OpenGLCoreProfile   = 0x00032001
	OpenGLCompatProfile = 0x00032002

	CursorNormal   = 0x00034001
	CursorHidden   = 0x00034002
	CursorDisabled = 0x00034003

	AnyReleaseBehavior   = 0
	ReleaseBehaviorFlush = 0x00035001
	ReleaseBehaviorNone  = 0x00035002

	NativeContextAPI = 0x00036001
	EGLContextAPI    = 0x00036002
	OSMesaContextAPI = 0x00036003

	DontCare = -1
)

type Action int

const (
	Release Action = 0
	Press   Action = 1
	Repeat  Action = 2
)

type Hint int

const (
	Focused                Hint = 0x00020001
	Iconified              Hint = 0x00020002
	Resizable              Hint = 0x00020003
	Visible                Hint = 0x00020004
	Decorated              Hint = 0x00020005
	AutoIconify            Hint = 0x00020006
	Floating               Hint = 0x00020007
	Maximized              Hint = 0x00020008
	CenterCursor           Hint = 0x00020009
	TransparentFramebuffer Hint = 0x0002000A
	Hovered                Hint = 0x0002000B
	FocusOnShow            Hint = 0x0002000C

	RedBits        Hint = 0x00021001
	GreenBits      Hint = 0x00021002
	BlueBits       Hint = 0x00021003
	AlphaBits      Hint = 0x00021004
	DepthBits      Hint = 0x00021005
	StencilBits    Hint = 0x00021006
	AccumRedBits   Hint = 0x00021007
	AccumGreenBits Hint = 0x00021008
	AccumBlueBits  Hint = 0x00021009
	AccumAlphaBits Hint = 0x0002100A
	AuxBuffers     Hint = 0x0002100B
	Stereo         Hint = 0x0002100C
	Samples        Hint = 0x0002100D
	SRGBCapable    Hint = 0x0002100E
	RefreshRate    Hint = 0x0002100F
	DoubleBuffer   Hint = 0x00021010

	ClientAPI              Hint = 0x00022001
	ContextVersionMajor    Hint = 0x00022002
	ContextVersionMinor    Hint = 0x00022003
	ContextRevision        Hint = 0x00022004
	ContextRobustness      Hint = 0x00022005
	OpenGLForwardCompat    Hint = 0x00022006
	OpenGLDebugContext     Hint = 0x00022007
	OpenGLProfile          Hint = 0x00022008
	ContextReleaseBehavior Hint = 0x00022009
	ContextNoError         Hint = 0x0002200A
	ContextCreationAPI     Hint = 0x0002200B
	ScaleToMonitor         Hint = 0x0002200C
)

type InputMode int

const (
	CursorMode             InputMode = 0x00033001
	StickyKeysMode         InputMode = 0x00033002
	StickyMouseButtonsMode InputMode = 0x00033003
	LockKeyMods            InputMode = 0x00033004
	RawMouseMotion         InputMode = 0x00033005
)

type Key int

const (
	KeyUnknown Key = -1

	// Printable keys
	KeySpace        Key = 32
	KeyApostrophe   Key = 39 // '
	KeyComma        Key = 44 // ,
	KeyMinus        Key = 45 // -
	KeyPeriod       Key = 46 // .
	KeySlash        Key = 47 // /
	Key0            Key = 48
	Key1            Key = 49
	Key2            Key = 50
	Key3            Key = 51
	Key4            Key = 52
	Key5            Key = 53
	Key6            Key = 54
	Key7            Key = 55
	Key8            Key = 56
	Key9            Key = 57
	KeySemicolon    Key = 59 // ;
	KeyEqual        Key = 61 // =
	KeyA            Key = 65
	KeyB            Key = 66
	KeyC            Key = 67
	KeyD            Key = 68
	KeyE            Key = 69
	KeyF            Key = 70
	KeyG            Key = 71
	KeyH            Key = 72
	KeyI            Key = 73
	KeyJ            Key = 74
	KeyK            Key = 75
	KeyL            Key = 76
	KeyM            Key = 77
	KeyN            Key = 78
	KeyO            Key = 79
	KeyP            Key = 80
	KeyQ            Key = 81
	KeyR            Key = 82
	KeyS            Key = 83
	KeyT            Key = 84
	KeyU            Key = 85
	KeyV            Key = 86
	KeyW            Key = 87
	KeyX            Key = 88
	KeyY            Key = 89
	KeyZ            Key = 90
	KeyLeftBracket  Key = 91  // [
	KeyBackslash    Key = 92  // \
	KeyRightBracket Key = 93  // ]
	KeyGraveAccent  Key = 96  // `
	KeyWorld1       Key = 161 // non-US #1
	KeyWorld2       Key = 162 // non-US #2

	// Function keys
	KeyEscape       Key = 256
	KeyEnter        Key = 257
	KeyTab          Key = 258
	KeyBackspace    Key = 259
	KeyInsert       Key = 260
	KeyDelete       Key = 261
	KeyRight        Key = 262
	KeyLeft         Key = 263
	KeyDown         Key = 264
	KeyUp           Key = 265
	KeyPageUp       Key = 266
	KeyPageDown     Key = 267
	KeyHome         Key = 268
	KeyEnd          Key = 269
	KeyCapsLock     Key = 280
	KeyScrollLock   Key = 281
	KeyNumLock      Key = 282
	KeyPrintScreen  Key = 283
	KeyPause        Key = 284
	KeyF1           Key = 290
	KeyF2           Key = 291
	KeyF3           Key = 292
	KeyF4           Key = 293
	KeyF5           Key = 294
	KeyF6           Key = 295
	KeyF7           Key = 296
	KeyF8           Key = 297
	KeyF9           Key = 298
	KeyF10          Key = 299
	KeyF11          Key = 300
	KeyF12          Key = 301
	KeyF13          Key = 302
	KeyF14          Key = 303
	KeyF15          Key = 304
	KeyF16          Key = 305
	KeyF17          Key = 306
	KeyF18          Key = 307
	KeyF19          Key = 308
	KeyF20          Key = 309
	KeyF21          Key = 310
	KeyF22          Key = 311
	KeyF23          Key = 312
	KeyF24          Key = 313
	KeyF25          Key = 314
	KeyKP0          Key = 320
	KeyKP1          Key = 321
	KeyKP2          Key = 322
	KeyKP3          Key = 323
	KeyKP4          Key = 324
	KeyKP5          Key = 325
	KeyKP6          Key = 326
	KeyKP7          Key = 327
	KeyKP8          Key = 328
	KeyKP9          Key = 329
	KeyKPDecimal    Key = 330
	KeyKPDivide     Key = 331
	KeyKPMultiply   Key = 332
	KeyKPSubtract   Key = 333
	KeyKPAdd        Key = 334
	KeyKPEnter      Key = 335
	KeyKPEqual      Key = 336
	KeyLeftShift    Key = 340
	KeyLeftControl  Key = 341
	KeyLeftAlt      Key = 342
	KeyLeftSuper    Key = 343
	KeyRightShift   Key = 344
	KeyRightControl Key = 345
	KeyRightAlt     Key = 346
	KeyRightSuper   Key = 347
	KeyMenu         Key = 348

	KeyLast Key = KeyMenu
)

type ModifierKey int

const (
	ModShift    ModifierKey = 0x0001
	ModControl  ModifierKey = 0x0002
	ModAlt      ModifierKey = 0x0004
	ModSuper    ModifierKey = 0x0008
	ModCapsLock ModifierKey = 0x0010
	ModNumLock  ModifierKey = 0x0020
)

type MouseButton int

const (
	MouseButton1      MouseButton = 0
	MouseButton2      MouseButton = 1
	MouseButton3      MouseButton = 2
	MouseButton4      MouseButton = 3
	MouseButton5      MouseButton = 4
	MouseButton6      MouseButton = 5
	MouseButton7      MouseButton = 6
	MouseButton8      MouseButton = 7
	MouseButtonLast   MouseButton = MouseButton8
	MouseButtonLeft   MouseButton = MouseButton1
	MouseButtonRight  MouseButton = MouseButton2
	MouseButtonMiddle MouseButton = MouseButton3
)

var (
	NotInitialized     Error = 0x00010001
	NoCurrentContext   Error = 0x00010002
	InvalidEnum        Error = 0x00010003
	InvalidValue       Error = 0x00010004
	OutOfMemory        Error = 0x00010005
	ApiUnavailable     Error = 0x00010006
	VersionUnavailable Error = 0x00010007
	PlatformError      Error = 0x00010008
	FormatUnavailable  Error = 0x00010009
	NoWindowContext    Error = 0x0001000A
)

type PeripheralEvent int

const (
	Connected    PeripheralEvent = 0x00040001
	Disconnected PeripheralEvent = 0x00040002
)

type StandardCursor int

const (
	ArrowCursor     StandardCursor = 0x00036001
	IBeamCursor     StandardCursor = 0x00036002
	CrosshairCursor StandardCursor = 0x00036003
	HandCursor      StandardCursor = 0x00036004
	HResizeCursor   StandardCursor = 0x00036005
	VResizeCursor   StandardCursor = 0x00036006
)

type Error int

func (e Error) Error() string {
	switch e {
	case NotInitialized:
		return "the GLFW library is not initialized"
	case NoCurrentContext:
		return "there is no current context"
	case InvalidEnum:
		return "invalid argument for enum parameter"
	case InvalidValue:
		return "invalid value for parameter"
	case OutOfMemory:
		return "out of memory"
	case ApiUnavailable:
		return "the requested API is unavailable"
	case VersionUnavailable:
		return "the requested API version is unavailable"
	case PlatformError:
		return "a platform-specific error occurred"
	case FormatUnavailable:
		return "the requested format is unavailable"
	case NoWindowContext:
		return "the specified window has no context"
	default:
		return "ERROR: UNKNOWN GLFW ERROR"
	}
}

type VidMode struct {
	Width       int
	Height      int
	RedBits     int
	GreenBits   int
	BlueBits    int
	RefreshRate int
}

type Image struct {
	Width  int
	Height int
	Pixels []byte
}
