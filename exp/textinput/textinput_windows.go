// Copyright 2024 The Ebitengine Authors
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

package textinput

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type textInput struct {
	session *session

	origWndProc     uintptr
	wndProcCallback uintptr
	window          windows.HWND
	immContext      uintptr

	highSurrogate uint16

	initOnce sync.Once

	err error
}

var theTextInput textInput

func (t *textInput) Start(x, y int) (<-chan State, func()) {
	if microsoftgdk.IsXbox() {
		return nil, nil
	}

	var session *session
	var err error
	ui.Get().RunOnMainThread(func() {
		t.end()
		err = t.start(x, y)
		session = newSession()
		t.session = session
	})
	if err != nil {
		session.ch <- State{Error: err}
		session.end()
	}
	return session.ch, func() {
		ui.Get().RunOnMainThread(func() {
			// Disable IME again.
			if t.immContext != 0 {
				return
			}
			c, err := _ImmAssociateContext(t.window, 0)
			if err != nil {
				t.err = err
				return
			}
			t.immContext = c
			t.end()
		})
	}
}

// start must be called from the main thread.
func (t *textInput) start(x, y int) error {
	if t.err != nil {
		return t.err
	}

	if t.window == 0 {
		t.window = _GetActiveWindow()
	}
	if t.origWndProc == 0 {
		if t.wndProcCallback == 0 {
			t.wndProcCallback = windows.NewCallback(t.wndProc)
		}
		// Note that a Win32API GetActiveWindow doesn't work on Xbox.
		h, err := _SetWindowLongPtrW(t.window, _GWL_WNDPROC, t.wndProcCallback)
		if err != nil {
			return err
		}
		t.origWndProc = h
	}

	// By default, IME was disabled by setting 0 as the IMM context.
	// Restore the context once.
	var err error
	t.initOnce.Do(func() {
		err = ui.Get().RestoreIMMContextOnMainThread()
	})
	if err != nil {
		return err
	}

	if t.immContext != 0 {
		if _, err := _ImmAssociateContext(t.window, t.immContext); err != nil {
			return err
		}
		t.immContext = 0
	}
	h := _ImmGetContext(t.window)
	if err := _ImmSetCandidateWindow(h, &_CANDIDATEFORM{
		dwIndex: 0,
		dwStyle: _CFS_CANDIDATEPOS,
		ptCurrentPos: _POINT{
			x: int32(x),
			y: int32(y),
		},
	}); err != nil {
		return err
	}
	if err := _ImmReleaseContext(t.window, h); err != nil {
		return err
	}
	return nil
}

func (t *textInput) wndProc(hWnd uintptr, uMsg uint32, wParam, lParam uintptr) uintptr {
	if t.session == nil {
		return _CallWindowProcW(t.origWndProc, hWnd, uMsg, wParam, lParam)
	}

	switch uMsg {
	case _WM_IME_SETCONTEXT:
		// Draw preedit text by an application side.
		if lParam&_ISC_SHOWUICOMPOSITIONWINDOW != 0 {
			lParam &^= _ISC_SHOWUICOMPOSITIONWINDOW
		}
	case _WM_IME_COMPOSITION:
		if lParam&(_GCS_RESULTSTR|_GCS_COMPSTR) != 0 {
			if lParam&_GCS_RESULTSTR != 0 {
				if err := t.commit(); err != nil {
					t.session.ch <- State{Error: err}
					t.end()
				}
			}
			if lParam&_GCS_COMPSTR != 0 {
				if err := t.update(); err != nil {
					t.session.ch <- State{Error: err}
					t.end()
				}
			}
			return 1
		}
	case _WM_CHAR, _WM_SYSCHAR:
		if wParam >= 0xd800 && wParam <= 0xdbff {
			t.highSurrogate = uint16(wParam)
		} else {
			var c rune
			if wParam >= 0xdc00 && wParam <= 0xdfff {
				if t.highSurrogate != 0 {
					c += (rune(t.highSurrogate) - 0xd800) << 10
					c += (rune(wParam) & 0xffff) - 0xdc00
					c += 0x10000
				}
			} else {
				c = rune(wParam) & 0xffff
			}
			t.highSurrogate = 0
			if c >= 0x20 {
				str := string(c)
				t.send(str, 0, len(str), true)
			}
		}
	case _WM_UNICHAR:
		if wParam == _UNICODE_NOCHAR {
			// WM_UNICHAR is not sent by Windows, but is sent by some third-party input method engine.
			// Returning TRUE here announces support for this message.
			return 1
		}
		if r := rune(wParam); r >= 0x20 {
			str := string(r)
			t.send(str, 0, len(str), true)
		}
	}

	return _CallWindowProcW(t.origWndProc, hWnd, uMsg, wParam, lParam)
}

// send must be called from the main thread.
func (t *textInput) send(text string, startInBytes, endInBytes int, committed bool) {
	if t.session != nil {
		t.session.trySend(State{
			Text:                             text,
			CompositionSelectionStartInBytes: startInBytes,
			CompositionSelectionEndInBytes:   endInBytes,
			Committed:                        committed,
		})
	}
	if committed {
		t.end()
	}
}

// end must be called from the main thread.
func (t *textInput) end() {
	if t.session == nil {
		return
	}

	t.session.end()
	t.session = nil
}

// update must be called from the main thread.
func (t *textInput) update() (ferr error) {
	if t.err != nil {
		return t.err
	}

	hIMC := _ImmGetContext(t.window)
	defer func() {
		if err := _ImmReleaseContext(t.window, hIMC); err != nil && ferr != nil {
			ferr = err
		}
	}()

	bufferLen, err := _ImmGetCompositionStringW(hIMC, _GCS_COMPSTR, nil, 0)
	if err != nil {
		return err
	}
	if bufferLen == 0 {
		return nil
	}

	buffer16 := make([]uint16, bufferLen/uint32(unsafe.Sizeof(uint16(0))))
	if _, err := _ImmGetCompositionStringW(hIMC, _GCS_COMPSTR, unsafe.Pointer(&buffer16[0]), bufferLen); err != nil {
		return err
	}

	attrLen, err := _ImmGetCompositionStringW(hIMC, _GCS_COMPATTR, nil, 0)
	if err != nil {
		return err
	}
	attr := make([]byte, attrLen)
	if _, err := _ImmGetCompositionStringW(hIMC, _GCS_COMPATTR, unsafe.Pointer(&attr[0]), attrLen); err != nil {
		return err
	}

	clauseLen, err := _ImmGetCompositionStringW(hIMC, _GCS_COMPCLAUSE, nil, 0)
	if err != nil {
		return err
	}
	clause := make([]uint32, clauseLen/uint32(unsafe.Sizeof(uint32(0))))
	if _, err := _ImmGetCompositionStringW(hIMC, _GCS_COMPCLAUSE, unsafe.Pointer(&clause[0]), clauseLen); err != nil {
		return err
	}

	var start16 int
	var end16 int
	if len(clause) > 0 {
		for i, c := range clause[:len(clause)-1] {
			if int(c) == len(attr) {
				break
			}
			if attr[c] == _ATTR_TARGET_CONVERTED || attr[c] == _ATTR_TARGET_NOTCONVERTED {
				start16 = int(c)
				end16 = int(clause[i+1])
				break
			}
		}
	}
	text := windows.UTF16ToString(buffer16)
	t.send(text, convertUTF16CountToByteCount(text, start16), convertUTF16CountToByteCount(text, end16), false)

	return nil
}

// commit must be called from the main thread.
func (t *textInput) commit() (ferr error) {
	if t.err != nil {
		return t.err
	}

	hIMC := _ImmGetContext(t.window)
	defer func() {
		if err := _ImmReleaseContext(t.window, hIMC); err != nil && ferr != nil {
			ferr = err
		}
	}()

	bufferLen, err := _ImmGetCompositionStringW(hIMC, _GCS_RESULTSTR, nil, 0)
	if err != nil {
		return err
	}
	if bufferLen == 0 {
		return nil
	}

	buffer16 := make([]uint16, bufferLen/uint32(unsafe.Sizeof(uint16(0))))
	if _, err := _ImmGetCompositionStringW(hIMC, _GCS_RESULTSTR, unsafe.Pointer(&buffer16[0]), bufferLen); err != nil {
		return err
	}

	text := windows.UTF16ToString(buffer16)
	t.send(text, 0, len(text), true)

	return nil
}
