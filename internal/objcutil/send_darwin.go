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

package objcutil

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var msgSend uintptr

func init() {
	lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("objcutil: %w", err))
	}
	msgSend, err = purego.Dlsym(lib, "objc_msgSend")
	if err != nil {
		panic(fmt.Errorf("objcutil: %w", err))
	}
}

// maxMsgArgs is the largest number of arguments of a call, the receiver and the selector included.
const maxMsgArgs = 16

// msgCall holds the arguments and the result area of a call.
type msgCall struct {
	args   [maxMsgArgs]uintptr
	out    [4]uintptr
	pinner runtime.Pinner
}

var msgCallPool = sync.Pool{
	New: func() any {
		return &msgCall{}
	},
}

// send sends sel to id with the arguments in c and returns the integer result register.
func (c *msgCall) send(id objc.ID, sel objc.SEL, args []uintptr) uintptr {
	if 2+len(args) > len(c.args) {
		panic("objcutil: too many arguments")
	}
	c.args[0] = uintptr(id)
	c.args[1] = uintptr(sel)
	n := 2 + copy(c.args[2:], args)
	r, _, _ := purego.SyscallN(msgSend, c.args[:n]...)
	return r
}

// Send sends sel to id and returns the integer result register. args must not hold Go pointers.
// Floating-point arguments and results are not supported. A BOOL result must be masked with 0xff.
func Send(id objc.ID, sel objc.SEL, args ...uintptr) uintptr {
	c := msgCallPool.Get().(*msgCall)
	r := c.send(id, sel, args)
	msgCallPool.Put(c)
	return r
}

var (
	sel_valueForKey  = objc.RegisterName("valueForKey:")
	sel_getValueSize = objc.RegisterName("getValue:size:")
)

// ValueForKey copies size bytes of the NSValue that valueForKey: returns for key to out, and reports
// whether the value is not nil. It must be called inside an autorelease pool.
func ValueForKey(id objc.ID, key objc.ID, out unsafe.Pointer, size uintptr) bool {
	v := objc.ID(Send(id, sel_valueForKey, uintptr(key)))
	if v == 0 {
		return false
	}

	c := msgCallPool.Get().(*msgCall)
	defer msgCallPool.Put(c)
	if size > unsafe.Sizeof(c.out) {
		panic("objcutil: unsupported value size")
	}
	clear(c.out[:])
	c.pinner.Pin(c)
	defer c.pinner.Unpin()
	c.send(v, sel_getValueSize, []uintptr{uintptr(unsafe.Pointer(&c.out)), size})
	copy(unsafe.Slice((*byte)(out), size), unsafe.Slice((*byte)(unsafe.Pointer(&c.out)), size))
	return true
}
