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

//go:build go1.13
// +build go1.13

package ebiten

// Ebiten forces to use Go 1.13 or later, since os.UserConfigDir is defined as of Go 1.13.

const __EBITEN_REQUIRES_GO_VERSION_1_13_OR_LATER__ = true
