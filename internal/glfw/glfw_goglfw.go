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

//go:build windows

package glfw

import (
	"errors"
	"image"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2/internal/goglfw"
)

type Cursor goglfw.Cursor

func CreateStandardCursor(shape StandardCursor) *Cursor {
	c, err := goglfw.CreateStandardCursor(goglfw.StandardCursor(shape))
	if err != nil {
		panic(err)
	}
	return (*Cursor)(c)
}

type Monitor goglfw.Monitor

func (m *Monitor) GetContentScale() (float32, float32, error) {
	return (*goglfw.Monitor)(m).GetContentScale()
}

func (m *Monitor) GetPos() (int, int) {
	x, y, err := (*goglfw.Monitor)(m).GetPos()
	if err != nil {
		panic(err)
	}
	return x, y
}

func (m *Monitor) GetVideoMode() *VidMode {
	v, err := (*goglfw.Monitor)(m).GetVideoMode()
	if err != nil {
		panic(err)
	}
	return (*VidMode)(v)
}

func (m *Monitor) GetName() string {
	v, err := (*goglfw.Monitor)(m).GetName()
	if err != nil {
		panic(err)
	}
	return v
}

type Window goglfw.Window

func (w *Window) Destroy() {
	if err := (*goglfw.Window)(w).Destroy(); err != nil {
		panic(err)
	}
}

func (w *Window) Focus() {
	if err := (*goglfw.Window)(w).Focus(); err != nil {
		panic(err)
	}
}

func (w *Window) GetAttrib(attrib Hint) int {
	r, err := (*goglfw.Window)(w).GetAttrib(goglfw.Hint(attrib))
	if err != nil {
		panic(err)
	}
	return r
}

func (w *Window) GetCursorPos() (x, y float64) {
	x, y, err := (*goglfw.Window)(w).GetCursorPos()
	if err != nil {
		panic(err)
	}
	return x, y
}

func (w *Window) GetInputMode(mode InputMode) int {
	r, err := (*goglfw.Window)(w).GetInputMode(goglfw.InputMode(mode))
	if err != nil {
		panic(err)
	}
	return r
}

func (w *Window) GetKey(key Key) Action {
	r, err := (*goglfw.Window)(w).GetKey(goglfw.Key(key))
	if err != nil {
		panic(err)
	}
	return Action(r)
}

func (w *Window) GetMonitor() *Monitor {
	m, err := (*goglfw.Window)(w).GetMonitor()
	if err != nil {
		panic(err)
	}
	return (*Monitor)(m)
}

func (w *Window) GetMouseButton(button MouseButton) Action {
	r, err := (*goglfw.Window)(w).GetMouseButton(goglfw.MouseButton(button))
	if err != nil {
		panic(err)
	}
	return Action(r)
}

func (w *Window) GetPos() (int, int) {
	x, y, err := (*goglfw.Window)(w).GetPos()
	if err != nil {
		panic(err)
	}
	return x, y
}

func (w *Window) GetSize() (int, int) {
	width, height, err := (*goglfw.Window)(w).GetSize()
	if err != nil {
		panic(err)
	}
	return width, height
}

func (w *Window) Hide() {
	if err := (*goglfw.Window)(w).Hide(); err != nil {
		panic(err)
	}
}

func (w *Window) Iconify() {
	if err := (*goglfw.Window)(w).Iconify(); err != nil {
		panic(err)
	}
}

func (w *Window) MakeContextCurrent() {
	if err := (*goglfw.Window)(w).MakeContextCurrent(); err != nil {
		panic(err)
	}
}

func (w *Window) Maximize() {
	if err := (*goglfw.Window)(w).Maximize(); err != nil {
		panic(err)
	}
}

func (w *Window) Restore() {
	if err := (*goglfw.Window)(w).Restore(); err != nil {
		panic(err)
	}
}

func (w *Window) SetAttrib(attrib Hint, value int) {
	if err := (*goglfw.Window)(w).SetAttrib(goglfw.Hint(attrib), value); err != nil {
		panic(err)
	}
}

func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (previous CharModsCallback) {
	f, err := (*goglfw.Window)(w).SetCharModsCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetCloseCallback(cbfun CloseCallback) (previous CloseCallback) {
	f, err := (*goglfw.Window)(w).SetCloseCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetCursor(cursor *Cursor) {
	if err := (*goglfw.Window)(w).SetCursor((*goglfw.Cursor)(cursor)); err != nil {
		panic(err)
	}
}

func (w *Window) SetCursorPos(xpos, ypos float64) {
	if err := (*goglfw.Window)(w).SetCursorPos(xpos, ypos); err != nil {
		panic(err)
	}
}

func (w *Window) SetDropCallback(cbfun DropCallback) (previous DropCallback) {
	f, err := (*goglfw.Window)(w).SetDropCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback) {
	f, err := (*goglfw.Window)(w).SetFramebufferSizeCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback) {
	f, err := (*goglfw.Window)(w).SetScrollCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetShouldClose(value bool) {
	if err := (*goglfw.Window)(w).SetShouldClose(value); err != nil {
		panic(err)
	}
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous SizeCallback) {
	f, err := (*goglfw.Window)(w).SetSizeCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) {
	if err := (*goglfw.Window)(w).SetSizeLimits(minw, minh, maxw, maxh); err != nil {
		panic(err)
	}
}

func (w *Window) SetAspectRatio(numer, denom int) {
	if err := (*goglfw.Window)(w).SetAspectRatio(numer, denom); err != nil {
		panic(err)
	}
}

func (w *Window) SetIcon(images []image.Image) {
	gimgs := make([]*goglfw.Image, len(images))
	for i, img := range images {
		b := img.Bounds()
		m := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
		gimgs[i] = &goglfw.Image{
			Width:  b.Dx(),
			Height: b.Dy(),
			Pixels: m.Pix,
		}
	}

	if err := (*goglfw.Window)(w).SetIcon(gimgs); err != nil {
		panic(err)
	}
}

func (w *Window) SetInputMode(mode InputMode, value int) {
	if err := (*goglfw.Window)(w).SetInputMode(goglfw.InputMode(mode), value); err != nil {
		panic(err)
	}
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	if err := (*goglfw.Window)(w).SetMonitor((*goglfw.Monitor)(monitor), xpos, ypos, width, height, refreshRate); err != nil {
		panic(err)
	}
}

func (w *Window) SetPos(xpos, ypos int) {
	if err := (*goglfw.Window)(w).SetPos(xpos, ypos); err != nil {
		panic(err)
	}
}

func (w *Window) SetSize(width, height int) {
	if err := (*goglfw.Window)(w).SetSize(width, height); err != nil {
		panic(err)
	}
}

func (w *Window) SetTitle(title string) {
	if err := (*goglfw.Window)(w).SetTitle(title); err != nil {
		panic(err)
	}
}

func (w *Window) ShouldClose() bool {
	r, err := (*goglfw.Window)(w).ShouldClose()
	if err != nil {
		panic(err)
	}
	return r
}

func (w *Window) Show() {
	if err := (*goglfw.Window)(w).Show(); err != nil {
		panic(err)
	}
}

func (w *Window) SwapBuffers() {
	if err := (*goglfw.Window)(w).SwapBuffers(); err != nil {
		panic(err)
	}
}

func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (*Window, error) {
	w, err := goglfw.CreateWindow(width, height, title, (*goglfw.Monitor)(monitor), (*goglfw.Window)(share))
	// TODO: acceptError(APIUnavailable, VersionUnavailable)?
	return (*Window)(w), err
}

func GetKeyName(key Key, scancode int) string {
	name, err := goglfw.GetKeyName(goglfw.Key(key), scancode)
	if err != nil {
		panic(err)
	}
	return name
}

func GetMonitors() []*Monitor {
	ms, err := goglfw.GetMonitors()
	if err != nil {
		panic(err)
	}
	result := make([]*Monitor, 0, len(ms))
	for _, m := range ms {
		result = append(result, (*Monitor)(m))
	}
	return result
}

func GetPrimaryMonitor() *Monitor {
	m, err := goglfw.GetPrimaryMonitor()
	if err != nil {
		panic(err)
	}
	return (*Monitor)(m)
}

func Init() error {
	// InvalidValue can happen when specific joysticks are used. This issue
	// will be fixed in GLFW 3.3.5. As a temporary fix, ignore this error.
	// See go-gl/glfw#292, go-gl/glfw#324, and glfw/glfw#1763
	// (#1229).
	// TODO: acceptError(APIUnavailable, InvalidValue)?
	err := goglfw.Init()
	if err != nil && !errors.Is(err, goglfw.InvalidValue) {
		return err
	}
	return nil
}

func PollEvents() {
	if err := goglfw.PollEvents(); err != nil && !errors.Is(err, goglfw.InvalidValue) {
		panic(err)
	}
}

func PostEmptyEvent() {
	if err := goglfw.PostEmptyEvent(); err != nil {
		panic(err)
	}
}

func SetMonitorCallback(cbfun MonitorCallback) MonitorCallback {
	f, err := goglfw.SetMonitorCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func SwapInterval(interval int) {
	if err := goglfw.SwapInterval(interval); err != nil {
		panic(err)
	}
}

func Terminate() {
	if err := goglfw.Terminate(); err != nil {
		panic(err)
	}
}

func WaitEvents() {
	if err := goglfw.WaitEvents(); err != nil {
		panic(err)
	}
}

func WindowHint(target Hint, hint int) {
	if err := goglfw.WindowHint(goglfw.Hint(target), hint); err != nil {
		panic(err)
	}
}
