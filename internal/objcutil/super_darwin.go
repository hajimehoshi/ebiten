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
	"structs"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

type super struct {
	_        structs.HostLayout
	receiver objc.ID
	class    objc.Class
}

var (
	msgSendSuperBool func(*super, objc.SEL, ...any) bool
	msgSendSuperID   func(*super, objc.SEL, ...any) objc.ID
	msgSendSuperVoid func(*super, objc.SEL, ...any) struct{}
)

func init() {
	lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("objcutil: %w", err))
	}
	msgSendSuper, err := purego.Dlsym(lib, "objc_msgSendSuper")
	if err != nil {
		panic(fmt.Errorf("objcutil: %w", err))
	}
	purego.RegisterFunc(&msgSendSuperBool, msgSendSuper)
	purego.RegisterFunc(&msgSendSuperID, msgSendSuper)
	// PureGo treats a zero-sized struct return as having no native result.
	purego.RegisterFunc(&msgSendSuperVoid, msgSendSuper)
}

// SendSuper calls a method on the superclass of the class defining the calling method.
// Use struct{} for a void return.
//
// PureGo v0.11.0's SendSuper starts superclass lookup relative to the
// receiver's runtime class. When a subclass instance inherits the calling
// method, lookup can find that same method again and recurse indefinitely.
// Superclass dispatch must be relative to the class defining the method.
func SendSuper[T bool | objc.ID | struct{}](receiver objc.ID, definingClass objc.Class, sel objc.SEL, args ...any) T {
	// objc_msgSendSuper starts lookup at the supplied class, preserving the receiver.
	s := super{
		receiver: receiver,
		class:    definingClass.SuperClass(),
	}
	var zero T
	switch any(zero).(type) {
	case bool:
		return any(msgSendSuperBool(&s, sel, args...)).(T)
	case objc.ID:
		return any(msgSendSuperID(&s, sel, args...)).(T)
	case struct{}:
		return any(msgSendSuperVoid(&s, sel, args...)).(T)
	}
	panic("objcutil: unsupported return type")
}
