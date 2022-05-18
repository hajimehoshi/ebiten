// SPDX-License-Identifier: MIT

package gl

import (
	"github.com/ebiten/purego"
)

func getProcAddress(namea string) uintptr {
	return purego.Dlsym(purego.RTLD_DEFAULT, namea)
}
