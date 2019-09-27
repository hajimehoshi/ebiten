// Copyright 2016 Hajime Hoshi
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

// +build js

package clock

import (
	"syscall/js"
	"time"
)

var (
	jsPerformance = js.Global().Get("performance")
	jsNow         = jsPerformance.Get("now").Call("bind", jsPerformance)
)

func now() int64 {
	// time.Now() is not reliable until GopherJS supports performance.now().
	//
	// performance.now is monotonic:
	// https://www.w3.org/TR/hr-time-2/#sec-monotonic-clock
	return int64(jsNow.Invoke().Float() * float64(time.Millisecond))
}
