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

//go:build ignore
// +build ignore

// The key name convention follows the Web standard: https://www.w3.org/TR/uievents-code/#keyboard-key-codes

package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-gl/glfw/v3.3/glfw"
	"golang.org/x/mobile/event/key"
)

var (
	glfwKeyNameToGLFWKey        map[string]glfw.Key
	uiKeyNameToGLFWKeyName      map[string]string
	androidKeyToUIKeyName       map[int]string
	gbuildKeyToUIKeyName        map[key.Code]string
	uiKeyNameToJSKey            map[string]string
	edgeKeyCodeToName           map[int]string
	oldEbitenKeyNameToUIKeyName map[string]string
)

func init() {
	glfwKeyNameToGLFWKey = map[string]glfw.Key{
		"Unknown":      glfw.KeyUnknown,
		"Space":        glfw.KeySpace,
		"Apostrophe":   glfw.KeyApostrophe,
		"Comma":        glfw.KeyComma,
		"Minus":        glfw.KeyMinus,
		"Period":       glfw.KeyPeriod,
		"Slash":        glfw.KeySlash,
		"Semicolon":    glfw.KeySemicolon,
		"Equal":        glfw.KeyEqual,
		"LeftBracket":  glfw.KeyLeftBracket,
		"Backslash":    glfw.KeyBackslash,
		"RightBracket": glfw.KeyRightBracket,
		"GraveAccent":  glfw.KeyGraveAccent,
		"World1":       glfw.KeyWorld1,
		"World2":       glfw.KeyWorld2,
		"Escape":       glfw.KeyEscape,
		"Enter":        glfw.KeyEnter,
		"Tab":          glfw.KeyTab,
		"Backspace":    glfw.KeyBackspace,
		"Insert":       glfw.KeyInsert,
		"Delete":       glfw.KeyDelete,
		"Right":        glfw.KeyRight,
		"Left":         glfw.KeyLeft,
		"Down":         glfw.KeyDown,
		"Up":           glfw.KeyUp,
		"PageUp":       glfw.KeyPageUp,
		"PageDown":     glfw.KeyPageDown,
		"Home":         glfw.KeyHome,
		"End":          glfw.KeyEnd,
		"CapsLock":     glfw.KeyCapsLock,
		"ScrollLock":   glfw.KeyScrollLock,
		"NumLock":      glfw.KeyNumLock,
		"PrintScreen":  glfw.KeyPrintScreen,
		"Pause":        glfw.KeyPause,
		"LeftShift":    glfw.KeyLeftShift,
		"LeftControl":  glfw.KeyLeftControl,
		"LeftAlt":      glfw.KeyLeftAlt,
		"LeftSuper":    glfw.KeyLeftSuper,
		"RightShift":   glfw.KeyRightShift,
		"RightControl": glfw.KeyRightControl,
		"RightAlt":     glfw.KeyRightAlt,
		"RightSuper":   glfw.KeyRightSuper,
		"Menu":         glfw.KeyMenu,
		"KPDecimal":    glfw.KeyKPDecimal,
		"KPDivide":     glfw.KeyKPDivide,
		"KPMultiply":   glfw.KeyKPMultiply,
		"KPSubtract":   glfw.KeyKPSubtract,
		"KPAdd":        glfw.KeyKPAdd,
		"KPEnter":      glfw.KeyKPEnter,
		"KPEqual":      glfw.KeyKPEqual,
		"Last":         glfw.KeyLast,
	}

	uiKeyNameToGLFWKeyName = map[string]string{
		"Space":          "Space",
		"Quote":          "Apostrophe",
		"Comma":          "Comma",
		"Minus":          "Minus",
		"Period":         "Period",
		"Slash":          "Slash",
		"Semicolon":      "Semicolon",
		"Equal":          "Equal",
		"BracketLeft":    "LeftBracket",
		"Backslash":      "Backslash",
		"BracketRight":   "RightBracket",
		"Backquote":      "GraveAccent",
		"Escape":         "Escape",
		"Enter":          "Enter",
		"Tab":            "Tab",
		"Backspace":      "Backspace",
		"Insert":         "Insert",
		"Delete":         "Delete",
		"ArrowRight":     "Right",
		"ArrowLeft":      "Left",
		"ArrowDown":      "Down",
		"ArrowUp":        "Up",
		"PageUp":         "PageUp",
		"PageDown":       "PageDown",
		"Home":           "Home",
		"End":            "End",
		"CapsLock":       "CapsLock",
		"ScrollLock":     "ScrollLock",
		"NumLock":        "NumLock",
		"PrintScreen":    "PrintScreen",
		"Pause":          "Pause",
		"ShiftLeft":      "LeftShift",
		"ControlLeft":    "LeftControl",
		"AltLeft":        "LeftAlt",
		"MetaLeft":       "LeftSuper",
		"ShiftRight":     "RightShift",
		"ControlRight":   "RightControl",
		"AltRight":       "RightAlt",
		"MetaRight":      "RightSuper",
		"ContextMenu":    "Menu",
		"NumpadAdd":      "KPAdd",
		"NumpadDecimal":  "KPDecimal",
		"NumpadDivide":   "KPDivide",
		"NumpadMultiply": "KPMultiply",
		"NumpadSubtract": "KPSubtract",
		"NumpadEnter":    "KPEnter",
		"NumpadEqual":    "KPEqual",
	}

	// https://developer.android.com/reference/android/view/KeyEvent
	androidKeyToUIKeyName = map[int]string{
		55:  "Comma",
		56:  "Period",
		57:  "AltLeft",
		58:  "AltRight",
		115: "CapsLock",
		113: "ControlLeft",
		114: "ControlRight",
		59:  "ShiftLeft",
		60:  "ShiftRight",
		66:  "Enter",
		62:  "Space",
		61:  "Tab",
		112: "Delete", // KEYCODE_FORWARD_DEL
		123: "End",
		122: "Home",
		124: "Insert",
		93:  "PageDown",
		92:  "PageUp",
		20:  "ArrowDown",
		21:  "ArrowLeft",
		22:  "ArrowRight",
		19:  "ArrowUp",
		111: "Escape",
		67:  "Backspace", // KEYCODE_DEL
		75:  "Quote",
		69:  "Minus",
		76:  "Slash",
		74:  "Semicolon",
		70:  "Equal",
		71:  "BracketLeft",
		73:  "Backslash",
		72:  "BracketRight",
		68:  "Backquote",
		143: "NumLock",
		121: "Pause",       // KEYCODE_BREAK
		120: "PrintScreen", // KEYCODE_SYSRQ
		116: "ScrollLock",
		82:  "ContextMenu",
		157: "NumpadAdd",
		158: "NumpadDecimal",
		154: "NumpadDivide",
		155: "NumpadMultiply",
		156: "NumpadSubtract",
		160: "NumpadEnter",
		161: "NumpadEqual",
		117: "MetaLeft",
		118: "MetaRight",
	}

	gbuildKeyToUIKeyName = map[key.Code]string{
		key.CodeComma:              "Comma",
		key.CodeFullStop:           "Period",
		key.CodeLeftAlt:            "AltLeft",
		key.CodeRightAlt:           "AltRight",
		key.CodeCapsLock:           "CapsLock",
		key.CodeLeftControl:        "ControlLeft",
		key.CodeRightControl:       "ControlRight",
		key.CodeLeftShift:          "ShiftLeft",
		key.CodeRightShift:         "ShiftRight",
		key.CodeReturnEnter:        "Enter",
		key.CodeSpacebar:           "Space",
		key.CodeTab:                "Tab",
		key.CodeDeleteForward:      "Delete",
		key.CodeEnd:                "End",
		key.CodeHome:               "Home",
		key.CodeInsert:             "Insert",
		key.CodePageDown:           "PageDown",
		key.CodePageUp:             "PageUp",
		key.CodeDownArrow:          "ArrowDown",
		key.CodeLeftArrow:          "ArrowLeft",
		key.CodeRightArrow:         "ArrowRight",
		key.CodeUpArrow:            "ArrowUp",
		key.CodeEscape:             "Escape",
		key.CodeDeleteBackspace:    "Backspace",
		key.CodeApostrophe:         "Quote",
		key.CodeHyphenMinus:        "Minus",
		key.CodeSlash:              "Slash",
		key.CodeSemicolon:          "Semicolon",
		key.CodeEqualSign:          "Equal",
		key.CodeLeftSquareBracket:  "BracketLeft",
		key.CodeBackslash:          "Backslash",
		key.CodeRightSquareBracket: "BracketRight",
		key.CodeGraveAccent:        "Backquote",
		key.CodeKeypadNumLock:      "NumLock",
		key.CodePause:              "Pause",
		key.CodeKeypadPlusSign:     "NumpadAdd",
		key.CodeKeypadFullStop:     "NumpadDecimal",
		key.CodeKeypadSlash:        "NumpadDivide",
		key.CodeKeypadAsterisk:     "NumpadMultiply",
		key.CodeKeypadHyphenMinus:  "NumpadSubtract",
		key.CodeKeypadEnter:        "NumpadEnter",
		key.CodeKeypadEqualSign:    "NumpadEqual",
		key.CodeLeftGUI:            "MetaLeft",
		key.CodeRightGUI:           "MetaRight",

		// Missing keys:
		//   ui.KeyPrintScreen
		//   ui.KeyScrollLock
		//   ui.KeyMenu
	}

	// The UI key and JS key are almost same but very slightly different (e.g., 'A' vs 'KeyA').
	uiKeyNameToJSKey = map[string]string{
		"Comma":          "Comma",
		"Period":         "Period",
		"AltLeft":        "AltLeft",
		"AltRight":       "AltRight",
		"CapsLock":       "CapsLock",
		"ControlLeft":    "ControlLeft",
		"ControlRight":   "ControlRight",
		"ShiftLeft":      "ShiftLeft",
		"ShiftRight":     "ShiftRight",
		"Enter":          "Enter",
		"Space":          "Space",
		"Tab":            "Tab",
		"Delete":         "Delete",
		"End":            "End",
		"Home":           "Home",
		"Insert":         "Insert",
		"PageDown":       "PageDown",
		"PageUp":         "PageUp",
		"ArrowDown":      "ArrowDown",
		"ArrowLeft":      "ArrowLeft",
		"ArrowRight":     "ArrowRight",
		"ArrowUp":        "ArrowUp",
		"Escape":         "Escape",
		"Backspace":      "Backspace",
		"Quote":          "Quote",
		"Minus":          "Minus",
		"Slash":          "Slash",
		"Semicolon":      "Semicolon",
		"Equal":          "Equal",
		"BracketLeft":    "BracketLeft",
		"Backslash":      "Backslash",
		"BracketRight":   "BracketRight",
		"Backquote":      "Backquote",
		"NumLock":        "NumLock",
		"Pause":          "Pause",
		"PrintScreen":    "PrintScreen",
		"ScrollLock":     "ScrollLock",
		"ContextMenu":    "ContextMenu",
		"NumpadAdd":      "NumpadAdd",
		"NumpadDecimal":  "NumpadDecimal",
		"NumpadDivide":   "NumpadDivide",
		"NumpadMultiply": "NumpadMultiply",
		"NumpadSubtract": "NumpadSubtract",
		"NumpadEnter":    "NumpadEnter",
		"NumpadEqual":    "NumpadEqual",
		"MetaLeft":       "MetaLeft",
		"MetaRight":      "MetaRight",
	}

	// ASCII: 0 - 9
	for c := '0'; c <= '9'; c++ {
		glfwKeyNameToGLFWKey[string(c)] = glfw.Key0 + glfw.Key(c) - '0'
		name := "Digit" + string(c)
		uiKeyNameToGLFWKeyName[name] = string(c)
		androidKeyToUIKeyName[7+int(c)-'0'] = name
		// Gomobile's key code (= USB HID key codes) has successive key codes for 1, 2, ..., 9, 0
		// in this order.
		if c == '0' {
			gbuildKeyToUIKeyName[key.Code0] = name
		} else {
			gbuildKeyToUIKeyName[key.Code1+key.Code(c)-'1'] = name
		}
		uiKeyNameToJSKey[name] = name
	}
	// ASCII: A - Z
	for c := 'A'; c <= 'Z'; c++ {
		glfwKeyNameToGLFWKey[string(c)] = glfw.KeyA + glfw.Key(c) - 'A'
		uiKeyNameToGLFWKeyName[string(c)] = string(c)
		androidKeyToUIKeyName[29+int(c)-'A'] = string(c)
		gbuildKeyToUIKeyName[key.CodeA+key.Code(c)-'A'] = string(c)
		uiKeyNameToJSKey[string(c)] = "Key" + string(c)
	}
	// Function keys
	for i := 1; i <= 12; i++ {
		name := "F" + strconv.Itoa(i)
		glfwKeyNameToGLFWKey[name] = glfw.KeyF1 + glfw.Key(i) - 1
		uiKeyNameToGLFWKeyName[name] = name
		androidKeyToUIKeyName[131+i-1] = name
		gbuildKeyToUIKeyName[key.CodeF1+key.Code(i)-1] = name
		uiKeyNameToJSKey[name] = name
	}
	// Numpad
	// https://www.w3.org/TR/uievents-code/#key-numpad-section
	for c := '0'; c <= '9'; c++ {
		name := "Numpad" + string(c)
		glfwKeyNameToGLFWKey["KP"+string(c)] = glfw.KeyKP0 + glfw.Key(c) - '0'
		uiKeyNameToGLFWKeyName[name] = "KP" + string(c)
		androidKeyToUIKeyName[144+int(c)-'0'] = name
		// Gomobile's key code (= USB HID key codes) has successive key codes for 1, 2, ..., 9, 0
		// in this order.
		if c == '0' {
			gbuildKeyToUIKeyName[key.CodeKeypad0] = name
		} else {
			gbuildKeyToUIKeyName[key.CodeKeypad1+key.Code(c)-'1'] = name
		}
		uiKeyNameToJSKey[name] = name
	}

	// Keys for backward compatibility
	oldEbitenKeyNameToUIKeyName = map[string]string{
		"0":            "Digit0",
		"1":            "Digit1",
		"2":            "Digit2",
		"3":            "Digit3",
		"4":            "Digit4",
		"5":            "Digit5",
		"6":            "Digit6",
		"7":            "Digit7",
		"8":            "Digit8",
		"9":            "Digit9",
		"Apostrophe":   "Quote",
		"Down":         "ArrowDown",
		"GraveAccent":  "Backquote",
		"KP0":          "Numpad0",
		"KP1":          "Numpad1",
		"KP2":          "Numpad2",
		"KP3":          "Numpad3",
		"KP4":          "Numpad4",
		"KP5":          "Numpad5",
		"KP6":          "Numpad6",
		"KP7":          "Numpad7",
		"KP8":          "Numpad8",
		"KP9":          "Numpad9",
		"KPAdd":        "NumpadAdd",
		"KPDecimal":    "NumpadDecimal",
		"KPDivide":     "NumpadDivide",
		"KPMultiply":   "NumpadMultiply",
		"KPSubtract":   "NumpadSubtract",
		"KPEnter":      "NumpadEnter",
		"KPEqual":      "NumpadEqual",
		"Left":         "ArrowLeft",
		"LeftBracket":  "BracketLeft",
		"Menu":         "ContextMenu",
		"Right":        "ArrowRight",
		"RightBracket": "BracketRight",
		"Up":           "ArrowUp",
	}
}

func init() {
	// https://developer.mozilla.org/en-US/docs/Web/API/KeyboardEvent/keyCode
	// TODO: How should we treat modifier keys? Now 'left' modifier keys are available.
	edgeKeyCodeToName = map[int]string{
		0xbc: "Comma",
		0xbe: "Period",
		0x12: "AltLeft",
		0x14: "CapsLock",
		0x11: "ControlLeft",
		0x10: "ShiftLeft",
		0x0D: "Enter",
		0x20: "Space",
		0x09: "Tab",
		0x2E: "Delete",
		0x23: "End",
		0x24: "Home",
		0x2D: "Insert",
		0x22: "PageDown",
		0x21: "PageUp",
		0x28: "ArrowDown",
		0x25: "ArrowLeft",
		0x27: "ArrowRight",
		0x26: "ArrowUp",
		0x1B: "Escape",
		0xde: "Quote",
		0xbd: "Minus",
		0xbf: "Slash",
		0xba: "Semicolon",
		0xbb: "Equal",
		0xdb: "BracketLeft",
		0xdc: "Backslash",
		0xdd: "BracketRight",
		0xc0: "Backquote",
		0x08: "Backspace",
		0x90: "NumLock",
		0x6b: "NumpadAdd",
		0x6e: "NumpadDecimal",
		0x6f: "NumpadDivide",
		0x6a: "NumpadMultiply",
		0x6d: "NumpadSubtract",
		0x13: "Pause",
		0x91: "ScrollLock",
		0x5d: "ContextMenu",
		0x5b: "MetaLeft",
		0x5c: "MetaRight",

		// On Edge, this key does not work. PrintScreen works only on keyup event.
		// 0x2C: "PrintScreen",

		// On Edge, it is impossible to tell NumpadEnter and Enter / NumpadEqual and Equal.
		// 0x0d: "NumpadEnter",
		// 0x0c: "NumpadEqual",
	}
	// ASCII: 0 - 9
	for c := '0'; c <= '9'; c++ {
		edgeKeyCodeToName[int(c)] = "Digit" + string(c)
	}
	// ASCII: A - Z
	for c := 'A'; c <= 'Z'; c++ {
		edgeKeyCodeToName[int(c)] = string(c)
	}
	// Function keys
	for i := 1; i <= 12; i++ {
		edgeKeyCodeToName[0x70+i-1] = "F" + strconv.Itoa(i)
	}
	// Numpad keys
	for c := '0'; c <= '9'; c++ {
		edgeKeyCodeToName[0x60+int(c-'0')] = "Numpad" + string(c)
	}
}

const ebitenKeysTmpl = `{{.License}}

{{.DoNotEdit}}

package ebiten

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// A Key represents a keyboard key.
// These keys represent pysical keys of US keyboard.
// For example, KeyQ represents Q key on US keyboards and ' (quote) key on Dvorak keyboards.
type Key int

// Keys.
const (
{{range $index, $name := .EbitenKeyNamesWithoutMods}}Key{{$name}} Key = Key(ui.Key{{$name}})
{{end}}	KeyAlt     Key = Key(ui.KeyReserved0)
	KeyControl Key = Key(ui.KeyReserved1)
	KeyShift   Key = Key(ui.KeyReserved2)
	KeyMeta    Key = Key(ui.KeyReserved3)
	KeyMax     Key = KeyMeta

	// Keys for backward compatibility.
	// Deprecated: as of v2.1.
{{range $old, $new := .OldEbitenKeyNameToUIKeyName}}Key{{$old}} Key = Key(ui.Key{{$new}})
{{end}}
)

func (k Key) isValid() bool {
	switch k {
	{{range $name := .EbitenKeyNamesWithoutOld}}case Key{{$name}}:
		return true
	{{end}}
	default:
		return false
	}
}

// String returns a string representing the key.
//
// If k is an undefined key, String returns an empty string.
func (k Key) String() string {
	switch k {
	{{range $name := .EbitenKeyNamesWithoutOld}}case Key{{$name}}:
		return {{$name | printf "%q"}}
	{{end}}}
	return ""
}

func keyNameToKeyCode(name string) (Key, bool) {
	switch strings.ToLower(name) {
	{{range $name := .EbitenKeyNames}}case {{$name | printf "%q" | ToLower}}:
		return Key{{$name}}, true
	{{end}}}
	return 0, false
}

// MarshalText implements encoding.TextMarshaler.
func (k Key) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (k *Key) UnmarshalText(text []byte) error {
	key, ok := keyNameToKeyCode(string(text))
	if !ok {
		return fmt.Errorf("ebiten: unexpected key name: %s", string(text))
	}
	*k = key
	return nil
}
`

const uiKeysTmpl = `{{.License}}

{{.DoNotEdit}}

package ui

import (
	"fmt"
)

type Key int

const (
{{range $index, $name := .UIKeyNames}}Key{{$name}}{{if eq $index 0}} Key = iota{{end}}
{{end}}	KeyReserved0
	KeyReserved1
	KeyReserved2
	KeyReserved3
)

func (k Key) String() string {
	switch k {
	{{range $index, $name := .UIKeyNames}}case Key{{$name}}:
		return {{$name | printf "Key%s" | printf "%q"}}
	{{end}}}
	panic(fmt.Sprintf("ui: invalid key: %d", k))
}
`

const eventKeysTmpl = `{{.License}}

{{.DoNotEdit}}

package event

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type Key = ui.Key

const (
{{range $index, $name := .UIKeyNames}}Key{{$name}} = ui.Key{{$name}}
{{end}}
)
`

const uiGLFWKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

var uiKeyToGLFWKey = map[Key]glfw.Key{
{{range $dname, $gname := .UIKeyNameToGLFWKeyName}}Key{{$dname}}: glfw.Key{{$gname}},
{{end}}
}
`

const uiJSKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package ui

import (
	"syscall/js"
)

var uiKeyToJSKey = map[Key]js.Value{
{{range $name, $code := .UIKeyNameToJSKey}}Key{{$name}}: js.ValueOf({{$code | printf "%q"}}),
{{end}}
}

var edgeKeyCodeToUIKey = map[int]Key{
{{range $code, $name := .EdgeKeyCodeToName}}{{$code}}: Key{{$name}},
{{end}}
}
`

const glfwKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package glfw

const (
{{range $name, $key := .GLFWKeyNameToGLFWKey}}Key{{$name}} = Key({{$key}})
{{end}}
)
`

const mobileAndroidKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package ebitenmobileview

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var androidKeyToUIKey = map[int]ui.Key{
{{range $key, $name := .AndroidKeyToUIKeyName}}{{$key}}: ui.Key{{$name}},
{{end}}
}
`

const uiMobileKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package ui

import (
	"golang.org/x/mobile/event/key"
)

var gbuildKeyToUIKey = map[key.Code]Key{
{{range $key, $name := .GBuildKeyToUIKeyName}}key.{{$key}}: Key{{$name}},
{{end}}
}
`

func digitKey(name string) int {
	if len(name) != 1 {
		return -1
	}
	c := name[0]
	if c < '0' || '9' < c {
		return -1
	}
	return int(c - '0')
}

func alphabetKey(name string) rune {
	if len(name) != 1 {
		return -1
	}
	c := rune(name[0])
	if c < 'A' || 'Z' < c {
		return -1
	}
	return c
}

func functionKey(name string) int {
	if len(name) < 2 {
		return -1
	}
	if name[0] != 'F' {
		return -1
	}
	i, err := strconv.Atoi(name[1:])
	if err != nil {
		return -1
	}
	return i
}

func keyNamesLess(k []string) func(i, j int) bool {
	return func(i, j int) bool {
		k0, k1 := k[i], k[j]
		d0, d1 := digitKey(k0), digitKey(k1)
		a0, a1 := alphabetKey(k0), alphabetKey(k1)
		f0, f1 := functionKey(k0), functionKey(k1)
		if d0 != -1 {
			if d1 != -1 {
				return d0 < d1
			}
			return true
		}
		if a0 != -1 {
			if d1 != -1 {
				return false
			}
			if a1 != -1 {
				return a0 < a1
			}
			return true
		}
		if d1 != -1 {
			return false
		}
		if a1 != -1 {
			return false
		}
		if f0 != -1 && f1 != -1 {
			return f0 < f1
		}
		return k0 < k1
	}
}

const license = `// Copyright 2013 The Ebitengine Authors
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
`

func main() {
	// Follow the standard comment rule (https://golang.org/s/generatedcode).
	doNotEdit := "// Code generated by genkeys.go using 'go generate'. DO NOT EDIT."

	ebitenKeyNames := []string{}
	ebitenKeyNamesWithoutOld := []string{}
	ebitenKeyNamesWithoutMods := []string{}
	uiKeyNames := []string{}

	for name := range uiKeyNameToJSKey {
		uiKeyNames = append(uiKeyNames, name)
		ebitenKeyNames = append(ebitenKeyNames, name)
		ebitenKeyNamesWithoutOld = append(ebitenKeyNamesWithoutOld, name)
		ebitenKeyNamesWithoutMods = append(ebitenKeyNamesWithoutMods, name)
	}
	for old := range oldEbitenKeyNameToUIKeyName {
		ebitenKeyNames = append(ebitenKeyNames, old)
	}
	// Keys for modifiers
	ebitenKeyNames = append(ebitenKeyNames, "Alt", "Control", "Shift", "Meta")
	ebitenKeyNamesWithoutOld = append(ebitenKeyNamesWithoutOld, "Alt", "Control", "Shift", "Meta")

	sort.Slice(ebitenKeyNames, keyNamesLess(ebitenKeyNames))
	sort.Slice(ebitenKeyNamesWithoutOld, keyNamesLess(ebitenKeyNamesWithoutOld))
	sort.Slice(ebitenKeyNamesWithoutMods, keyNamesLess(ebitenKeyNamesWithoutMods))
	sort.Slice(uiKeyNames, keyNamesLess(uiKeyNames))

	// TODO: Add this line for event package (#926).
	//
	//     filepath.Join("event", "keys.go"):                              eventKeysTmpl,

	for path, tmpl := range map[string]string{
		filepath.Join("internal", "glfw", "keys.go"):                   glfwKeysTmpl,
		filepath.Join("internal", "ui", "keys.go"):                     uiKeysTmpl,
		filepath.Join("internal", "ui", "keys_glfw.go"):                uiGLFWKeysTmpl,
		filepath.Join("internal", "ui", "keys_mobile.go"):              uiMobileKeysTmpl,
		filepath.Join("internal", "ui", "keys_js.go"):                  uiJSKeysTmpl,
		filepath.Join("keys.go"):                                       ebitenKeysTmpl,
		filepath.Join("mobile", "ebitenmobileview", "keys_android.go"): mobileAndroidKeysTmpl,
	} {
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		funcs := template.FuncMap{
			"ToLower": strings.ToLower,
		}
		tmpl, err := template.New(path).Funcs(funcs).Parse(tmpl)
		if err != nil {
			log.Fatal(err)
		}

		// The build tag can't be included in the templates because of `go vet`.
		// Pass the build tag and extract this in the template to make `go vet` happy.
		buildTag := ""
		switch path {
		case filepath.Join("internal", "glfw", "keys.go"):
			buildTag = "//go:build !js" +
				"\n// +build !js"
		case filepath.Join("internal", "ui", "keys_mobile.go"):
			buildTag = "//go:build (android || ios) && !nintendosdk" +
				"\n// +build android ios" +
				"\n// +build !nintendosdk"
		case filepath.Join("internal", "ui", "keys_glfw.go"):
			buildTag = "//go:build !android && !ios && !js && !nintendosdk" +
				"\n// +build !android,!ios,!js,!nintendosdk"
		}
		// NOTE: According to godoc, maps are automatically sorted by key.
		if err := tmpl.Execute(f, struct {
			License                     string
			DoNotEdit                   string
			BuildTag                    string
			UIKeyNameToJSKey            map[string]string
			EdgeKeyCodeToName           map[int]string
			EbitenKeyNames              []string
			EbitenKeyNamesWithoutOld    []string
			EbitenKeyNamesWithoutMods   []string
			GLFWKeyNameToGLFWKey        map[string]glfw.Key
			UIKeyNames                  []string
			UIKeyNameToGLFWKeyName      map[string]string
			AndroidKeyToUIKeyName       map[int]string
			GBuildKeyToUIKeyName        map[key.Code]string
			OldEbitenKeyNameToUIKeyName map[string]string
		}{
			License:                     license,
			DoNotEdit:                   doNotEdit,
			BuildTag:                    buildTag,
			UIKeyNameToJSKey:            uiKeyNameToJSKey,
			EdgeKeyCodeToName:           edgeKeyCodeToName,
			EbitenKeyNames:              ebitenKeyNames,
			EbitenKeyNamesWithoutOld:    ebitenKeyNamesWithoutOld,
			EbitenKeyNamesWithoutMods:   ebitenKeyNamesWithoutMods,
			GLFWKeyNameToGLFWKey:        glfwKeyNameToGLFWKey,
			UIKeyNames:                  uiKeyNames,
			UIKeyNameToGLFWKeyName:      uiKeyNameToGLFWKeyName,
			AndroidKeyToUIKeyName:       androidKeyToUIKeyName,
			GBuildKeyToUIKeyName:        gbuildKeyToUIKeyName,
			OldEbitenKeyNameToUIKeyName: oldEbitenKeyNameToUIKeyName,
		}); err != nil {
			log.Fatal(err)
		}
	}
}
