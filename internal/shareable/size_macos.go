// Copyright 2018 The Ebiten Authors
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

// +build darwin
// +build !ios
// +build !js

package shareable

// On MacBook Pro 2013 (Late), there is a bug in texture rendering and
// extending shareable textures sometimes fail (#593). This is due to
// a bug in the grahics driver, and there is nothing we can do. Let's
// not extend shareable textures in such environment.

const (
	initSize = 4096
	maxSize  = 4096
)
