// Copyright 2002-2006 Marcus Geelnard
// Copyright 2006-2019 Camilla Löwy
// Copyright 2022 The Ebiten Authors
//
// This software is provided 'as-is', without any express or implied
// warranty. In no event will the authors be held liable for any damages
// arising from the use of this software.
//
// Permission is granted to anyone to use this software for any purpose,
// including commercial applications, and to alter it and redistribute it
// freely, subject to the following restrictions:
//
// 1. The origin of this software must not be misrepresented; you must not
//    claim that you wrote the original software. If you use this software
//    in a product, an acknowledgment in the product documentation would
//    be appreciated but is not required.
//
// 2. Altered source versions must be plainly marked as such, and must not
//    be misrepresented as being the original software.
//
// 3. This notice may not be removed or altered from any source
//    distribution.

package glfwwin

func (t *tls) create() error {
	if t.win32.allocated {
		panic("glfwwin: TLS must not be allocated")
	}

	i, err := _TlsAlloc()
	if err != nil {
		return err
	}
	t.win32.index = i
	t.win32.allocated = true
	return nil
}

func (t *tls) destroy() error {
	if t.win32.allocated {
		if err := _TlsFree(t.win32.index); err != nil {
			return err
		}
	}
	t.win32.allocated = false
	t.win32.index = 0
	return nil
}

func (t *tls) get() (uintptr, error) {
	if !t.win32.allocated {
		panic("glfwwin: TLS must be allocated")
	}

	return _TlsGetValue(t.win32.index)
}

func (t *tls) set(value uintptr) error {
	if !t.win32.allocated {
		panic("glfwwin: TLS must be allocated")
	}

	return _TlsSetValue(t.win32.index, value)
}
