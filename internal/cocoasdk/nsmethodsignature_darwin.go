package cocoasdk

import "github.com/ebitengine/purego/objc"

var class_NSMethodSignature = objc.GetClass("NSMethodSignature\x00")

var (
	sel_instanceMethodSignatureForSelector = objc.RegisterName("instanceMethodSignatureForSelector:\x00")
	sel_signatureWithObjCTypes             = objc.RegisterName("signatureWithObjCTypes:\x00")
)

type NSMethodSignature objc.ID

func InstanceMethodSignatureForSelector(self objc.ID, _cmd objc.SEL) NSMethodSignature {
	return NSMethodSignature(self.Send(sel_instanceMethodSignatureForSelector, _cmd))
}

func NSMethodSignature_SignatureWithObjCTypes(types string) NSMethodSignature {
	return NSMethodSignature(objc.ID(class_NSMethodSignature).Send(sel_signatureWithObjCTypes, types))
}
