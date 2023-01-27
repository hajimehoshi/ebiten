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
//   // Actually this method should be called on the main thread,
//   // but there is no way to do that in the current Ebitengine implementation.
//   // dispatch_(a)sync causes a deadlock.
//   // As this is for an invetigation of iOS errors, that's OK to leave this issue so far.
//   return [[UIApplication sharedApplication] applicationState];
// }
import "C"

import (
	"fmt"
)

// addErrorInfoForContextCreation adds an additional information to the error when creating an audio context.
// See also hajimehoshi/oto#93.
func addErrorInfoForContextCreation(err error) error {
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
