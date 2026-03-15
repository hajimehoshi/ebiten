// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2017 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

package glfw

import "sync"

func (t *tls) create() error {
	if t.platform.allocated {
		panic("glfw: TLS must not be allocated")
	}
	t.platform.allocated = true
	return nil
}

func (t *tls) destroy() error {
	t.platform = platformTLSState{}
	return nil
}

func (t *tls) get() (uintptr, error) {
	if !t.platform.allocated {
		panic("glfw: TLS must be allocated")
	}
	return t.platform.value, nil
}

func (t *tls) set(value uintptr) error {
	if !t.platform.allocated {
		panic("glfw: TLS must be allocated")
	}
	t.platform.value = value
	return nil
}

type mutex struct {
	mu sync.Mutex
}

func (m *mutex) lock() {
	m.mu.Lock()
}

func (m *mutex) unlock() {
	m.mu.Unlock()
}
