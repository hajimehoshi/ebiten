// Copyright 2022 The Ebitengine Authors
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

package cocoasdk

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var class_NSInvocation = objc.GetClass("NSInvocation\x00")

var (
	sel_invocationWithMethodSignature = objc.RegisterName("invocationWithMethodSignature:\x00")

	sel_setSelector        = objc.RegisterName("setSelector:\x00")
	sel_setTarget          = objc.RegisterName("setTarget:\x00")
	sel_setArgumentAtIndex = objc.RegisterName("setArgument:atIndex:\x00")
	sel_getReturnValue     = objc.RegisterName("getReturnValue:\x00")
	sel_invoke             = objc.RegisterName("invoke\x00")
)

type NSInvocation struct {
	objc.ID
}

func InvocationWithMethodSignature(sig NSMethodSignature) NSInvocation {
	return NSInvocation{objc.ID(class_NSInvocation).Send(sel_invocationWithMethodSignature, objc.ID(sig))}
}

func (inv NSInvocation) SetSelector(_cmd objc.SEL) {
	inv.Send(sel_setSelector, _cmd)
}

func (inv NSInvocation) SetTarget(target objc.ID) {
	inv.Send(sel_setTarget, target)
}

func (inv NSInvocation) SetArgumentAtIndex(arg unsafe.Pointer, idx int) {
	inv.Send(sel_setArgumentAtIndex, arg, idx)
}

func (inv NSInvocation) GetReturnValue(ret unsafe.Pointer) {
	inv.Send(sel_getReturnValue, ret)
}

func (inv NSInvocation) Invoke() {
	inv.Send(sel_invoke)
}
