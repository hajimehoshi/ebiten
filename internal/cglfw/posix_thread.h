// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2017 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

#include <pthread.h>

#define _GLFW_PLATFORM_TLS_STATE    _GLFWtlsPOSIX   posix
#define _GLFW_PLATFORM_MUTEX_STATE  _GLFWmutexPOSIX posix


// POSIX-specific thread local storage data
//
typedef struct _GLFWtlsPOSIX
{
    GLFWbool        allocated;
    pthread_key_t   key;
} _GLFWtlsPOSIX;

// POSIX-specific mutex data
//
typedef struct _GLFWmutexPOSIX
{
    GLFWbool        allocated;
    pthread_mutex_t handle;
} _GLFWmutexPOSIX;

