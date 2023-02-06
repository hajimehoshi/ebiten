// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package goglfw

func (t *tls) create() error {
	if t.state.allocated {
		panic("goglfw: TLS must not be allocated")
	}

	i, err := _TlsAlloc()
	if err != nil {
		return err
	}
	t.state.index = i
	t.state.allocated = true
	return nil
}

func (t *tls) destroy() error {
	if t.state.allocated {
		if err := _TlsFree(t.state.index); err != nil {
			return err
		}
	}
	t.state.allocated = false
	t.state.index = 0
	return nil
}

func (t *tls) get() (uintptr, error) {
	if !t.state.allocated {
		panic("goglfw: TLS must be allocated")
	}

	return _TlsGetValue(t.state.index)
}

func (t *tls) set(value uintptr) error {
	if !t.state.allocated {
		panic("goglfw: TLS must be allocated")
	}

	return _TlsSetValue(t.state.index, value)
}
