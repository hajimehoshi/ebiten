// Copyright 2023 The Ebitengine Authors
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

//go:build !ios

package textinput

// TODO: Remove Cgo after ebitengine/purego#143 is resolved.

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa
//
// #import <Cocoa/Cocoa.h>
//
// @interface TextInputClient : NSView<NSTextInputClient>
// @end
//
// static TextInputClient* getTextInputClient() {
//   static TextInputClient* textInputClient;
//   if (!textInputClient) {
//     textInputClient = [[TextInputClient alloc] init];
//   }
//   return textInputClient;
// }
//
// static void start(int x, int y) {
//   TextInputClient* textInputClient = getTextInputClient();
//   NSWindow* window = [[NSApplication sharedApplication] mainWindow];
//   [[window contentView] addSubview: textInputClient];
//   [window makeFirstResponder: textInputClient];
//
//   y = [[window contentView] frame].size.height - y - 4;
//   [textInputClient setFrame:NSMakeRect(x, y, 1, 1)];
// }
import "C"

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type textInput struct {
	// session must be accessed from the main thread.
	session *session
}

var theTextInput textInput

func (t *textInput) Start(x, y int) (chan State, func()) {
	var session *session
	ui.Get().RunOnMainThread(func() {
		t.end()
		C.start(C.int(x), C.int(y))
		session = newSession()
		t.session = session
	})
	return session.ch, session.end
}

//export ebitengine_textinput_update
func ebitengine_textinput_update(text *C.char, start, end C.int, committed C.int) {
	theTextInput.update(C.GoString(text), int(start), int(end), committed != 0)
}

func (t *textInput) update(text string, start, end int, committed bool) {
	if t.session != nil {
		startInBytes := convertUTF16CountToByteCount(text, start)
		endInBytes := convertUTF16CountToByteCount(text, end)
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

//export ebitengine_textinput_end
func ebitengine_textinput_end() {
	theTextInput.end()
}

func (t *textInput) end() {
	if t.session != nil {
		t.session.end()
		t.session = nil
	}
}
