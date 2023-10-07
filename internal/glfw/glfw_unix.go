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
	c, err := cglfw.CreateStandardCursor(cglfw.StandardCursor(shape))
	if err != nil {
		return nil, err
	}
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
	return m.m.GetPos()
}

func (m *Monitor) GetVideoMode() (*VidMode, error) {
	v, err := m.m.GetVideoMode()
	if err != nil {
		return nil, err
	}
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
	return m.m.GetName()
}

type Window struct {
	w *cglfw.Window
}

func (w *Window) Destroy() error {
	if err := w.w.Destroy(); err != nil {
		return err
	}
	theWindows.remove(w.w)
	return nil
}

func (w *Window) Focus() error {
	w.w.Focus()
	return nil
}

func (w *Window) GetAttrib(attrib Hint) (int, error) {
	return w.w.GetAttrib(cglfw.Hint(attrib))
}

func (w *Window) GetCursorPos() (x, y float64, err error) {
	return w.w.GetCursorPos()
}

func (w *Window) GetInputMode(mode InputMode) (int, error) {
	return w.w.GetInputMode(cglfw.InputMode(mode))
}

func (w *Window) GetKey(key Key) (Action, error) {
	a, err := w.w.GetKey(cglfw.Key(key))
	if err != nil {
		return 0, err
	}
	return Action(a), nil
}

func (w *Window) GetMonitor() (*Monitor, error) {
	m, err := w.w.GetMonitor()
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return &Monitor{m}, nil
}

func (w *Window) GetMouseButton(button MouseButton) (Action, error) {
	a, err := w.w.GetMouseButton(cglfw.MouseButton(button))
	if err != nil {
		return 0, err
	}
	return Action(a), nil
}

func (w *Window) GetPos() (x, y int, err error) {
	return w.w.GetPos()
}

func (w *Window) GetSize() (width, height int, err error) {
	return w.w.GetSize()
}

func (w *Window) Hide() error {
	return w.w.Hide()
}

func (w *Window) Iconify() error {
	w.w.Iconify()
	return nil
}

func (w *Window) MakeContextCurrent() error {
	return w.w.MakeContextCurrent()
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
	return w.w.SetCharModsCallback(cbfun)
}

func (w *Window) SetCloseCallback(cbfun CloseCallback) (previous CloseCallback, err error) {
	return w.w.SetCloseCallback(cbfun)
}

func (w *Window) SetCursor(cursor *Cursor) error {
	var c *cglfw.Cursor
	if cursor != nil {
		c = cursor.c
	}
	return w.w.SetCursor(c)
}

func (w *Window) SetCursorPos(xpos, ypos float64) error {
	return w.w.SetCursorPos(xpos, ypos)
}

func (w *Window) SetDropCallback(cbfun DropCallback) (previous DropCallback, err error) {
	return w.w.SetDropCallback(cbfun)
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback, err error) {
	return w.w.SetFramebufferSizeCallback(cbfun)
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback, err error) {
	return w.w.SetScrollCallback(cbfun)
}

func (w *Window) SetShouldClose(value bool) error {
	return w.w.SetShouldClose(value)
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous SizeCallback, err error) {
	return w.w.SetSizeCallback(cbfun)
}

func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) error {
	return w.w.SetSizeLimits(minw, minh, maxw, maxh)
}

func (w *Window) SetIcon(images []image.Image) error {
	return w.w.SetIcon(images)
}

func (w *Window) SetInputMode(mode InputMode, value int) error {
	return w.w.SetInputMode(cglfw.InputMode(mode), value)
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	var m *cglfw.Monitor
	if monitor != nil {
		m = monitor.m
	}
	if err := w.w.SetMonitor(m, xpos, ypos, width, height, refreshRate); err != nil {
		return err
	}
	return nil
}

func (w *Window) SetPos(xpos, ypos int) error {
	return w.w.SetPos(xpos, ypos)
}

func (w *Window) SetSize(width, height int) error {
	return w.w.SetSize(width, height)
}

func (w *Window) SetTitle(title string) error {
	return w.w.SetTitle(title)
}

func (w *Window) ShouldClose() (bool, error) {
	return w.w.ShouldClose()
}

func (w *Window) Show() error {
	return w.w.Show()
}

func (w *Window) SwapBuffers() error {
	return w.w.SwapBuffers()
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
	return cglfw.GetKeyName(cglfw.Key(key), scancode)
}

func GetMonitors() ([]*Monitor, error) {
	monitors, err := cglfw.GetMonitors()
	if err != nil {
		return nil, err
	}
	var ms []*Monitor
	for _, m := range monitors {
		if m != nil {
			ms = append(ms, &Monitor{m})
		} else {
			ms = append(ms, nil)
		}
	}
	return ms, nil
}

func GetPrimaryMonitor() (*Monitor, error) {
	m, err := cglfw.GetPrimaryMonitor()
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return &Monitor{m}, nil
}

func Init() error {
	return cglfw.Init()
}

func PollEvents() error {
	return cglfw.PollEvents()
}

func PostEmptyEvent() error {
	return cglfw.PostEmptyEvent()
}

func SetMonitorCallback(cbfun MonitorCallback) (MonitorCallback, error) {
	cglfw.SetMonitorCallback(cbfun)
	return ToMonitorCallback(nil), nil
}

func SwapInterval(interval int) error {
	return cglfw.SwapInterval(interval)
}

func Terminate() error {
	return cglfw.Terminate()
}

func WaitEvents() error {
	return cglfw.WaitEvents()
}

func WaitEventsTimeout(timeout float64) error {
	return cglfw.WaitEventsTimeout(timeout)
}

func WindowHint(target Hint, hint int) error {
	return cglfw.WindowHint(cglfw.Hint(target), hint)
}
