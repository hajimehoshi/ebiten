// Copyright 2022 The Ebiten Authors
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

package audio

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Foundation -framework UIKit
//
// #import <UIKit/UIKit.h>
//
// static UIApplicationState applicationState() {
//   if ([NSThread isMainThread]) {
//     return [[UIApplication sharedApplication] applicationState];
//   }
//   __block UIApplicationState state;
//   dispatch_sync(dispatch_get_main_queue(), ^{
//     state = [[UIApplication sharedApplication] applicationState];
//   });
//   return state;
// }
import "C"

import (
	"fmt"
)

// addErrorInfo adds an additional information to the error when creating an audio context.
// See also ebitengine/oto#93.
func addErrorInfo(err error) error {
	if err == nil {
		return nil
	}

	var state string
	switch s := C.applicationState(); s {
	case C.UIApplicationStateActive:
		state = "active"
	case C.UIApplicationStateInactive:
		state = "inactive"
	case C.UIApplicationStateBackground:
		state = "background"
	default:
		state = fmt.Sprintf("UIApplicationState(%d)", s)
	}
	return fmt.Errorf("%w, application state: %s", err, state)
}
