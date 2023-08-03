// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2017 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

#define _GLFW_PLATFORM_LIBRARY_TIMER_STATE _GLFWtimerPOSIX posix

#include <stdint.h>


// POSIX-specific global timer data
//
typedef struct _GLFWtimerPOSIX
{
    GLFWbool    monotonic;
    uint64_t    frequency;
} _GLFWtimerPOSIX;


void _glfwInitTimerPOSIX(void);

