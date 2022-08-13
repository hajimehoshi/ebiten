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
	"errors"
	"image"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2/internal/glfwwin"
)

type Cursor glfwwin.Cursor

func CreateStandardCursor(shape StandardCursor) *Cursor {
	c, err := glfwwin.CreateStandardCursor(glfwwin.StandardCursor(shape))
	if err != nil {
		panic(err)
	}
	return (*Cursor)(c)
}

type Monitor glfwwin.Monitor

func (m *Monitor) GetContentScale() (float32, float32) {
	sx, sy, err := (*glfwwin.Monitor)(m).GetContentScale()
	if err != nil {
		panic(err)
	}
	return sx, sy
}

func (m *Monitor) GetPos() (int, int) {
	x, y, err := (*glfwwin.Monitor)(m).GetPos()
	if err != nil {
		panic(err)
	}
	return x, y
}

func (m *Monitor) GetVideoMode() *VidMode {
	v, err := (*glfwwin.Monitor)(m).GetVideoMode()
	if err != nil {
		panic(err)
	}
	return (*VidMode)(v)
}

type Window glfwwin.Window

func (w *Window) Destroy() {
	if err := (*glfwwin.Window)(w).Destroy(); err != nil {
		panic(err)
	}
}

func (w *Window) Focus() {
	if err := (*glfwwin.Window)(w).Focus(); err != nil {
		panic(err)
	}
}

func (w *Window) GetAttrib(attrib Hint) int {
	r, err := (*glfwwin.Window)(w).GetAttrib(glfwwin.Hint(attrib))
	if err != nil {
		panic(err)
	}
	return r
}

func (w *Window) GetCursorPos() (x, y float64) {
	x, y, err := (*glfwwin.Window)(w).GetCursorPos()
	if err != nil {
		panic(err)
	}
	return x, y
}

func (w *Window) GetInputMode(mode InputMode) int {
	r, err := (*glfwwin.Window)(w).GetInputMode(glfwwin.InputMode(mode))
	if err != nil {
		panic(err)
	}
	return r
}

func (w *Window) GetKey(key Key) Action {
	r, err := (*glfwwin.Window)(w).GetKey(glfwwin.Key(key))
	if err != nil {
		panic(err)
	}
	return Action(r)
}

func (w *Window) GetMonitor() *Monitor {
	m, err := (*glfwwin.Window)(w).GetMonitor()
	if err != nil {
		panic(err)
	}
	return (*Monitor)(m)
}

func (w *Window) GetMouseButton(button MouseButton) Action {
	r, err := (*glfwwin.Window)(w).GetMouseButton(glfwwin.MouseButton(button))
	if err != nil {
		panic(err)
	}
	return Action(r)
}

func (w *Window) GetPos() (int, int) {
	x, y, err := (*glfwwin.Window)(w).GetPos()
	if err != nil {
		panic(err)
	}
	return x, y
}

func (w *Window) GetSize() (int, int) {
	width, height, err := (*glfwwin.Window)(w).GetSize()
	if err != nil {
		panic(err)
	}
	return width, height
}

func (w *Window) Hide() {
	if err := (*glfwwin.Window)(w).Hide(); err != nil {
		panic(err)
	}
}

func (w *Window) Iconify() {
	if err := (*glfwwin.Window)(w).Iconify(); err != nil {
		panic(err)
	}
}

func (w *Window) MakeContextCurrent() {
	if err := (*glfwwin.Window)(w).MakeContextCurrent(); err != nil {
		panic(err)
	}
}

func (w *Window) Maximize() {
	if err := (*glfwwin.Window)(w).Maximize(); err != nil {
		panic(err)
	}
}

func (w *Window) Restore() {
	if err := (*glfwwin.Window)(w).Restore(); err != nil {
		panic(err)
	}
}

func (w *Window) SetAttrib(attrib Hint, value int) {
	if err := (*glfwwin.Window)(w).SetAttrib(glfwwin.Hint(attrib), value); err != nil {
		panic(err)
	}
}

func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (previous CharModsCallback) {
	f, err := (*glfwwin.Window)(w).SetCharModsCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetCloseCallback(cbfun CloseCallback) (previous CloseCallback) {
	f, err := (*glfwwin.Window)(w).SetCloseCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetCursor(cursor *Cursor) {
	if err := (*glfwwin.Window)(w).SetCursor((*glfwwin.Cursor)(cursor)); err != nil {
		panic(err)
	}
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback) {
	f, err := (*glfwwin.Window)(w).SetFramebufferSizeCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback) {
	f, err := (*glfwwin.Window)(w).SetScrollCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetShouldClose(value bool) {
	if err := (*glfwwin.Window)(w).SetShouldClose(value); err != nil {
		panic(err)
	}
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous SizeCallback) {
	f, err := (*glfwwin.Window)(w).SetSizeCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) {
	if err := (*glfwwin.Window)(w).SetSizeLimits(minw, minh, maxw, maxh); err != nil {
		panic(err)
	}
}

func (w *Window) SetAspectRatio(numer, denom int) {
	if err := (*glfwwin.Window)(w).SetAspectRatio(numer, denom); err != nil {
		panic(err)
	}
}

func (w *Window) SetIcon(images []image.Image) {
	gimgs := make([]*glfwwin.Image, len(images))
	for i, img := range images {
		b := img.Bounds()
		m := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
		gimgs[i] = &glfwwin.Image{
			Width:  b.Dx(),
			Height: b.Dy(),
			Pixels: m.Pix,
		}
	}

	if err := (*glfwwin.Window)(w).SetIcon(gimgs); err != nil {
		panic(err)
	}
}

func (w *Window) SetInputMode(mode InputMode, value int) {
	if err := (*glfwwin.Window)(w).SetInputMode(glfwwin.InputMode(mode), value); err != nil {
		panic(err)
	}
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	if err := (*glfwwin.Window)(w).SetMonitor((*glfwwin.Monitor)(monitor), xpos, ypos, width, height, refreshRate); err != nil {
		panic(err)
	}
}

func (w *Window) SetPos(xpos, ypos int) {
	if err := (*glfwwin.Window)(w).SetPos(xpos, ypos); err != nil {
		panic(err)
	}
}

func (w *Window) SetSize(width, height int) {
	if err := (*glfwwin.Window)(w).SetSize(width, height); err != nil {
		panic(err)
	}
}

func (w *Window) SetTitle(title string) {
	if err := (*glfwwin.Window)(w).SetTitle(title); err != nil {
		panic(err)
	}
}

func (w *Window) ShouldClose() bool {
	r, err := (*glfwwin.Window)(w).ShouldClose()
	if err != nil {
		panic(err)
	}
	return r
}

func (w *Window) Show() {
	if err := (*glfwwin.Window)(w).Show(); err != nil {
		panic(err)
	}
}

func (w *Window) SwapBuffers() {
	if err := (*glfwwin.Window)(w).SwapBuffers(); err != nil {
		panic(err)
	}
}

func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (*Window, error) {
	w, err := glfwwin.CreateWindow(width, height, title, (*glfwwin.Monitor)(monitor), (*glfwwin.Window)(share))
	// TODO: acceptError(APIUnavailable, VersionUnavailable)?
	return (*Window)(w), err
}

func GetMonitors() []*Monitor {
	ms, err := glfwwin.GetMonitors()
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
	m, err := glfwwin.GetPrimaryMonitor()
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
	err := glfwwin.Init()
	if err != nil && !errors.Is(err, glfwwin.InvalidValue) {
		return err
	}
	return nil
}

func PollEvents() {
	if err := glfwwin.PollEvents(); err != nil && !errors.Is(err, glfwwin.InvalidValue) {
		panic(err)
	}
}

func PostEmptyEvent() {
	if err := glfwwin.PostEmptyEvent(); err != nil {
		panic(err)
	}
}

func SetMonitorCallback(cbfun MonitorCallback) MonitorCallback {
	f, err := glfwwin.SetMonitorCallback(cbfun)
	if err != nil {
		panic(err)
	}
	return f
}

func SwapInterval(interval int) {
	if err := glfwwin.SwapInterval(interval); err != nil {
		panic(err)
	}
}

func Terminate() {
	if err := glfwwin.Terminate(); err != nil {
		panic(err)
	}
}

func WaitEvents() {
	if err := glfwwin.WaitEvents(); err != nil {
		panic(err)
	}
}

func WindowHint(target Hint, hint int) {
	if err := glfwwin.WindowHint(glfwwin.Hint(target), hint); err != nil {
		panic(err)
	}
}
