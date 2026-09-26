// Copyright 2026 The Ebitengine Authors
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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

// DesktopWindowForTest wraps a desktopWindow with a fake backend, so tests can exercise its
// window-state logic without a real window.
type DesktopWindowForTest struct {
	w desktopWindow
}

func NewDesktopWindowForTest() *DesktopWindowForTest {
	d := &DesktopWindowForTest{}
	d.w.ui = &UserInterface{}
	d.w.init()
	return d
}

func (d *DesktopWindowForTest) SetSizeLimits(minw, minh, maxw, maxh int) {
	d.w.setWindowSizeLimitsInDIP(minw, minh, maxw, maxh)
}

func (d *DesktopWindowForTest) SetResizingMode(mode WindowResizingMode) {
	d.w.SetResizingMode(mode)
}

func (d *DesktopWindowForTest) Restore() {
	d.w.Restore()
}

// InstallFakeBackendForTest installs a fake running backend whose window reports the given
// maximized state, and returns a handle that records whether Restore reached it.
func (d *DesktopWindowForTest) InstallFakeBackendForTest(maximized bool) *FakeBackendWindowForTest {
	fw := &FakeBackendWindowForTest{maximized: maximized}
	d.w.ui.setRunningBackend(&fakeUIBackendForTest{window: fw})
	return fw
}

// FakeBackendWindowForTest is a backendWindow whose only job is to record whether Restore was
// called on it.
type FakeBackendWindowForTest struct {
	backendWindow
	maximized bool

	Restored bool
}

func (w *FakeBackendWindowForTest) IsMaximized() bool {
	return w.maximized
}

func (w *FakeBackendWindowForTest) IsMinimized() bool {
	return false
}

func (w *FakeBackendWindowForTest) Restore() {
	w.Restored = true
}

type fakeUIBackendForTest struct {
	uiBackend
	window backendWindow
}

func (b *fakeUIBackendForTest) Window() backendWindow {
	return b.window
}
