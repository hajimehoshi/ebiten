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

//go:build linux && !android && !nintendosdk && !playstation5

package ui

import (
	"encoding/binary"
	"os"
	"path/filepath"
)

// evdevKeyboard reads keyboard keys from the Linux input layer; each
// /dev/input/event* device is read on its own goroutine.
type evdevKeyboard struct {
	onKey func(key Key, pressed bool)
	files []*os.File
}

func startEvdevKeyboard(onKey func(key Key, pressed bool)) *evdevKeyboard {
	k := &evdevKeyboard{onKey: onKey}
	paths, _ := filepath.Glob("/dev/input/event*")
	for _, p := range paths {
		f, err := os.OpenFile(p, os.O_RDONLY, 0)
		if err != nil {
			continue
		}
		k.files = append(k.files, f)
		go k.read(f)
	}
	return k
}

func (k *evdevKeyboard) read(f *os.File) {
	const (
		eventSize = 24 // 64-bit input_event: 16-byte timeval, u16 type, u16 code, s32 value
		evKey     = 0x01
	)
	var buf [eventSize * 32]byte
	for {
		n, err := f.Read(buf[:])
		if err != nil {
			return
		}
		for off := 0; off+eventSize <= n; off += eventSize {
			rec := buf[off : off+eventSize]
			typ := binary.LittleEndian.Uint16(rec[16:18])
			if typ != evKey {
				continue
			}
			code := binary.LittleEndian.Uint16(rec[18:20])
			value := int32(binary.LittleEndian.Uint32(rec[20:24]))
			key, ok := evdevKeyMap[code]
			if !ok {
				continue
			}
			switch value {
			case 1:
				k.onKey(key, true)
			case 0:
				k.onKey(key, false)
			}
		}
	}
}

func (k *evdevKeyboard) close() {
	for _, f := range k.files {
		_ = f.Close()
	}
	k.files = nil
}

// evdevKeyMap translates Linux KEY_* codes (linux/input-event-codes.h) to ui.Key.
var evdevKeyMap = map[uint16]Key{
	1:   KeyEscape,
	14:  KeyBackspace,
	15:  KeyTab,
	28:  KeyEnter,
	57:  KeySpace,
	103: KeyArrowUp,
	108: KeyArrowDown,
	105: KeyArrowLeft,
	106: KeyArrowRight,
	29:  KeyControlLeft,
	97:  KeyControlRight,
	42:  KeyShiftLeft,
	54:  KeyShiftRight,
	56:  KeyAltLeft,
	100: KeyAltRight,
	111: KeyDelete,
	102: KeyHome,
	107: KeyEnd,
	104: KeyPageUp,
	109: KeyPageDown,

	// Number row (KEY_1..KEY_0 = 2..11).
	2: KeyDigit1, 3: KeyDigit2, 4: KeyDigit3, 5: KeyDigit4, 6: KeyDigit5,
	7: KeyDigit6, 8: KeyDigit7, 9: KeyDigit8, 10: KeyDigit9, 11: KeyDigit0,

	// Numeric keypad (KEY_KP7=71 .. KEY_KP0=82, KEY_KPENTER=96).
	71: KeyNumpad7, 72: KeyNumpad8, 73: KeyNumpad9,
	75: KeyNumpad4, 76: KeyNumpad5, 77: KeyNumpad6,
	79: KeyNumpad1, 80: KeyNumpad2, 81: KeyNumpad3,
	82: KeyNumpad0, 96: KeyNumpadEnter,
	55: KeyNumpadMultiply, 98: KeyNumpadDivide,
	74: KeyNumpadSubtract, 78: KeyNumpadAdd,

	// Letters (KEY_A..KEY_Z in scan order).
	30: KeyA, 48: KeyB, 46: KeyC, 32: KeyD, 18: KeyE, 33: KeyF, 34: KeyG,
	35: KeyH, 23: KeyI, 36: KeyJ, 37: KeyK, 38: KeyL, 50: KeyM, 49: KeyN,
	24: KeyO, 25: KeyP, 16: KeyQ, 19: KeyR, 31: KeyS, 20: KeyT, 22: KeyU,
	47: KeyV, 17: KeyW, 45: KeyX, 21: KeyY, 44: KeyZ,

	// Function keys.
	59: KeyF1, 60: KeyF2, 61: KeyF3, 62: KeyF4, 63: KeyF5, 64: KeyF6,
	65: KeyF7, 66: KeyF8, 67: KeyF9, 68: KeyF10, 87: KeyF11, 88: KeyF12,
}
