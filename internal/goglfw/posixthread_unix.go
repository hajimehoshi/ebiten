// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin

package goglfw

type platformTLSState struct {
	// TODO: Implement this.
}

func (t *tls) create() error {
	panic("goglfw: tls.create is not implemented yet")
}

func (t *tls) destroy() error {
	panic("goglfw: tls.destroy is not implemented yet")
}

func (t *tls) get() (uintptr, error) {
	panic("goglfw: tls.get is not implemented yet")
}

func (t *tls) set(value uintptr) error {
	panic("goglfw: tls.set is not implemented yet")
}
