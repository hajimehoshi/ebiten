// Copyright 2019 The Ebiten Authors
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

// +build go1.12

package ebiten

// Ebiten forces to use Go 1.12 or later, since
// 1) Between Go 1.10 and Go 1.11, ioutil.TempFile's behavior is different. Ebiten forces the Go version in order to avoid confusion. (#777)
// 2) FuncOf in syscall/js is defined as of Go 1.12.

const __EBITEN_REQUIRES_GO_VERSION_1_12_OR_LATER__ = true
