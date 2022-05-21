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

//go:build !windows && !js
// +build !windows,!js

package glfw

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

type (
	CharModsCallback        = glfw.CharModsCallback
	CloseCallback           = glfw.CloseCallback
	FramebufferSizeCallback = glfw.FramebufferSizeCallback
	MonitorCallback         = glfw.MonitorCallback
	ScrollCallback          = glfw.ScrollCallback
	SizeCallback            = glfw.SizeCallback
)

type VidMode struct {
	Width       int
	Height      int
	RedBits     int
	GreenBits   int
	BlueBits    int
	RefreshRate int
}
