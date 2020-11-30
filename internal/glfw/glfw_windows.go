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
	"image"
	"image/draw"
	"math/bits"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type glfwImage struct {
	width  int32
	height int32
	pixels uintptr
}

type glfwWindows map[uintptr]*Window

var (
	theGLFWWindows = glfwWindows{}
	glfwWindowsM   sync.Mutex
)

func (w glfwWindows) add(win uintptr) *Window {
	if win == 0 {
		return nil
	}
	ww := &Window{win}
	glfwWindowsM.Lock()
	w[win] = ww
	glfwWindowsM.Unlock()
	return ww
}

func (w glfwWindows) remove(win uintptr) {
	glfwWindowsM.Lock()
	delete(w, win)
	glfwWindowsM.Unlock()
}

func (w glfwWindows) get(win uintptr) *Window {
	if win == 0 {
		return nil
	}
	glfwWindowsM.Lock()
	ww := w[win]
	glfwWindowsM.Unlock()
	return ww
}

type Monitor struct {
	m uintptr
}

func (m *Monitor) GetPos() (int, int) {
	var x, y int32
	glfwDLL.call("glfwGetMonitorPos", m.m, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	panicError()
	return int(x), int(y)
}

func (m *Monitor) GetVideoMode() *VidMode {
	v := glfwDLL.call("glfwGetVideoMode", m.m)
	panicError()
	return &VidMode{
		Width:       int(*(*int32)(unsafe.Pointer(v))),
		Height:      int(*(*int32)(unsafe.Pointer(v + 4))),
		RedBits:     int(*(*int32)(unsafe.Pointer(v + 8))),
		GreenBits:   int(*(*int32)(unsafe.Pointer(v + 12))),
		BlueBits:    int(*(*int32)(unsafe.Pointer(v + 16))),
		RefreshRate: int(*(*int32)(unsafe.Pointer(v + 20))),
	}
}

type Window struct {
	w uintptr
}

func (w *Window) Destroy() {
	glfwDLL.call("glfwDestroyWindow", w.w)
	panicError()
	theGLFWWindows.remove(w.w)
}

func (w *Window) GetAttrib(attrib Hint) int {
	r := glfwDLL.call("glfwGetWindowAttrib", w.w, uintptr(attrib))
	panicError()
	return int(r)
}

func (w *Window) SetAttrib(attrib Hint, value int) {
	glfwDLL.call("glfwSetWindowAttrib", w.w, uintptr(attrib), uintptr(value))
	panicError()
}

func (w *Window) GetCursorPos() (x, y float64) {
	glfwDLL.call("glfwGetCursorPos", w.w, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	panicError()
	return
}

func (w *Window) GetInputMode(mode InputMode) int {
	r := glfwDLL.call("glfwGetInputMode", w.w, uintptr(mode))
	panicError()
	return int(r)
}

func (w *Window) GetKey(key Key) Action {
	r := glfwDLL.call("glfwGetKey", w.w, uintptr(key))
	panicError()
	return Action(r)
}

func (w *Window) GetMonitor() *Monitor {
	m := glfwDLL.call("glfwGetWindowMonitor", w.w)
	panicError()
	if m == 0 {
		return nil
	}
	return &Monitor{m}
}

func (w *Window) GetMouseButton(button MouseButton) Action {
	r := glfwDLL.call("glfwGetMouseButton", w.w, uintptr(button))
	panicError()
	return Action(r)
}

func (w *Window) GetPos() (int, int) {
	var x, y int32
	glfwDLL.call("glfwGetWindowPos", w.w, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	panicError()
	return int(x), int(y)
}

func (w *Window) GetSize() (int, int) {
	var width, height int32
	glfwDLL.call("glfwGetWindowSize", w.w, uintptr(unsafe.Pointer(&width)), uintptr(unsafe.Pointer(&height)))
	panicError()
	return int(width), int(height)
}

func (w *Window) Iconify() {
	glfwDLL.call("glfwIconifyWindow", w.w)
	panicError()
}

func (w *Window) MakeContextCurrent() {
	glfwDLL.call("glfwMakeContextCurrent", w.w)
	panicError()
}

func (w *Window) Maximize() {
	glfwDLL.call("glfwMaximizeWindow", w.w)
	panicError()
}

func (w *Window) Restore() {
	glfwDLL.call("glfwRestoreWindow", w.w)
	panicError()
}

func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (previous CharModsCallback) {
	var gcb uintptr
	if cbfun != nil {
		gcb = windows.NewCallbackCDecl(func(window uintptr, char rune, mods ModifierKey) uintptr {
			cbfun(theGLFWWindows.get(window), char, mods)
			return 0
		})
	}
	glfwDLL.call("glfwSetCharModsCallback", w.w, gcb)
	panicError()
	return nil // TODO
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback) {
	var gcb uintptr
	if cbfun != nil {
		gcb = windows.NewCallbackCDecl(func(window uintptr, width int, height int) uintptr {
			cbfun(theGLFWWindows.get(window), width, height)
			return 0
		})
	}
	glfwDLL.call("glfwSetFramebufferSizeCallback", w.w, gcb)
	panicError()
	return nil // TODO
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback) {
	var gcb uintptr
	if cbfun != nil {
		gcb = windows.NewCallbackCDecl(func(window uintptr, xoff *float64, yoff *float64) uintptr {
			// xoff and yoff were originally float64, but there is no good way to pass them on 32bit
			// machines via NewCallback. We've fixed GLFW side to use pointer values.
			cbfun(theGLFWWindows.get(window), *xoff, *yoff)
			return 0
		})
	}
	glfwDLL.call("glfwSetScrollCallback", w.w, gcb)
	panicError()
	return nil // TODO
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous FramebufferSizeCallback) {
	var gcb uintptr
	if cbfun != nil {
		gcb = windows.NewCallbackCDecl(func(window uintptr, width int, height int) uintptr {
			cbfun(theGLFWWindows.get(window), width, height)
			return 0
		})
	}
	glfwDLL.call("glfwSetWindowSizeCallback", w.w, gcb)
	panicError()
	return nil // TODO
}

func (w *Window) SetIcon(images []image.Image) {
	gimgs := make([]glfwImage, len(images))
	defer runtime.KeepAlive(gimgs)

	for i, img := range images {
		b := img.Bounds()
		m := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
		gimgs[i].width = int32(b.Dx())
		gimgs[i].height = int32(b.Dy())
		gimgs[i].pixels = uintptr(unsafe.Pointer(&m.Pix[0]))
	}

	glfwDLL.call("glfwSetWindowIcon", w.w, uintptr(len(gimgs)), uintptr(unsafe.Pointer(&gimgs[0])))
	panicError()
}

func (w *Window) SetInputMode(mode InputMode, value int) {
	glfwDLL.call("glfwSetInputMode", w.w, uintptr(mode), uintptr(value))
	panicError()
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	var m uintptr
	if monitor != nil {
		m = monitor.m
	}
	glfwDLL.call("glfwSetWindowMonitor", w.w, m, uintptr(xpos), uintptr(ypos), uintptr(width), uintptr(height), uintptr(refreshRate))
	panicError()
}

func (w *Window) SetPos(xpos, ypos int) {
	glfwDLL.call("glfwSetWindowPos", w.w, uintptr(xpos), uintptr(ypos))
	panicError()
}

func (w *Window) SetSize(width, height int) {
	glfwDLL.call("glfwSetWindowSize", w.w, uintptr(width), uintptr(height))
	panicError()
}

func (w *Window) SetTitle(title string) {
	s := []byte(title)
	s = append(s, 0)
	defer runtime.KeepAlive(s)
	glfwDLL.call("glfwSetWindowTitle", w.w, uintptr(unsafe.Pointer(&s[0])))
	panicError()
}

func (w *Window) ShouldClose() bool {
	r := glfwDLL.call("glfwWindowShouldClose", w.w)
	panicError()
	return r == True
}

func (w *Window) Show() {
	glfwDLL.call("glfwShowWindow", w.w)
	panicError()
}

func (w *Window) SwapBuffers() {
	glfwDLL.call("glfwSwapBuffers", w.w)
	panicError()
}

func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (*Window, error) {
	s := []byte(title)
	s = append(s, 0)
	defer runtime.KeepAlive(s)

	var gm uintptr
	if monitor != nil {
		gm = monitor.m
	}
	var gw uintptr
	if share != nil {
		gw = share.w
	}

	w := glfwDLL.call("glfwCreateWindow", uintptr(width), uintptr(height), uintptr(unsafe.Pointer(&s[0])), gm, gw)
	if w == 0 {
		return nil, acceptError(APIUnavailable, VersionUnavailable)
	}
	return theGLFWWindows.add(w), nil
}

func (j Joystick) GetGUID() string {
	ptr := glfwDLL.call("glfwGetJoystickGUID", uintptr(j))
	panicError()
	var backed [256]byte
	as := backed[:0]
	for i := int32(0); ; i++ {
		b := *(*byte)(unsafe.Pointer(ptr))
		ptr += unsafe.Sizeof(byte(0))
		if b == 0 {
			break
		}
		as = append(as, b)
	}
	r := string(as)
	return r
}

func (j Joystick) GetName() string {
	ptr := glfwDLL.call("glfwGetJoystickName", uintptr(j))
	panicError()
	var backed [256]byte
	as := backed[:0]
	for i := int32(0); ; i++ {
		b := *(*byte)(unsafe.Pointer(ptr))
		ptr += unsafe.Sizeof(byte(0))
		if b == 0 {
			break
		}
		as = append(as, b)
	}
	r := string(as)
	return r
}

func (j Joystick) GetAxes() []float32 {
	var l int32
	ptr := glfwDLL.call("glfwGetJoystickAxes", uintptr(j), uintptr(unsafe.Pointer(&l)))
	panicError()
	as := make([]float32, l)
	for i := int32(0); i < l; i++ {
		as[i] = *(*float32)(unsafe.Pointer(ptr))
		ptr += unsafe.Sizeof(float32(0))
	}
	return as
}

func (j Joystick) GetButtons() []byte {
	var l int32
	ptr := glfwDLL.call("glfwGetJoystickButtons", uintptr(j), uintptr(unsafe.Pointer(&l)))
	panicError()
	bs := make([]byte, l)
	for i := int32(0); i < l; i++ {
		bs[i] = *(*byte)(unsafe.Pointer(ptr))
		ptr++
	}
	return bs
}

func GetMonitors() []*Monitor {
	var l int32
	ptr := glfwDLL.call("glfwGetMonitors", uintptr(unsafe.Pointer(&l)))
	panicError()
	ms := make([]*Monitor, l)
	for i := int32(0); i < l; i++ {
		m := *(*unsafe.Pointer)(unsafe.Pointer(ptr))
		if m != nil {
			ms[i] = &Monitor{uintptr(m)}
		}
		ptr += bits.UintSize / 8
	}
	return ms
}

func GetPrimaryMonitor() *Monitor {
	m := glfwDLL.call("glfwGetPrimaryMonitor")
	panicError()
	if m == 0 {
		return nil
	}
	return &Monitor{m}
}

func Init() error {
	glfwDLL.call("glfwInit")
	return acceptError(APIUnavailable)
}

func (j Joystick) Present() bool {
	r := glfwDLL.call("glfwJoystickPresent", uintptr(j))
	panicError()
	return r == True
}

func PollEvents() {
	glfwDLL.call("glfwPollEvents")
	panicError()
}

func SetMonitorCallback(cbfun func(monitor *Monitor, event PeripheralEvent)) {
	var gcb uintptr
	if cbfun != nil {
		gcb = windows.NewCallbackCDecl(func(monitor uintptr, event PeripheralEvent) uintptr {
			var m *Monitor
			if monitor != 0 {
				m = &Monitor{monitor}
			}
			cbfun(m, event)
			return 0
		})
	}
	glfwDLL.call("glfwSetMonitorCallback", gcb)
	panicError()
}

func SwapInterval(interval int) {
	glfwDLL.call("glfwSwapInterval", uintptr(interval))
	panicError()
}

func Terminate() {
	flushErrors()
	glfwDLL.call("glfwTerminate")
	if err := glfwDLL.unload(); err != nil {
		panic(err)
	}
}

func WindowHint(target Hint, hint int) {
	glfwDLL.call("glfwWindowHint", uintptr(target), uintptr(hint))
	panicError()
}
