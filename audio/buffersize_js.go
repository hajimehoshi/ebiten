// Copyright 2021 The Ebiten Authors
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

import (
	"syscall/js"
)

var isGo2Cpp bool = js.Global().Get("go2cpp").Truthy()

func bufferSize() int {
	// On some devices targetted by go2cpp, 8192 is not enough. Use x2 bytes.
	if isGo2Cpp {
		return 16384
	}
	return 8192
}
