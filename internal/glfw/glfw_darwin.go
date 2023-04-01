// Copyright 2022 The Ebiten Authors
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
	"math"
	"math/bits"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
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
	ww := &Window{w: win}
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

type Cursor struct {
	c uintptr
}

func CreateStandardCursor(shape StandardCursor) *Cursor {
	c := libglfw.call("glfwCreateStandardCursor", uintptr(shape))
	panicError()
	return &Cursor{c: c}
}

type Monitor struct {
	m uintptr
}

func (m *Monitor) GetContentScale() (float32, float32, error) {
	var sx, sy float32
	libglfw.call("glfwGetMonitorContentScale", m.m, uintptr(unsafe.Pointer(&sx)), uintptr(unsafe.Pointer(&sy)))
	return sx, sy, fetchError()
}

func (m *Monitor) GetPos() (int, int) {
	var x, y int32
	libglfw.call("glfwGetMonitorPos", m.m, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	panicError()
	return int(x), int(y)
}

func (m *Monitor) GetVideoMode() *VidMode {
	v := libglfw.call("glfwGetVideoMode", m.m)
	panicError()
	var vals []int32
	h := (*reflect.SliceHeader)(unsafe.Pointer(&vals))
	h.Data = v
	h.Len = 6
	h.Cap = 6
	return &VidMode{
		Width:       int(vals[0]),
		Height:      int(vals[1]),
		RedBits:     int(vals[2]),
		GreenBits:   int(vals[3]),
		BlueBits:    int(vals[4]),
		RefreshRate: int(vals[5]),
	}
}

type Window struct {
	w uintptr

	prevSizeCallback SizeCallback
}

func (w *Window) Destroy() {
	libglfw.call("glfwDestroyWindow", w.w)
	panicError()
	theGLFWWindows.remove(w.w)
}

func (w *Window) GetAttrib(attrib Hint) int {
	r := libglfw.call("glfwGetWindowAttrib", w.w, uintptr(attrib))
	panicError()
	return int(r)
}

func (w *Window) SetAttrib(attrib Hint, value int) {
	libglfw.call("glfwSetWindowAttrib", w.w, uintptr(attrib), uintptr(value))
	panicError()
}

func (w *Window) GetCursorPos() (x, y float64) {
	libglfw.call("glfwGetCursorPos", w.w, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	panicError()
	return
}

func (w *Window) GetInputMode(mode InputMode) int {
	r := libglfw.call("glfwGetInputMode", w.w, uintptr(mode))
	panicError()
	return int(r)
}

func (w *Window) GetKey(key Key) Action {
	r := libglfw.call("glfwGetKey", w.w, uintptr(key))
	panicError()
	return Action(r)
}

func (w *Window) GetMonitor() *Monitor {
	m := libglfw.call("glfwGetWindowMonitor", w.w)
	panicError()
	if m == 0 {
		return nil
	}
	return &Monitor{m}
}

func (w *Window) GetMouseButton(button MouseButton) Action {
	r := libglfw.call("glfwGetMouseButton", w.w, uintptr(button))
	panicError()
	return Action(r)
}

func (w *Window) GetPos() (int, int) {
	var x, y int32
	libglfw.call("glfwGetWindowPos", w.w, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)))
	panicError()
	return int(x), int(y)
}

func (w *Window) GetSize() (int, int) {
	var width, height int32
	libglfw.call("glfwGetWindowSize", w.w, uintptr(unsafe.Pointer(&width)), uintptr(unsafe.Pointer(&height)))
	panicError()
	return int(width), int(height)
}

func (w *Window) Focus() {
	libglfw.call("glfwFocusWindow", w.w)
	panicError()
}

func (w *Window) Hide() {
	libglfw.call("glfwHideWindow", w.w)
	panicError()
}

func (w *Window) Iconify() {
	libglfw.call("glfwIconifyWindow", w.w)
	panicError()
}

func (w *Window) MakeContextCurrent() {
	libglfw.call("glfwMakeContextCurrent", w.w)
	panicError()
}

func (w *Window) Maximize() {
	libglfw.call("glfwMaximizeWindow", w.w)
	panicError()
}

func (w *Window) Restore() {
	libglfw.call("glfwRestoreWindow", w.w)
	panicError()
}

func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (previous CharModsCallback) {
	libglfw.call("glfwSetCharModsCallback", w.w, purego.NewCallback(cbfun))
	panicError()
	return ToCharModsCallback(nil) // TODO
}

func (w *Window) SetCloseCallback(cbfun CloseCallback) (previous CloseCallback) {
	libglfw.call("glfwSetWindowCloseCallback", w.w, purego.NewCallback(cbfun))
	panicError()
	return ToCloseCallback(nil) // TODO
}

func (w *Window) SetDropCallback(cbfun DropCallback) (previous DropCallback) {
	libglfw.call("glfwSetDropCallback", w.w, purego.NewCallback(cbfun))
	panicError()
	return DropCallback(nil) // TODO
}

func (w *Window) SetCursor(cursor *Cursor) {
	var c uintptr
	if cursor != nil {
		c = cursor.c
	}
	libglfw.call("glfwSetCursor", w.w, c)
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback) {
	libglfw.call("glfwSetFramebufferSizeCallback", w.w, purego.NewCallback(cbfun))
	panicError()
	return ToFramebufferSizeCallback(nil) // TODO
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback) {
	libglfw.call("glfwSetScrollCallback", w.w, purego.NewCallback(cbfun))
	panicError()
	return ToScrollCallback(nil) // TODO
}

func (w *Window) SetShouldClose(value bool) {
	var v uintptr = False
	if value {
		v = True
	}
	libglfw.call("glfwSetWindowShouldClose", w.w, v)
	panicError()
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous SizeCallback) {
	libglfw.call("glfwSetWindowSizeCallback", w.w, purego.NewCallback(cbfun))
	panicError()
	prev := w.prevSizeCallback
	w.prevSizeCallback = cbfun
	return prev
}

func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) {
	libglfw.call("glfwSetWindowSizeLimits", w.w, uintptr(minw), uintptr(minh), uintptr(maxw), uintptr(maxh))
	panicError()
}

func (w *Window) SetAspectRatio(numer, denom int) {
	libglfw.call("glfwSetWindowAspectRatio", w.w, uintptr(numer), uintptr(denom))
	panicError()
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

	libglfw.call("glfwSetWindowIcon", w.w, uintptr(len(gimgs)), uintptr(unsafe.Pointer(&gimgs[0])))
	panicError()
}

func (w *Window) SetInputMode(mode InputMode, value int) {
	libglfw.call("glfwSetInputMode", w.w, uintptr(mode), uintptr(value))
	panicError()
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	var m uintptr
	if monitor != nil {
		m = monitor.m
	}
	libglfw.call("glfwSetWindowMonitor", w.w, m, uintptr(xpos), uintptr(ypos), uintptr(width), uintptr(height), uintptr(refreshRate))
	panicError()
}

func (w *Window) SetPos(xpos, ypos int) {
	libglfw.call("glfwSetWindowPos", w.w, uintptr(xpos), uintptr(ypos))
	panicError()
}

func (w *Window) SetSize(width, height int) {
	libglfw.call("glfwSetWindowSize", w.w, uintptr(width), uintptr(height))
	panicError()
}

func (w *Window) SetTitle(title string) {
	s := []byte(title)
	s = append(s, 0)
	defer runtime.KeepAlive(s)
	libglfw.call("glfwSetWindowTitle", w.w, uintptr(unsafe.Pointer(&s[0])))
	panicError()
}

func (w *Window) ShouldClose() bool {
	r := libglfw.call("glfwWindowShouldClose", w.w)
	panicError()
	return byte(r) == True
}

func (w *Window) Show() {
	libglfw.call("glfwShowWindow", w.w)
	panicError()
}

func (w *Window) SwapBuffers() {
	libglfw.call("glfwSwapBuffers", w.w)
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

	w := libglfw.call("glfwCreateWindow", uintptr(width), uintptr(height), uintptr(unsafe.Pointer(&s[0])), gm, gw)
	if w == 0 {
		return nil, acceptError(APIUnavailable, VersionUnavailable)
	}
	return theGLFWWindows.add(w), nil
}

func GetKeyName(key Key, scancode int) string {
	n := libglfw.call("glfwGetKeyName", uintptr(key), uintptr(scancode))
	panicError()
	return goString(n)
}

// goString copies a char* to a Go string.
func goString(c uintptr) string {
	// We take the address and then dereference it to trick go vet from creating a possible misuse of unsafe.Pointer
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&c))
	if ptr == nil {
		return ""
	}
	var length int
	for {
		if *(*byte)(unsafe.Add(ptr, uintptr(length))) == '\x00' {
			break
		}
		length++
	}
	return string(unsafe.Slice((*byte)(ptr), length))
}

func GetMonitors() []*Monitor {
	var l int32
	ptr := libglfw.call("glfwGetMonitors", uintptr(unsafe.Pointer(&l)))
	panicError()
	ms := make([]*Monitor, l)
	for i := int32(0); i < l; i++ {
		m := **(**unsafe.Pointer)(unsafe.Pointer(&ptr))
		if m != nil {
			ms[i] = &Monitor{uintptr(m)}
		}
		ptr += bits.UintSize / 8
	}
	return ms
}

func GetPrimaryMonitor() *Monitor {
	m := libglfw.call("glfwGetPrimaryMonitor")
	panicError()
	if m == 0 {
		return nil
	}
	return &Monitor{m}
}

func Init() error {
	libglfw.call("glfwInit")

	err := acceptError(APIUnavailable, InvalidValue)
	if e, ok := err.(*glfwError); ok && e.code == InvalidValue {
		return nil
	}
	return err
}

func PollEvents() {
	libglfw.call("glfwPollEvents")
}

func PostEmptyEvent() {
	libglfw.call("glfwPostEmptyEvent")
	panicError()
}

func SetMonitorCallback(cbfun MonitorCallback) {
	libglfw.call("glfwSetMonitorCallback", purego.NewCallback(cbfun))
	panicError()
}

func SwapInterval(interval int) {
	libglfw.call("glfwSwapInterval", uintptr(interval))
	panicError()
}

func Terminate() {
	flushErrors()
	libglfw.call("glfwTerminate")
	if err := libglfw.unload(); err != nil {
		panic(err)
	}
}

func WaitEvents() {
	libglfw.call("glfwWaitEvents")
	panicError()
}

func WindowHint(target Hint, hint int) {
	libglfw.call("glfwWindowHint", uintptr(target), uintptr(hint))
	panicError()
}

func WaitEventsTimeout(timeout float64) {
	libglfw.call("glfwWaitEventsTimeout", uintptr(math.Float64bits(timeout)))
	panicError()
}
