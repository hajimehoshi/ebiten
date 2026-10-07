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

package directx_test

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/directx"
)

// The native union has an 8-byte alignment and occupies 32 bytes on Windows.
const (
	textureCopyUnionOffset  = 2 * unsafe.Sizeof(uintptr(0))
	textureCopyLocationSize = textureCopyUnionOffset + 32
	subresourceIndexOffset  = unsafe.Offsetof(directx.TextureCopyLocationSubresourceIndex{}.SubresourceIndex)
	subresourceLocationSize = unsafe.Sizeof(directx.TextureCopyLocationSubresourceIndex{})
)

// Check both directions so cross-compiling the test verifies the ABI as well.
var (
	_ [subresourceIndexOffset - textureCopyUnionOffset]byte
	_ [textureCopyUnionOffset - subresourceIndexOffset]byte
	_ [subresourceLocationSize - textureCopyLocationSize]byte
	_ [textureCopyLocationSize - subresourceLocationSize]byte
)
