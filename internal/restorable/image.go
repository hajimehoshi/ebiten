// Copyright 2016 The Ebiten Authors
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

package restorable

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
)

// Image represents an image.
type Image struct {
	// Image is the underlying image.
	// This member is exported on purpose.
	// TODO: Move the implementation to internal/atlas package (#805).
	Image *graphicscommand.Image
}
