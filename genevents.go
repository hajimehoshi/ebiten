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

// +build ignore

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Event struct {
	Comment string
	Members []Member
}

func (e *Event) Name() string {
	return strings.SplitN(e.Comment, " ", 2)[0]
}

type Member struct {
	Comment string
	Type    string
}

func (m *Member) Name() string {
	return strings.SplitN(m.Comment, " ", 2)[0]
}

var (
	membersKeyboardKey = []Member{
		{
			Comment: "Key is the key code of the key pressed or released.",
			Type:    "Key",
		},
		{
			Comment: "Modifier is the logical-or value of the modifiers pressed together with the key.",
			Type:    "Modifier",
		},
	}
	membersGamepadButton = []Member{
		{
			Comment: "ID represents which gamepad caused the event.",
			Type:    "int",
		},
		{
			Comment: "Button is the button that was pressed on the game pad.",
			Type:    "int",
		},
		{
			Comment: "Pressure is the pressure that is applied to the gamepad button. It varies between 0.0 for not pressed, and 1.0 for completely pressed.",
			Type:    "float32",
		},
	}
	membersMouseButton = []Member{
		{
			Comment: "X is the X position of the mouse pointer. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
		{
			Comment: "Y is the Y position of the mouse pointer. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
		{
			Comment: "Button is the button on the mouse that was pressed. TODO: this should change later from an int to an enumeration type.",
			Type:    "int",
		},
		{
			Comment: "Pressure is the pressure applied on the mouse button. It varies between 0.0 for not pressed, and 1.0 for completely pressed.",
			Type:    "float32",
		},
	}
	membersMouseEnter = []Member{
		{
			Comment: "X is the X position of the mouse pointer. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
		{
			Comment: "Y is the Y position of the mouse pointer. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
	}
	membersTouch = []Member{
		{
			Comment: "ID identifies the touch that caused the touch event.",
			Type:    "int",
		},
		{
			Comment: "X is the X position of the touch. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
		{
			Comment: "Y is the Y position of the touch. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
		{
			Comment: "DeltaX is the change in X since last touch event. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
		{
			Comment: "Deltay is the change in Y since last touch event. This value is expressed in device independent pixels.",
			Type:    "float32",
		},
		{
			Comment: "Pressure of applied touch. It varies between 0.0 for not pressed, and 1.0 for completely pressed.",
			Type:    "float32",
		},
		{
			Comment: "Primary represents whether the touch event is the primary touch or not. If it is true, then it is a primary touch. If it is false then it is not.",
			Type:    "bool",
		},
	}

	events = []Event{
		{
			Comment: "KeyboardKeyCharacter is an event that occurs when a character is actually typed on the keyboard. This may be provided by an input method.",
			Members: []Member{
				{
					Comment: "Key is the key code of the key typed.",
					Type:    "Key",
				},
				{
					Comment: "Modifier is the logical-or value of the modifiers pressed together with the key.",
					Type:    "Modifier",
				},
				{
					Comment: "Character is the character that was typed.",
					Type:    "rune",
				},
			},
		},
		{
			Comment: "KeyboardKeyDown is an event that occurs when a key is pressed on the keyboard.",
			Members: membersKeyboardKey,
		},
		{
			Comment: "KeyboardKeyUp is an event that occurs when a key is released on the keyboard.",
			Members: membersKeyboardKey,
		},
		{
			Comment: "GamepadAxis is for event where an axis on a gamepad changes.",
			Members: []Member{
				{
					Comment: "ID represents which gamepad caused the event.",
					Type:    "int",
				},
				{
					Comment: "Axis is the axis of the game pad that changed position.",
					Type:    "int",
				},
				{
					Comment: "Position is the position of the axis after the change. It varies between -1.0 and 1.0.",
					Type:    "float32",
				},
			},
		},
		{
			Comment: "GamepadButtonDown is a gamepad button press event.",
			Members: membersGamepadButton,
		},
		{
			Comment: "GamepadButtonUp is a gamepad button release event.",
			Members: membersGamepadButton,
		},
		{
			Comment: "GamepadAttach happens when a new gamepad is attached.",
			Members: []Member{
				{
					Comment: "ID represents which gamepad caused the event.",
					Type:    "int",
				},
				{
					Comment: "Axes represents the amount of axes the gamepad has.",
					Type:    "int",
				},
				{
					Comment: "Buttons represents the amount of buttons the gamepad has.",
					Type:    "int",
				},
			},
		},
		{
			Comment: "GamepadDetach happens when a gamepad is detached.",
			Members: []Member{
				{
					Comment: "ID represents which gamepad caused the event.",
					Type:    "int",
				},
			},
		},
		{
			Comment: "MouseMove is a mouse movement event.",
			Members: []Member{
				{
					Comment: "X is the X position of the mouse pointer. This value is expressed in device independent pixels.",
					Type:    "float32",
				},
				{
					Comment: "Y is the Y position of the mouse pointer. This value is expressed in device independent pixels.",
					Type:    "float32",
				},
				{
					Comment: "DeltaX is the change in X since the last MouseMove event. This value is expressed in device independent pixels.",
					Type:    "float32",
				},
				{
					Comment: "DeltaY is the change in Y since the last MouseMove event. This value is expressed in device independent pixels.",
					Type:    "float32",
				},
			},
		},
		{
			Comment: "MouseWheel is a mouse wheel event.",
			Members: []Member{
				{
					Comment: "X is the X position of the mouse wheel. This value is expressed in arbitrary units. It increases when the mouse wheel is scrolled downwards, and decreases when the mouse is scrolled upwards.",
					Type:    "float32",
				},
				{
					Comment: "Y is the Y position of the mouse wheel. This value is expressed in arbitrary units. It increases when the mouse wheel is scrolled to the right, and decreases when the mouse is scrolled to the left.",
					Type:    "float32",
				},
				{
					Comment: "DeltaX is the change in X since the last MouseWheel event. This value is expressed in arbitrary units. It is positive when the mouse wheel is scrolled downwards, and negative when the mouse is scrolled upwards.",
					Type:    "float32",
				},
				{
					Comment: "DeltaY is the change in Y since the last MouseWheel event. This value is expressed in arbitrary units. It is positive when the mouse wheel is scrolled to the right, and negative when the mouse is scrolled to the left.",
					Type:    "float32",
				},
			},
		},
		{
			Comment: "MouseButtonDown is a mouse button press event.",
			Members: membersMouseButton,
		},
		{
			Comment: "MouseButtonUp is a mouse button release event.",
			Members: membersMouseButton,
		},
		{
			Comment: "MouseEnter occurs when the mouse enters the view window.",
			Members: membersMouseEnter,
		},
		{
			Comment: "MouseLeave occurs when the mouse leaves the view window.",
			Members: membersMouseEnter,
		},
		{
			Comment: "TouchBegin occurs when a touch begins.",
			Members: membersTouch,
		},
		{
			Comment: "TouchMove occurs when a touch moved, or in other words, is dragged.",
			Members: membersTouch,
		},
		{
			Comment: "TouchEnd occurs when a touch ends.",
			Members: membersTouch,
		},
		{
			Comment: "TouchCancel occurs when a touch is canceled. This can happen in various situations, depending on the underlying platform, for example when the application loses focus.",
			Members: []Member{
				{
					Comment: "ID identifies the touch that caused the touch event.",
					Type:    "int",
				},
			},
		},
		{
			Comment: "ViewUpdate occurs when the application is ready to update the next frame on the view port.",
		},
		{
			Comment: "ViewSize occurs when the size of the application's view port changes.",
			Members: []Member{
				{
					Comment: "Width is the width of the view. This value is expressed in device independent pixels.",
					Type:    "int",
				},
				{
					Comment: "Height is the height of the view. This value is expressed in device independent pixels.",
					Type:    "int",
				},
			},
		},
	}
)

const (
	license = `// Copyright 2013 The Ebiten Authors
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
	doNotEdit = "// Code generated by genevents.go using 'go generate'. DO NOT EDIT."
)

var eventTmpl = template.Must(template.New("event.go").Parse(`{{.License}}

{{.DoNotEdit}}

package {{.Package}}

type Event interface{}

{{range .Events}}// {{.Comment}}
type {{.Name}} struct {
{{range .Members}}	// {{.Comment}}
	{{.Name}} {{.Type}}

{{end}}
}

{{end}}
`))

var chanTmpl = template.Must(template.New("chan.go").Parse(`{{.License}}

{{.DoNotEdit}}

package {{.Package}}

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/driver"
)

func convertCh(driverCh chan driver.Event) (chan Event) {
	ch := make(chan Event)
	go func() {
		defer close(ch)

		for v := range driverCh {
			switch v := v.(type) {
			{{range .Events}}case driver.{{.Name}}:
				ch <- {{.Name}}(v)
			{{end}}default:
				panic(fmt.Sprintf("event: unknown event: %v", v))
			}
		}
	}()
	return ch
}
`))

func main() {
	for path, tmpl := range map[string]*template.Template{
		filepath.Join("event", "event.go"):              eventTmpl,
		filepath.Join("event", "chan.go"):               chanTmpl,
		filepath.Join("internal", "driver", "event.go"): eventTmpl,
	} {
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		tokens := strings.Split(path, string(filepath.Separator))
		pkg := tokens[len(tokens)-2]

		if err := tmpl.Execute(f, struct {
			License   string
			DoNotEdit string
			Package   string
			Events    []Event
		}{
			License:   license,
			DoNotEdit: doNotEdit,
			Package:   pkg,
			Events:    events,
		}); err != nil {
			log.Fatal(err)
		}
	}
}
