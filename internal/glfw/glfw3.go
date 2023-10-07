// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd || windows

package glfw

type VidMode struct {
	Width       int
	Height      int
	RedBits     int
	GreenBits   int
	BlueBits    int
	RefreshRate int
}

type Image struct {
	Width  int
	Height int
	Pixels []byte
}
