// SPDX-License-Identifier: BSD-3-Clause
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build !windows

package glfw

/*
#include "glfw/src/context.c"
#include "glfw/src/init.c"
#include "glfw/src/input.c"
#include "glfw/src/monitor.c"
#include "glfw/src/window.c"
#include "glfw/src/osmesa_context.c"
*/
import "C"
