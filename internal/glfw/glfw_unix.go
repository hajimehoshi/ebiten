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

//go:build darwin || freebsd || linux || netbsd || openbsd

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

func CreateStandardCursor(shape StandardCursor) (*Cursor, error) {
	c := cglfw.CreateStandardCursor(cglfw.StandardCursor(shape))
	return &Cursor{c: c}, nil
}

type Monitor struct {
	m *cglfw.Monitor
}

func (m *Monitor) GetContentScale() (float32, float32, error) {
	x, y := m.m.GetContentScale()
	return x, y, nil
}

func (m *Monitor) GetPos() (x, y int, err error) {
	x, y = m.m.GetPos()
	return
}

func (m *Monitor) GetVideoMode() (*VidMode, error) {
	v := m.m.GetVideoMode()
	if v == nil {
		return nil, nil
	}
	return &VidMode{
		Width:       v.Width,
		Height:      v.Height,
		RedBits:     v.RedBits,
		GreenBits:   v.GreenBits,
		BlueBits:    v.BlueBits,
		RefreshRate: v.RefreshRate,
	}, nil
}

func (m *Monitor) GetName() (string, error) {
	return m.m.GetName(), nil
}

type Window struct {
	w *cglfw.Window

	prevSizeCallback SizeCallback
}

func (w *Window) Destroy() {
	w.w.Destroy()
	theWindows.remove(w.w)
}

func (w *Window) Focus() error {
	w.w.Focus()
	return nil
}

func (w *Window) GetAttrib(attrib Hint) (int, error) {
	return w.w.GetAttrib(cglfw.Hint(attrib)), nil
}

func (w *Window) GetCursorPos() (x, y float64, err error) {
	x, y = w.w.GetCursorPos()
	return
}

func (w *Window) GetInputMode(mode InputMode) (int, error) {
	return w.w.GetInputMode(cglfw.InputMode(mode)), nil
}

func (w *Window) GetKey(key Key) (Action, error) {
	return Action(w.w.GetKey(cglfw.Key(key))), nil
}

func (w *Window) GetMonitor() (*Monitor, error) {
	m := w.w.GetMonitor()
	if m == nil {
		return nil, nil
	}
	return &Monitor{m}, nil
}

func (w *Window) GetMouseButton(button MouseButton) (Action, error) {
	return Action(w.w.GetMouseButton(cglfw.MouseButton(button))), nil
}

func (w *Window) GetPos() (x, y int, err error) {
	x, y = w.w.GetPos()
	return
}

func (w *Window) GetSize() (width, height int, err error) {
	width, height = w.w.GetSize()
	return
}

func (w *Window) Hide() {
	w.w.Hide()
}

func (w *Window) Iconify() error {
	w.w.Iconify()
	return nil
}

func (w *Window) MakeContextCurrent() error {
	w.w.MakeContextCurrent()
	return nil
}

func (w *Window) Maximize() error {
	w.w.Maximize()
	return nil
}

func (w *Window) Restore() error {
	w.w.Restore()
	return nil
}

func (w *Window) SetAttrib(attrib Hint, value int) error {
	w.w.SetAttrib(cglfw.Hint(attrib), value)
	return nil
}

func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (previous CharModsCallback, err error) {
	w.w.SetCharModsCallback(cbfun)
	return ToCharModsCallback(nil), nil // TODO
}

func (w *Window) SetCloseCallback(cbfun CloseCallback) (previous CloseCallback, err error) {
	w.w.SetCloseCallback(cbfun)
	return ToCloseCallback(nil), nil // TODO
}

func (w *Window) SetCursor(cursor *Cursor) error {
	var c *cglfw.Cursor
	if cursor != nil {
		c = cursor.c
	}
	w.w.SetCursor(c)
	return nil
}

func (w *Window) SetCursorPos(xpos, ypos float64) error {
	w.w.SetCursorPos(xpos, ypos)
	return nil
}

func (w *Window) SetDropCallback(cbfun DropCallback) (previous DropCallback, err error) {
	w.w.SetDropCallback(cbfun)
	return ToDropCallback(nil), nil // TODO
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback, err error) {
	w.w.SetFramebufferSizeCallback(cbfun)
	return ToFramebufferSizeCallback(nil), nil // TODO
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback, err error) {
	w.w.SetScrollCallback(cbfun)
	return ToScrollCallback(nil), nil // TODO
}

func (w *Window) SetShouldClose(value bool) error {
	w.w.SetShouldClose(value)
	return nil
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous SizeCallback, err error) {
	w.w.SetSizeCallback(cbfun)
	prev := w.prevSizeCallback
	w.prevSizeCallback = cbfun
	return prev, nil
}

func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) error {
	w.w.SetSizeLimits(minw, minh, maxw, maxh)
	return nil
}

func (w *Window) SetIcon(images []image.Image) error {
	w.w.SetIcon(images)
	return nil
}

func (w *Window) SetInputMode(mode InputMode, value int) error {
	w.w.SetInputMode(cglfw.InputMode(mode), value)
	return nil
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	var m *cglfw.Monitor
	if monitor != nil {
		m = monitor.m
	}
	w.w.SetMonitor(m, xpos, ypos, width, height, refreshRate)
	return nil
}

func (w *Window) SetPos(xpos, ypos int) error {
	w.w.SetPos(xpos, ypos)
	return nil
}

func (w *Window) SetSize(width, height int) error {
	w.w.SetSize(width, height)
	return nil
}

func (w *Window) SetTitle(title string) error {
	w.w.SetTitle(title)
	return nil
}

func (w *Window) ShouldClose() (bool, error) {
	return w.w.ShouldClose(), nil
}

func (w *Window) Show() error {
	w.w.Show()
	return nil
}

func (w *Window) SwapBuffers() error {
	w.w.SwapBuffers()
	return nil
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

func GetKeyName(key Key, scancode int) (string, error) {
	return cglfw.GetKeyName(cglfw.Key(key), scancode), nil
}

func GetMonitors() ([]*Monitor, error) {
	var ms []*Monitor
	for _, m := range cglfw.GetMonitors() {
		if m != nil {
			ms = append(ms, &Monitor{m})
		} else {
			ms = append(ms, nil)
		}
	}
	return ms, nil
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

func PollEvents() error {
	cglfw.PollEvents()
	return nil
}

func PostEmptyEvent() error {
	cglfw.PostEmptyEvent()
	return nil
}

func SetMonitorCallback(cbfun MonitorCallback) (MonitorCallback, error) {
	cglfw.SetMonitorCallback(cbfun)
	return ToMonitorCallback(nil), nil
}

func SwapInterval(interval int) error {
	cglfw.SwapInterval(interval)
	return nil
}

func Terminate() error {
	cglfw.Terminate()
	return nil
}

func WaitEvents() error {
	cglfw.WaitEvents()
	return nil
}

func WaitEventsTimeout(timeout float64) {
	cglfw.WaitEventsTimeout(timeout)
}

func WindowHint(target Hint, hint int) error {
	cglfw.WindowHint(cglfw.Hint(target), hint)
	return nil
}
