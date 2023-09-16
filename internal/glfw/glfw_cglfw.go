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

package glfw

import (
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/cglfw"
)

type windows map[*cglfw.Window]*Window

var (
	theWindows = windows{}
	windowsM   sync.Mutex
)

func (w windows) add(win *cglfw.Window) *Window {
	if win == nil {
		return nil
	}
	ww := &Window{w: win}
	windowsM.Lock()
	w[win] = ww
	windowsM.Unlock()
	return ww
}

func (w windows) remove(win *cglfw.Window) {
	windowsM.Lock()
	delete(w, win)
	windowsM.Unlock()
}

func (w windows) get(win *cglfw.Window) *Window {
	if win == nil {
		return nil
	}
	windowsM.Lock()
	ww := w[win]
	windowsM.Unlock()
	return ww
}

type Cursor struct {
	c *cglfw.Cursor
}

func CreateStandardCursor(shape StandardCursor) *Cursor {
	c := cglfw.CreateStandardCursor(cglfw.StandardCursor(shape))
	return &Cursor{c: c}
}

type Monitor struct {
	m *cglfw.Monitor
}

func (m *Monitor) GetContentScale() (float32, float32, error) {
	x, y := m.m.GetContentScale()
	return x, y, nil
}

func (m *Monitor) GetPos() (x, y int) {
	return m.m.GetPos()
}

func (m *Monitor) GetVideoMode() *VidMode {
	v := m.m.GetVideoMode()
	if v == nil {
		return nil
	}
	return &VidMode{
		Width:       v.Width,
		Height:      v.Height,
		RedBits:     v.RedBits,
		GreenBits:   v.GreenBits,
		BlueBits:    v.BlueBits,
		RefreshRate: v.RefreshRate,
	}
}

func (m *Monitor) GetName() string {
	return m.m.GetName()
}

type Window struct {
	w *cglfw.Window

	prevSizeCallback SizeCallback
}

func (w *Window) Destroy() {
	w.w.Destroy()
	theWindows.remove(w.w)
}

func (w *Window) Focus() {
	w.w.Focus()
}

func (w *Window) GetAttrib(attrib Hint) int {
	return w.w.GetAttrib(cglfw.Hint(attrib))
}

func (w *Window) GetCursorPos() (x, y float64) {
	return w.w.GetCursorPos()
}

func (w *Window) GetInputMode(mode InputMode) int {
	return w.w.GetInputMode(cglfw.InputMode(mode))
}

func (w *Window) GetKey(key Key) Action {
	return Action(w.w.GetKey(cglfw.Key(key)))
}

func (w *Window) GetMonitor() *Monitor {
	m := w.w.GetMonitor()
	if m == nil {
		return nil
	}
	return &Monitor{m}
}

func (w *Window) GetMouseButton(button MouseButton) Action {
	return Action(w.w.GetMouseButton(cglfw.MouseButton(button)))
}

func (w *Window) GetPos() (x, y int) {
	return w.w.GetPos()
}

func (w *Window) GetSize() (width, height int) {
	return w.w.GetSize()
}

func (w *Window) Hide() {
	w.w.Hide()
}

func (w *Window) Iconify() {
	w.w.Iconify()
}

func (w *Window) MakeContextCurrent() {
	w.w.MakeContextCurrent()
}

func (w *Window) Maximize() {
	w.w.Maximize()
}

func (w *Window) Restore() {
	w.w.Restore()
}

func (w *Window) SetAttrib(attrib Hint, value int) {
	w.w.SetAttrib(cglfw.Hint(attrib), value)
}

func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (previous CharModsCallback) {
	w.w.SetCharModsCallback(cbfun)
	return ToCharModsCallback(nil) // TODO
}

func (w *Window) SetCloseCallback(cbfun CloseCallback) (previous CloseCallback) {
	w.w.SetCloseCallback(cbfun)
	return ToCloseCallback(nil) // TODO
}

func (w *Window) SetCursor(cursor *Cursor) {
	var c *cglfw.Cursor
	if cursor != nil {
		c = cursor.c
	}
	w.w.SetCursor(c)
}

func (w *Window) SetCursorPos(xpos, ypos float64) {
	w.w.SetCursorPos(xpos, ypos)
}

func (w *Window) SetDropCallback(cbfun DropCallback) (previous DropCallback) {
	w.w.SetDropCallback(cbfun)
	return ToDropCallback(nil) // TODO
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback) {
	w.w.SetFramebufferSizeCallback(cbfun)
	return ToFramebufferSizeCallback(nil) // TODO
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback) {
	w.w.SetScrollCallback(cbfun)
	return ToScrollCallback(nil) // TODO
}

func (w *Window) SetShouldClose(value bool) {
	w.w.SetShouldClose(value)
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous SizeCallback) {
	w.w.SetSizeCallback(cbfun)
	prev := w.prevSizeCallback
	w.prevSizeCallback = cbfun
	return prev
}

func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) {
	w.w.SetSizeLimits(minw, minh, maxw, maxh)
}

func (w *Window) SetAspectRatio(numer, denom int) {
	w.w.SetAspectRatio(numer, denom)
}

func (w *Window) SetIcon(images []image.Image) {
	w.w.SetIcon(images)
}

func (w *Window) SetInputMode(mode InputMode, value int) {
	w.w.SetInputMode(cglfw.InputMode(mode), value)
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	var m *cglfw.Monitor
	if monitor != nil {
		m = monitor.m
	}
	w.w.SetMonitor(m, xpos, ypos, width, height, refreshRate)
}

func (w *Window) SetPos(xpos, ypos int) {
	w.w.SetPos(xpos, ypos)
}

func (w *Window) SetSize(width, height int) {
	w.w.SetSize(width, height)
}

func (w *Window) SetTitle(title string) {
	w.w.SetTitle(title)
}

func (w *Window) ShouldClose() bool {
	return w.w.ShouldClose()
}

func (w *Window) Show() {
	w.w.Show()
}

func (w *Window) SwapBuffers() {
	w.w.SwapBuffers()
}

func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (*Window, error) {
	var gm *cglfw.Monitor
	if monitor != nil {
		gm = monitor.m
	}
	var gw *cglfw.Window
	if share != nil {
		gw = share.w
	}

	w, err := cglfw.CreateWindow(width, height, title, gm, gw)
	if err != nil {
		return nil, err
	}
	return theWindows.add(w), nil
}

func GetKeyName(key Key, scancode int) string {
	return cglfw.GetKeyName(cglfw.Key(key), scancode)
}

func GetMonitors() []*Monitor {
	ms := []*Monitor{}
	for _, m := range cglfw.GetMonitors() {
		if m != nil {
			ms = append(ms, &Monitor{m})
		} else {
			ms = append(ms, nil)
		}
	}
	return ms
}

func GetPrimaryMonitor() *Monitor {
	m := cglfw.GetPrimaryMonitor()
	if m == nil {
		return nil
	}
	return &Monitor{m}
}

func Init() error {
	return cglfw.Init()
}

func PollEvents() {
	cglfw.PollEvents()
}

func PostEmptyEvent() {
	cglfw.PostEmptyEvent()
}

func SetMonitorCallback(cbfun MonitorCallback) MonitorCallback {
	cglfw.SetMonitorCallback(cbfun)
	return ToMonitorCallback(nil)
}

func SwapInterval(interval int) {
	cglfw.SwapInterval(interval)
}

func Terminate() {
	cglfw.Terminate()
}

func WaitEvents() {
	cglfw.WaitEvents()
}

func WaitEventsTimeout(timeout float64) {
	cglfw.WaitEventsTimeout(timeout)
}

func WindowHint(target Hint, hint int) {
	cglfw.WindowHint(cglfw.Hint(target), hint)
}
