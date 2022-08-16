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

package cocoa

import (
	"github.com/ebitengine/purego/objc"
)

var class_NSMethodSignature = objc.GetClass("NSMethodSignature")

var (
	sel_instanceMethodSignatureForSelector = objc.RegisterName("instanceMethodSignatureForSelector:")
	sel_signatureWithObjCTypes             = objc.RegisterName("signatureWithObjCTypes:")
)

type NSMethodSignature objc.ID

func NSMethodSignature_InstanceMethodSignatureForSelector(self objc.ID, _cmd objc.SEL) NSMethodSignature {
	return NSMethodSignature(self.Send(sel_instanceMethodSignatureForSelector, _cmd))
}

// NSMethodSignature_SignatureWithObjCTypes takes a string that represents the type signature of a method.
// It follows the encoding specified in the Apple Docs.
//
// [Apple Docs]: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/ObjCRuntimeGuide/Articles/ocrtTypeEncodings.html#//apple_ref/doc/uid/TP40008048-CH100
func NSMethodSignature_SignatureWithObjCTypes(types string) NSMethodSignature {
	return NSMethodSignature(objc.ID(class_NSMethodSignature).Send(sel_signatureWithObjCTypes, types))
}
