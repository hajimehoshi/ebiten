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

package objcutil_test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/objcutil"
)

var classSerial atomic.Uint64

func TestSendSuperInherited(t *testing.T) {
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("EbitengineSuperTest%d", classSerial.Add(1))
	register := func(name string, parent objc.Class, methods []objc.MethodDef) objc.Class {
		t.Helper()
		class, err := objc.RegisterClass(prefix+name, parent, nil, nil, methods)
		if err != nil {
			t.Fatal(err)
		}
		return class
	}
	selVoid := objc.RegisterName("ebitengineSuperVoid:second:")
	selBool := objc.RegisterName("ebitengineSuperBool:")
	selID := objc.RegisterName("ebitengineSuperID:")
	selDealloc := objc.RegisterName("dealloc")
	var receiver objc.ID
	var receiverClass objc.Class
	var baseCalls, methodCalls, deallocCalls int
	checkReceiver := func(self objc.ID, cmd, wantCmd objc.SEL) {
		t.Helper()
		if self != receiver {
			t.Errorf("receiver = %v, want %v", self, receiver)
		}
		if got := self.Class(); got != receiverClass {
			t.Errorf("receiver class = %v, want %v", got, receiverClass)
		}
		if cmd != wantCmd {
			t.Errorf("selector = %v, want %v", cmd, wantCmd)
		}
	}
	base := register("Base", objc.GetClass("NSObject"), []objc.MethodDef{
		{
			Cmd: selVoid,
			Fn: func(self objc.ID, cmd objc.SEL, first, second objc.ID) {
				baseCalls++
				checkReceiver(self, cmd, selVoid)
				if first != receiver || second != 0 {
					t.Errorf("arguments = (%v, %v), want (%v, 0)", first, second, receiver)
				}
			},
		},
		{
			Cmd: selBool,
			Fn: func(self objc.ID, cmd objc.SEL, value bool) bool {
				baseCalls++
				checkReceiver(self, cmd, selBool)
				return value
			},
		},
		{
			Cmd: selID,
			Fn: func(self objc.ID, cmd objc.SEL, value objc.ID) objc.ID {
				baseCalls++
				checkReceiver(self, cmd, selID)
				return value
			},
		},
	})
	var child1 objc.Class
	// Stop recursive redispatch so a regression fails without exhausting the stack.
	enter := func() bool {
		methodCalls++
		if methodCalls > 1 {
			t.Error("super dispatch reentered the overriding method")
			return false
		}
		return true
	}
	child1 = register("Child1", base, []objc.MethodDef{
		{
			Cmd: selVoid,
			Fn: func(self objc.ID, cmd objc.SEL, first, second objc.ID) {
				if enter() {
					objcutil.SendSuper[struct{}](self, child1, cmd, first, second)
				}
			},
		},
		{
			Cmd: selBool,
			Fn: func(self objc.ID, cmd objc.SEL, value bool) bool {
				if !enter() {
					return false
				}
				return objcutil.SendSuper[bool](self, child1, cmd, value)
			},
		},
		{
			Cmd: selID,
			Fn: func(self objc.ID, cmd objc.SEL, value objc.ID) objc.ID {
				if !enter() {
					return 0
				}
				return objcutil.SendSuper[objc.ID](self, child1, cmd, value)
			},
		},
		{
			Cmd: selDealloc,
			Fn: func(self objc.ID, cmd objc.SEL) {
				deallocCalls++
				if deallocCalls > 1 {
					t.Error("super dispatch reentered dealloc")
					return
				}
				checkReceiver(self, cmd, selDealloc)
				objcutil.SendSuper[struct{}](self, child1, cmd)
			},
		},
	})
	child2 := register("Child2", child1, nil)
	child3 := register("Child3", child2, nil)
	for _, tt := range []struct {
		name  string
		class objc.Class
	}{
		{
			name:  "Child1",
			class: child1,
		},
		{
			name:  "Child2",
			class: child2,
		},
		{
			name:  "Child3",
			class: child3,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			receiverClass = tt.class
			receiver = objc.ID(tt.class).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
			if receiver == 0 {
				t.Fatal("alloc/init returned nil")
			}
			checkCalls := func() {
				t.Helper()
				if baseCalls != 1 || methodCalls != 1 {
					t.Errorf("base/override calls = %d/%d, want 1/1", baseCalls, methodCalls)
				}
				baseCalls, methodCalls = 0, 0
			}
			receiver.Send(selVoid, receiver, objc.ID(0))
			checkCalls()
			for _, want := range []bool{false, true} {
				if got := objc.Send[bool](receiver, selBool, want); got != want {
					t.Errorf("boolean result = %v, want %v", got, want)
				}
				checkCalls()
			}
			for _, want := range []objc.ID{0, receiver} {
				if got := receiver.Send(selID, want); got != want {
					t.Errorf("object result = %v, want %v", got, want)
				}
				checkCalls()
			}
			deallocCalls = 0
			receiver.Send(objc.RegisterName("release"))
			if deallocCalls != 1 {
				t.Errorf("dealloc calls = %d, want 1", deallocCalls)
			}
		})
	}
}
