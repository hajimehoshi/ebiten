// Copyright 2018 The Ebiten Authors
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

package glfw

import (
	"github.com/go-gl/glfw/v3.2/glfw"
)

type (
	Action       = glfw.Action
	Hint         = glfw.Hint
	InputMode    = glfw.InputMode
	Joystick     = glfw.Joystick
	Key          = glfw.Key
	ModifierKey  = glfw.ModifierKey
	MouseButton  = glfw.MouseButton
	MonitorEvent = glfw.MonitorEvent
)

type (
	Monitor = glfw.Monitor
	VidMode = glfw.VidMode
	Window  = glfw.Window
)

var (
	CreateWindow       = glfw.CreateWindow
	GetJoystickAxes    = glfw.GetJoystickAxes
	GetJoystickButtons = glfw.GetJoystickButtons
	GetMonitors        = glfw.GetMonitors
	GetPrimaryMonitor  = glfw.GetPrimaryMonitor
	Init               = glfw.Init
	JoystickPresent    = glfw.JoystickPresent
	PollEvents         = glfw.PollEvents
	SetMonitorCallback = glfw.SetMonitorCallback
	SwapInterval       = glfw.SwapInterval
	Terminate          = glfw.Terminate
	WindowHint         = glfw.WindowHint
)
