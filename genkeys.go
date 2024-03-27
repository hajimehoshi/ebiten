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

// The key name convention follows the Web standard: https://www.w3.org/TR/uievents-code/#code-value-tables

package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var (
	glfwKeyNameToGLFWKey            map[string]int
	uiKeyNameToGLFWKeyName          map[string]string
	androidKeyToUIKeyName           map[int]string
	iosKeyToUIKeyName               map[int]string
	uiKeyNameToJSCode               map[string]string
	oldEbitengineKeyNameToUIKeyName map[string]string
)

func init() {
	glfwKeyNameToGLFWKey = map[string]int{
		"Unknown":      -1,
		"Space":        32,
		"Apostrophe":   39,
		"Comma":        44,
		"Minus":        45,
		"Period":       46,
		"Slash":        47,
		"Semicolon":    59,
		"Equal":        61,
		"LeftBracket":  91,
		"Backslash":    92,
		"RightBracket": 93,
		"GraveAccent":  96,
		"World1":       161,
		"World2":       162,
		"Escape":       256,
		"Enter":        257,
		"Tab":          258,
		"Backspace":    259,
		"Insert":       260,
		"Delete":       261,
		"Right":        262,
		"Left":         263,
		"Down":         264,
		"Up":           265,
		"PageUp":       266,
		"PageDown":     267,
		"Home":         268,
		"End":          269,
		"CapsLock":     280,
		"ScrollLock":   281,
		"NumLock":      282,
		"PrintScreen":  283,
		"Pause":        284,
		"LeftShift":    340,
		"LeftControl":  341,
		"LeftAlt":      342,
		"LeftSuper":    343,
		"RightShift":   344,
		"RightControl": 345,
		"RightAlt":     346,
		"RightSuper":   347,
		"Menu":         348,
		"KPDecimal":    330,
		"KPDivide":     331,
		"KPMultiply":   332,
		"KPSubtract":   333,
		"KPAdd":        334,
		"KPEnter":      335,
		"KPEqual":      336,
		"Last":         348,
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
		"IntlBackslash":  "World1",
	}

	// https://developer.android.com/reference/android/view/KeyEvent
	//
	// Android doesn't distinguish these keys:
	// - a US backslash key (HID: 0x31),
	// - an international pound/tilde key (HID: 0x32), and
	// - an international backslash key (HID: 0x64).
	// These are mapped to the same key code KEYCODE_BACKSLASH (73).
	// See https://source.android.com/docs/core/interaction/input/keyboard-devices
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

	// https://developer.apple.com/documentation/uikit/uikeyboardhidusage?language=objc
	iosKeyToUIKeyName = map[int]string{
		0xE2: "AltLeft",
		0xE6: "AltRight",
		0x51: "ArrowDown",
		0x50: "ArrowLeft",
		0x4F: "ArrowRight",
		0x52: "ArrowUp",
		0x35: "Backquote",

		// These three keys are:
		// - US backslash-pipe key, and
		// - non-US hashmark key (bottom left of return; on German layout, this is the #' key).
		// On US layout configurations, they all map to the same characters - the backslash.
		//
		// See also: https://www.w3.org/TR/uievents-code/#keyboard-102
		0x31: "Backslash", // UIKeyboardHIDUsageKeyboardBackslash
		0x32: "Backslash", // UIKeyboardHIDUsageKeyboardNonUSPound

		0x64: "IntlBackslash", // UIKeyboardHIDUsageKeyboardNonUSBackslash

		0x2A: "Backspace",
		0x2F: "BracketLeft",
		0x30: "BracketRight",

		// Caps Lock can either be a normal key or a hardware toggle.
		0x39: "CapsLock", // UIKeyboardHIDUsageKeyboardCapsLock
		0x82: "CapsLock", // UIKeyboardHIDUsageKeyboardLockingCapsLock

		0x36: "Comma",
		0xE0: "ControlLeft",
		0xE4: "ControlRight",
		0x4C: "Delete",
		0x4D: "End",
		0x28: "Enter",
		0x2E: "Equal",
		0x29: "Escape",
		0x4A: "Home",
		0x49: "Insert",
		0x76: "ContextMenu",
		0xE3: "MetaLeft",
		0xE7: "MetaRight",
		0x2D: "Minus",

		// Num Lock can either be a normal key or a hardware toggle.
		0x53: "NumLock", // UIKeyboardHIDUsageKeyboardNumLock
		0x83: "NumLock", // UIKeyboardHIDUsageKeyboardLockingNumLock

		0x57: "NumpadAdd",

		// Some keyboard layouts have a comma, some a period on the numeric pad.
		// They are the same key, though.
		0x63: "NumpadDecimal", // UIKeyboardHIDUsageKeypadPeriod
		0x85: "NumpadDecimal", // UIKeyboardHIDUsageKeypadComma

		0x54: "NumpadDivide",
		0x58: "NumpadEnter",

		// Some numeric keypads also have an equals sign.
		// There appear to be two separate keycodes for that.
		0x67: "NumpadEqual", // UIKeyboardHIDUsageKeypadEqualSign
		0x86: "NumpadEqual", // UIKeyboardHIDUsageKeypadEqualSignAS400

		0x55: "NumpadMultiply",
		0x56: "NumpadSubtract",
		0x4E: "PageDown",
		0x4B: "PageUp",
		0x48: "Pause",
		0x37: "Period",
		0x46: "PrintScreen",
		0x34: "Quote",

		// Scroll Lock can either be a normal key or a hardware toggle.
		0x47: "ScrollLock", // UIKeyboardHIDUsageKeyboardScrollLock
		0x84: "ScrollLock", // UIKeyboardHIDUsageKeyboardLockingScrollLock

		0x33: "Semicolon",
		0xE1: "ShiftLeft",
		0xE5: "ShiftRight",
		0x38: "Slash",
		0x2C: "Space",
		0x2B: "Tab",
	}

	// The UI key and JS key are almost same but very slightly different (e.g., 'A' vs 'KeyA').
	uiKeyNameToJSCode = map[string]string{
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
		"IntlBackslash":  "IntlBackslash",
	}

	const (
		glfwKey0   = 48
		glfwKeyA   = 65
		glfwKeyF1  = 290
		glfwKeyKP0 = 320
	)

	// ASCII: 0 - 9
	for c := '0'; c <= '9'; c++ {
		glfwKeyNameToGLFWKey[string(c)] = int(glfwKey0 + c - '0')
		name := "Digit" + string(c)
		uiKeyNameToGLFWKeyName[name] = string(c)
		androidKeyToUIKeyName[7+int(c)-'0'] = name
		// Gomobile's key code (= USB HID key codes) has successive key codes for 1, 2, ..., 9, 0
		// in this order. Same for iOS.
		if c == '0' {
			iosKeyToUIKeyName[0x27] = name
		} else {
			iosKeyToUIKeyName[0x1E+int(c)-'1'] = name
		}
		uiKeyNameToJSCode[name] = name

	}
	// ASCII: A - Z
	for c := 'A'; c <= 'Z'; c++ {
		glfwKeyNameToGLFWKey[string(c)] = int(glfwKeyA + c - 'A')
		uiKeyNameToGLFWKeyName[string(c)] = string(c)
		androidKeyToUIKeyName[29+int(c)-'A'] = string(c)
		iosKeyToUIKeyName[0x04+int(c)-'A'] = string(c)
		uiKeyNameToJSCode[string(c)] = "Key" + string(c)
	}
	// Function keys
	for i := 1; i <= 24; i++ {
		name := "F" + strconv.Itoa(i)
		glfwKeyNameToGLFWKey[name] = glfwKeyF1 + i - 1
		uiKeyNameToGLFWKeyName[name] = name
		// Android doesn't support F13 and more as constants of KeyEvent:
		// https://developer.android.com/reference/android/view/KeyEvent
		//
		// Note that F13 might be avilable if HID devices are available directly:
		// https://source.android.com/docs/core/interaction/input/keyboard-devices
		if i <= 12 {
			androidKeyToUIKeyName[131+i-1] = name
		}
		if i <= 12 {
			iosKeyToUIKeyName[0x3A+i-1] = name
		} else {
			iosKeyToUIKeyName[0x68+i-13] = name
		}
		uiKeyNameToJSCode[name] = name
	}
	// Numpad
	// https://www.w3.org/TR/uievents-code/#key-numpad-section
	for c := '0'; c <= '9'; c++ {
		name := "Numpad" + string(c)
		glfwKeyNameToGLFWKey["KP"+string(c)] = int(glfwKeyKP0 + c - '0')
		uiKeyNameToGLFWKeyName[name] = "KP" + string(c)
		androidKeyToUIKeyName[144+int(c)-'0'] = name
		// Gomobile's key code (= USB HID key codes) has successive key codes for 1, 2, ..., 9, 0
		// in this order. Same for iOS.
		if c == '0' {
			iosKeyToUIKeyName[0x62] = name
		} else {
			iosKeyToUIKeyName[0x59+int(c)-'1'] = name
		}
		uiKeyNameToJSCode[name] = name
	}

	// Keys for backward compatibility
	oldEbitengineKeyNameToUIKeyName = map[string]string{
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

const ebitengineKeysTmpl = `{{.License}}

{{.DoNotEdit}}

package ebiten

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// A Key represents a keyboard key.
// These keys represent physical keys of US keyboard.
// For example, KeyQ represents Q key on US keyboards and ' (quote) key on Dvorak keyboards.
type Key int

// Keys.
const (
{{range $index, $name := .EbitengineKeyNamesWithoutMods}}Key{{$name}} Key = Key(ui.Key{{$name}})
{{end}}	KeyAlt     Key = Key(ui.KeyReserved0)
	KeyControl Key = Key(ui.KeyReserved1)
	KeyShift   Key = Key(ui.KeyReserved2)
	KeyMeta    Key = Key(ui.KeyReserved3)
	KeyMax     Key = KeyMeta

	// Keys for backward compatibility.
	// Deprecated: as of v2.1.
{{range $old, $new := .OldEbitengineKeyNameToUIKeyName}}Key{{$old}} Key = Key(ui.Key{{$new}})
{{end}}
)

func (k Key) isValid() bool {
	switch k {
	{{range $name := .EbitengineKeyNamesWithoutOld}}case Key{{$name}}:
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
	{{range $name := .EbitengineKeyNamesWithoutOld}}case Key{{$name}}:
		return {{$name | printf "%q"}}
	{{end}}}
	return ""
}

func keyNameToKeyCode(name string) (Key, bool) {
	switch strings.ToLower(name) {
	{{range $name := .EbitengineKeyNames}}case {{$name | printf "%q" | ToLower}}:
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
	KeyMax = KeyReserved3
)

func (k Key) String() string {
	switch k {
	{{range $index, $name := .UIKeyNames}}case Key{{$name}}:
		return {{$name | printf "Key%s" | printf "%q"}}
	{{end}}}
	return fmt.Sprintf("Key(%d)", k)
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

{{.BuildConstraints}}

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

{{.BuildConstraints}}

package ui

import (
	"syscall/js"
)

var uiKeyToJSCode = map[Key]js.Value{
{{range $name, $code := .UIKeyNameToJSCode}}Key{{$name}}: js.ValueOf({{$code | printf "%q"}}),
{{end}}
}
`

const glfwKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildConstraints}}

package glfw

const (
{{range $name, $key := .GLFWKeyNameToGLFWKey}}Key{{$name}} = Key({{$key}})
{{end}}
)
`

const mobileAndroidKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildConstraints}}

package ebitenmobileview

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var androidKeyToUIKey = map[int]ui.Key{
{{range $key, $name := .AndroidKeyToUIKeyName}}{{$key}}: ui.Key{{$name}},
{{end}}
}
`

const mobileIOSKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildConstraints}}

package ebitenmobileview

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var iosKeyToUIKey = map[int]ui.Key{
{{range $key, $name := .IOSKeyToUIKeyName}}{{$key}}: ui.Key{{$name}},
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
	// Follow the standard comment rule (https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source).
	doNotEdit := "// Code generated by genkeys.go using 'go generate'. DO NOT EDIT."

	ebitengineKeyNames := []string{}
	ebitengineKeyNamesWithoutOld := []string{}
	ebitengineKeyNamesWithoutMods := []string{}
	uiKeyNames := []string{}

	for name := range uiKeyNameToJSCode {
		uiKeyNames = append(uiKeyNames, name)
		ebitengineKeyNames = append(ebitengineKeyNames, name)
		ebitengineKeyNamesWithoutOld = append(ebitengineKeyNamesWithoutOld, name)
		ebitengineKeyNamesWithoutMods = append(ebitengineKeyNamesWithoutMods, name)
	}
	for old := range oldEbitengineKeyNameToUIKeyName {
		ebitengineKeyNames = append(ebitengineKeyNames, old)
	}
	// Keys for modifiers
	ebitengineKeyNames = append(ebitengineKeyNames, "Alt", "Control", "Shift", "Meta")
	ebitengineKeyNamesWithoutOld = append(ebitengineKeyNamesWithoutOld, "Alt", "Control", "Shift", "Meta")

	sort.Slice(ebitengineKeyNames, keyNamesLess(ebitengineKeyNames))
	sort.Slice(ebitengineKeyNamesWithoutOld, keyNamesLess(ebitengineKeyNamesWithoutOld))
	sort.Slice(ebitengineKeyNamesWithoutMods, keyNamesLess(ebitengineKeyNamesWithoutMods))
	sort.Slice(uiKeyNames, keyNamesLess(uiKeyNames))

	// TODO: Add this line for event package (#926).
	//
	//     filepath.Join("event", "keys.go"):                              eventKeysTmpl,

	for path, tmpl := range map[string]string{
		filepath.Join("internal", "glfw", "keys.go"):                   glfwKeysTmpl,
		filepath.Join("internal", "ui", "keys.go"):                     uiKeysTmpl,
		filepath.Join("internal", "ui", "keys_glfw.go"):                uiGLFWKeysTmpl,
		filepath.Join("internal", "ui", "keys_js.go"):                  uiJSKeysTmpl,
		filepath.Join("keys.go"):                                       ebitengineKeysTmpl,
		filepath.Join("mobile", "ebitenmobileview", "keys_android.go"): mobileAndroidKeysTmpl,
		filepath.Join("mobile", "ebitenmobileview", "keys_ios.go"):     mobileIOSKeysTmpl,
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
		buildConstraints := ""
		switch path {
		case filepath.Join("internal", "glfw", "keys.go"):
			buildConstraints = "//go:build darwin || freebsd || linux || netbsd || openbsd || windows"
		case filepath.Join("internal", "ui", "keys_mobile.go"):
			buildConstraints = "//go:build android || ios"
		case filepath.Join("internal", "ui", "keys_glfw.go"):
			buildConstraints = "//go:build !android && !ios && !js && !nintendosdk && !playstation5"
		}
		// NOTE: According to godoc, maps are automatically sorted by key.
		if err := tmpl.Execute(f, struct {
			License                         string
			DoNotEdit                       string
			BuildConstraints                string
			UIKeyNameToJSCode               map[string]string
			EbitengineKeyNames              []string
			EbitengineKeyNamesWithoutOld    []string
			EbitengineKeyNamesWithoutMods   []string
			GLFWKeyNameToGLFWKey            map[string]int
			UIKeyNames                      []string
			UIKeyNameToGLFWKeyName          map[string]string
			AndroidKeyToUIKeyName           map[int]string
			IOSKeyToUIKeyName               map[int]string
			OldEbitengineKeyNameToUIKeyName map[string]string
		}{
			License:                         license,
			DoNotEdit:                       doNotEdit,
			BuildConstraints:                buildConstraints,
			UIKeyNameToJSCode:               uiKeyNameToJSCode,
			EbitengineKeyNames:              ebitengineKeyNames,
			EbitengineKeyNamesWithoutOld:    ebitengineKeyNamesWithoutOld,
			EbitengineKeyNamesWithoutMods:   ebitengineKeyNamesWithoutMods,
			GLFWKeyNameToGLFWKey:            glfwKeyNameToGLFWKey,
			UIKeyNames:                      uiKeyNames,
			UIKeyNameToGLFWKeyName:          uiKeyNameToGLFWKeyName,
			AndroidKeyToUIKeyName:           androidKeyToUIKeyName,
			IOSKeyToUIKeyName:               iosKeyToUIKeyName,
			OldEbitengineKeyNameToUIKeyName: oldEbitengineKeyNameToUIKeyName,
		}); err != nil {
			log.Fatal(err)
		}
	}
}
