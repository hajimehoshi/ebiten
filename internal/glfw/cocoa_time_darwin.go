// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2009-2016 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

package glfw

import "time"

func initTimerNS() {
}

func platformGetTimerValue() uint64 {
	return uint64(time.Now().UnixNano())
}

func platformGetTimerFrequency() uint64 {
	return 1_000_000_000
}
