// SPDX-License-Identifier: MIT

package gl

import (
	"github.com/ebiten/purego"
	"unsafe"
)

func getProcAddress(namea string) unsafe.Pointer {
	return unsafe.Pointer(purego.Dlsym(purego.RTLD_DEFAULT, namea))
}
