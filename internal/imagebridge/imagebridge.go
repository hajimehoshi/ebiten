// Copyright 2026 The Ebitengine Authors
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

// Package imagebridge gives sibling packages access to the internals of the root package's Image.
//
// The root package calls [Set] at initialization, and sibling packages call [Get] afterwards.
// The image type is a type parameter since this package cannot import the root package.
package imagebridge

// Bridge is the set of functions accessing the internals of Image.
type Bridge[Image any] struct {
	// OriginalImage returns the original image of a sub-image, or the image itself otherwise.
	OriginalImage func(img Image) Image

	// AddUsage registers a callback that is invoked whenever the image is used, and returns a token to
	// remove it. The callback receives the original image.
	AddUsage func(img Image, callback func(img Image)) int64

	// RemoveUsage removes the callback registered with the token.
	RemoveUsage func(img Image, token int64)
}

var theBridge any

// Set installs the bridge.
func Set[Image any](bridge Bridge[Image]) {
	theBridge = bridge
}

// Get returns the bridge installed by [Set]. Get panics if the bridge was set with a different image type.
func Get[Image any]() Bridge[Image] {
	return theBridge.(Bridge[Image])
}
