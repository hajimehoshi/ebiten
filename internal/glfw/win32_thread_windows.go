// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2017 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

func (t *tls) create() error {
	if t.platform.allocated {
		panic("glfw: TLS must not be allocated")
	}

	i, err := _TlsAlloc()
	if err != nil {
		return err
	}
	t.platform.index = i
	t.platform.allocated = true
	return nil
}

func (t *tls) destroy() error {
	if t.platform.allocated {
		if err := _TlsFree(t.platform.index); err != nil {
			return err
		}
	}
	t.platform.allocated = false
	t.platform.index = 0
	return nil
}

func (t *tls) get() (uintptr, error) {
	if !t.platform.allocated {
		panic("glfw: TLS must be allocated")
	}

	return _TlsGetValue(t.platform.index)
}

func (t *tls) set(value uintptr) error {
	if !t.platform.allocated {
		panic("glfw: TLS must be allocated")
	}

	return _TlsSetValue(t.platform.index, value)
}
