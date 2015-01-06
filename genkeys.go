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

package main

import (
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var keyCodeToName map[int]string

func init() {
	keyCodeToName = map[int]string{
		0xBC: "KeyComma",
		0xBE: "KeyPeriod",
		0x12: "KeyLeftAlt",
		0x14: "KeyCapsLock",
		0x11: "KeyLeftControl",
		0x10: "KeyLeftShift",
		0x0D: "KeyEnter",
		0x20: "KeySpace",
		0x09: "KeyTab",
		0x2E: "KeyDelete",
		0x23: "KeyEnd",
		0x24: "KeyHome",
		0x2D: "KeyInsert",
		0x22: "KeyPageDown",
		0x21: "KeyPageUp",
		0x28: "KeyDown",
		0x25: "KeyLeft",
		0x27: "KeyRight",
		0x26: "KeyUp",
		0x1B: "KeyEscape",
	}
	// ASCII: 0 - 9
	for c := '0'; c <= '9'; c++ {
		keyCodeToName[int(c)] = "Key" + string(c)
	}
	// ASCII: A - Z
	for c := 'A'; c <= 'Z'; c++ {
		keyCodeToName[int(c)] = "Key" + string(c)
	}
	// Function keys
	for i := 1; i <= 12; i++ {
		keyCodeToName[0x70+i-1] = "KeyF" + strconv.Itoa(i)
	}
}

const ebitenKeysTmpl = `{{.License}}

package ebiten


import (
	"github.com/hajimehoshi/ebiten/internal/ui"
)

// A Key represents a keyboard key.
type Key int

// Keys
const (
{{range $index, $key := .KeyNames}}{{$key}} = Key(ui.{{$key}})
{{end}}
)
`

const uiKeysTmpl = `{{.License}}

package ui

type Key int

const (
{{range $index, $key := .KeyNames}}{{$key}}{{if eq $index 0}} Key = iota{{end}}
{{end}}
)
`

const uiKeysGlfwTmpl = `{{.License}}

// +build !js

package ui

import (
	glfw "github.com/go-gl/glfw3"
)

var glfwKeyCodeToKey = map[glfw.Key]Key{
{{range $index, $key := .KeyNames}}glfw.{{$key}}: {{$key}},
{{end}}
}
`

const uiKeysJSTmpl = `{{.License}}

// +build js

package ui

var keyCodeToKey = map[int]Key{
{{range $code, $name := .Keys}}{{$code}}: {{$name}},
{{end}}
}
`

func main() {
	l, err := ioutil.ReadFile("license.txt")
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(l), "\n")
	license := "// " + strings.Join(lines[:len(lines)-1], "\n// ")

	names := []string{}
	codes := []int{}
	for code, name := range keyCodeToName {
		names = append(names, name)
		codes = append(codes, code)
	}
	sort.Strings(names)
	sort.Ints(codes)

	for path, tmpl := range map[string]string{
		"keys.go":                  ebitenKeysTmpl,
		"internal/ui/keys.go":      uiKeysTmpl,
		"internal/ui/keys_glfw.go": uiKeysGlfwTmpl,
		"internal/ui/keys_js.go":   uiKeysJSTmpl,
	} {
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		tmpl, err := template.New(path).Parse(tmpl)
		if err != nil {
			log.Fatal(err)
		}
		// NOTE: According to godoc, maps are automatically sorted by key.
		tmpl.Execute(f, map[string]interface{}{
			"License":  license,
			"Keys":     keyCodeToName,
			"Codes":    codes,
			"KeyNames": names,
		})
	}
}
