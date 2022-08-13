package cocoasdk

import (
	"github.com/ebitengine/purego/objc"
	"unsafe"
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
