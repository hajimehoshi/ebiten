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

// +build ignore

// Note:
//   * Respect GLFW key names
//   * https://developer.mozilla.org/en-US/docs/Web/API/KeyboardEvent.keyCode
//   * It is best to replace keyCode with code, but many browsers don't implement it.

package main

import (
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-gl/glfw/v3.2/glfw"
)

var (
	nameToGLFWKeys    map[string]glfw.Key
	nameToJSKeyCodes  map[string][]string
	keyCodeToNameEdge map[int]string
)

func init() {
	nameToGLFWKeys = map[string]glfw.Key{
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
		"Last":         glfw.KeyLast,
	}
	nameToJSKeyCodes = map[string][]string{
		"Comma":        {"Comma"},
		"Period":       {"Period"},
		"Alt":          {"AltLeft", "AltRight"},
		"CapsLock":     {"CapsLock"},
		"Control":      {"ControlLeft", "ControlRight"},
		"Shift":        {"ShiftLeft", "ShiftRight"},
		"Enter":        {"Enter"},
		"Space":        {"Space"},
		"Tab":          {"Tab"},
		"Delete":       {"Delete"},
		"End":          {"End"},
		"Home":         {"Home"},
		"Insert":       {"Insert"},
		"PageDown":     {"PageDown"},
		"PageUp":       {"PageUp"},
		"Down":         {"ArrowDown"},
		"Left":         {"ArrowLeft"},
		"Right":        {"ArrowRight"},
		"Up":           {"ArrowUp"},
		"Escape":       {"Escape"},
		"Backspace":    {"Backspace"},
		"Apostrophe":   {"Quote"},
		"Minus":        {"Minus"},
		"Slash":        {"Slash"},
		"Semicolon":    {"Semicolon"},
		"Equal":        {"Equal"},
		"LeftBracket":  {"BracketLeft"},
		"Backslash":    {"Backslash"},
		"RightBracket": {"BracketRight"},
		"GraveAccent":  {"Backquote"},
		"NumLock":      {"NumLock"},
		"Pause":        {"Pause"},
		"PrintScreen":  {"PrintScreen"},
		"ScrollLock":   {"ScrollLock"},
		"Menu":         {"ContextMenu"},
	}
	// ASCII: 0 - 9
	for c := '0'; c <= '9'; c++ {
		nameToGLFWKeys[string(c)] = glfw.Key0 + glfw.Key(c) - '0'
		nameToJSKeyCodes[string(c)] = []string{"Digit" + string(c)}
	}
	// ASCII: A - Z
	for c := 'A'; c <= 'Z'; c++ {
		nameToGLFWKeys[string(c)] = glfw.KeyA + glfw.Key(c) - 'A'
		nameToJSKeyCodes[string(c)] = []string{"Key" + string(c)}
	}
	// Function keys
	for i := 1; i <= 12; i++ {
		name := "F" + strconv.Itoa(i)
		nameToGLFWKeys[name] = glfw.KeyF1 + glfw.Key(i) - 1
		nameToJSKeyCodes[name] = []string{name}
	}
	// Numpad
	// https://www.w3.org/TR/uievents-code/#key-numpad-section
	for c := '0'; c <= '9'; c++ {
		name := "KP" + string(c)
		nameToGLFWKeys[name] = glfw.KeyKP0 + glfw.Key(c) - '0'
		nameToJSKeyCodes[name] = []string{"Numpad" + string(c)}
	}

	nameToGLFWKeys["KPDecimal"] = glfw.KeyKPDecimal
	nameToGLFWKeys["KPDivide"] = glfw.KeyKPDivide
	nameToGLFWKeys["KPMultiply"] = glfw.KeyKPMultiply
	nameToGLFWKeys["KPSubtract"] = glfw.KeyKPSubtract
	nameToGLFWKeys["KPAdd"] = glfw.KeyKPAdd
	nameToGLFWKeys["KPEnter"] = glfw.KeyKPEnter
	nameToGLFWKeys["KPEqual"] = glfw.KeyKPEqual

	nameToJSKeyCodes["KPDecimal"] = []string{"NumpadDecimal"}
	nameToJSKeyCodes["KPDivide"] = []string{"NumpadDivide"}
	nameToJSKeyCodes["KPMultiply"] = []string{"NumpadMultiply"}
	nameToJSKeyCodes["KPSubtract"] = []string{"NumpadSubtract"}
	nameToJSKeyCodes["KPAdd"] = []string{"NumpadAdd"}
	nameToJSKeyCodes["KPEnter"] = []string{"NumpadEnter"}
	nameToJSKeyCodes["KPEqual"] = []string{"NumpadEqual"}
}

func init() {
	keyCodeToNameEdge = map[int]string{
		0xbc: "Comma",
		0xbe: "Period",
		0x12: "Alt",
		0x14: "CapsLock",
		0x11: "Control",
		0x10: "Shift",
		0x0D: "Enter",
		0x20: "Space",
		0x09: "Tab",
		0x2E: "Delete",
		0x23: "End",
		0x24: "Home",
		0x2D: "Insert",
		0x22: "PageDown",
		0x21: "PageUp",
		0x28: "Down",
		0x25: "Left",
		0x27: "Right",
		0x26: "Up",
		0x1B: "Escape",
		0xde: "Apostrophe",
		0xbd: "Minus",
		0xbf: "Slash",
		0xba: "Semicolon",
		0xbb: "Equal",
		0xdb: "LeftBracket",
		0xdc: "Backslash",
		0xdd: "RightBracket",
		0xc0: "GraveAccent",
		0x08: "Backspace",
		0x90: "NumLock",
		0x6e: "KPDecimal",
		0x6f: "KPDivide",
		0x6a: "KPMultiply",
		0x6d: "KPSubtract",
		0x6b: "KPAdd",
		0x13: "Pause",
		0x91: "ScrollLock",
		0x5d: "Menu",

		// On Edge, this key does not work. PrintScreen works only on keyup event.
		// 0x2C: "PrintScreen",

		// On Edge, it is impossible to tell KPEnter and Enter / KPEqual and Equal.
		// 0x0d: "KPEnter",
		// 0x0c: "KPEqual",
	}
	// ASCII: 0 - 9
	for c := '0'; c <= '9'; c++ {
		keyCodeToNameEdge[int(c)] = string(c)
	}
	// ASCII: A - Z
	for c := 'A'; c <= 'Z'; c++ {
		keyCodeToNameEdge[int(c)] = string(c)
	}
	// Function keys
	for i := 1; i <= 12; i++ {
		keyCodeToNameEdge[0x70+i-1] = "F" + strconv.Itoa(i)
	}
	// Numpad keys
	for c := '0'; c <= '9'; c++ {
		keyCodeToNameEdge[0x60+int(c-'0')] = "KP" + string(c)
	}
}

const ebitenKeysTmpl = `{{.License}}

{{.DoNotEdit}}

package ebiten

import (
	"strings"

	"github.com/hajimehoshi/ebiten/internal/input"
)

// A Key represents a keyboard key.
// These keys represent pysical keys of US keyboard.
// For example, KeyQ represents Q key on US keyboards and ' (quote) key on Dvorak keyboards.
type Key int

// Keys.
const (
{{range $index, $name := .KeyNames}}Key{{$name}} Key = Key(input.Key{{$name}})
{{end}}	KeyMax Key = Key{{.LastKeyName}}
)

// String returns a string representing the key.
//
// If k is an undefined key, String returns an empty string.
func (k Key) String() string {
	switch k {
	{{range $name := .KeyNames}}case Key{{$name}}:
		return {{$name | printf "%q"}}
	{{end}}}
	return ""
}

func keyNameToKey(name string) (Key, bool) {
	switch strings.ToLower(name) {
	{{range $name := .KeyNames}}case {{$name | printf "%q" | ToLower}}:
		return Key{{$name}}, true
	{{end}}}
	return 0, false
}
`

const inputKeysTmpl = `{{.License}}

{{.DoNotEdit}}

package input

type Key int

const (
{{range $index, $name := .KeyNames}}Key{{$name}}{{if eq $index 0}} Key = iota{{end}}
{{end}}
)
`

const inputKeysGlfwTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package input

import (
	"github.com/hajimehoshi/ebiten/internal/glfw"
)

var glfwKeyCodeToKey = map[glfw.Key]Key{
{{range $index, $name := .KeyNamesWithoutMods}}glfw.Key{{$name}}: Key{{$name}},
{{end}}
	glfw.KeyLeftAlt:      KeyAlt,
	glfw.KeyRightAlt:     KeyAlt,
	glfw.KeyLeftControl:  KeyControl,
	glfw.KeyRightControl: KeyControl,
	glfw.KeyLeftShift:    KeyShift,
	glfw.KeyRightShift:   KeyShift,
}
`

const inputKeysJSTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package input

var keyToCodes = map[Key][]string{
{{range $name, $codes := .NameToJSKeyCodes}}Key{{$name}}: []string{
{{range $code := $codes}}"{{$code}}",{{end}}
},
{{end}}
}

var keyCodeToKeyEdge = map[int]Key{
{{range $code, $name := .KeyCodeToNameEdge}}{{$code}}: Key{{$name}},
{{end}}
}
`

const glfwKeysTmpl = `{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package glfw

const (
{{range $name, $key := .NameToGLFWKeys}}Key{{$name}} = Key({{$key}})
{{end}}
)
`

type KeyNames []string

func (k KeyNames) digit(name string) int {
	if len(name) != 1 {
		return -1
	}
	c := name[0]
	if c < '0' || '9' < c {
		return -1
	}
	return int(c - '0')
}

func (k KeyNames) alphabet(name string) rune {
	if len(name) != 1 {
		return -1
	}
	c := rune(name[0])
	if c < 'A' || 'Z' < c {
		return -1
	}
	return c
}

func (k KeyNames) function(name string) int {
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

func (k KeyNames) Len() int {
	return len(k)
}

func (k KeyNames) Less(i, j int) bool {
	k0, k1 := k[i], k[j]
	d0, d1 := k.digit(k0), k.digit(k1)
	a0, a1 := k.alphabet(k0), k.alphabet(k1)
	f0, f1 := k.function(k0), k.function(k1)
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

func (k KeyNames) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

const license = `// Copyright 2013 The Ebiten Authors
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

	namesSet := map[string]struct{}{}
	namesWithoutModsSet := map[string]struct{}{}
	codes := []string{}
	for name, cs := range nameToJSKeyCodes {
		namesSet[name] = struct{}{}
		codes = append(codes, cs...)
		if name != "Alt" && name != "Control" && name != "Shift" {
			namesWithoutModsSet[name] = struct{}{}
		}
	}
	names := []string{}
	namesWithoutMods := []string{}
	for n := range namesSet {
		names = append(names, n)
	}
	for n := range namesWithoutModsSet {
		namesWithoutMods = append(namesWithoutMods, n)
	}

	sort.Sort(KeyNames(names))
	sort.Sort(KeyNames(namesWithoutMods))
	sort.Strings(codes)

	for path, tmpl := range map[string]string{
		"keys.go":                     ebitenKeysTmpl,
		"internal/input/keys.go":      inputKeysTmpl,
		"internal/input/keys_glfw.go": inputKeysGlfwTmpl,
		"internal/input/keys_js.go":   inputKeysJSTmpl,
		"internal/glfw/keys.go":       glfwKeysTmpl,
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
		case "internal/input/keys_glfw.go":
			buildTag = "// +build darwin freebsd linux windows" +
				"\n// +build !js" +
				"\n// +build !android" +
				"\n// +build !ios"
		case "internal/input/keys_js.go":
			buildTag = "// +build js"
		}
		// NOTE: According to godoc, maps are automatically sorted by key.
		if err := tmpl.Execute(f, map[string]interface{}{
			"License":             license,
			"DoNotEdit":           doNotEdit,
			"BuildTag":            buildTag,
			"NameToJSKeyCodes":    nameToJSKeyCodes,
			"KeyCodeToNameEdge":   keyCodeToNameEdge,
			"Codes":               codes,
			"KeyNames":            names,
			"LastKeyName":         names[len(names)-1],
			"KeyNamesWithoutMods": namesWithoutMods,
			"NameToGLFWKeys":      nameToGLFWKeys,
		}); err != nil {
			log.Fatal(err)
		}
	}
}
