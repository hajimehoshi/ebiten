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
// #include <stdatomic.h>
//
// enum { applicationStateUnknown = -1 };
// static _Atomic(NSInteger) cachedApplicationState = applicationStateUnknown;
//
// #cgo noescape applicationState
// #cgo nocallback applicationState
// static NSInteger applicationState() {
//   if ([NSThread isMainThread]) {
//     UIApplicationState state = [[UIApplication sharedApplication] applicationState];
//     atomic_store(&cachedApplicationState, state);
//     return state;
//   }
//   // The main thread can be waiting for the game loop to finish a frame.
//   // Return the last observed state without waiting for the main dispatch queue.
//   dispatch_async(dispatch_get_main_queue(), ^{
//     atomic_store(&cachedApplicationState, [[UIApplication sharedApplication] applicationState]);
//   });
//   return atomic_load(&cachedApplicationState);
// }
import "C"

import (
	"fmt"
)

// addErrorInfo adds the application state to an error the audio device reported.
//
// This is applied to every error the device reports, not only to the one from creating
// a context: the errors from Suspend, Resume and Err come from the device too, and
// (*Player).Seek reports a device error as well.
// Off the main thread, the state is the last observed value, or unknown before
// the first observation; refreshing it must not wait for the main thread.
// See also ebitengine/oto#93.
func addErrorInfo(err error) error {
	if err == nil {
		return nil
	}

	var state string
	switch s := C.applicationState(); s {
	case C.applicationStateUnknown:
		state = "unknown"
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
