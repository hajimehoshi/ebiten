// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

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
