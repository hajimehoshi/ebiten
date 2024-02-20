// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2017 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

#include "internal_unix.h"

#include <assert.h>
#include <string.h>


//////////////////////////////////////////////////////////////////////////
//////                       GLFW platform API                      //////
//////////////////////////////////////////////////////////////////////////

GLFWbool _glfwPlatformCreateTls(_GLFWtls* tls)
{
    assert(tls->posix.allocated == GLFW_FALSE);

    if (pthread_key_create(&tls->posix.key, NULL) != 0)
    {
        _glfwInputError(GLFW_PLATFORM_ERROR,
                        "POSIX: Failed to create context TLS");
        return GLFW_FALSE;
    }

    tls->posix.allocated = GLFW_TRUE;
    return GLFW_TRUE;
}

void _glfwPlatformDestroyTls(_GLFWtls* tls)
{
    if (tls->posix.allocated)
        pthread_key_delete(tls->posix.key);
    memset(tls, 0, sizeof(_GLFWtls));
}

void* _glfwPlatformGetTls(_GLFWtls* tls)
{
    assert(tls->posix.allocated == GLFW_TRUE);
    return pthread_getspecific(tls->posix.key);
}

void _glfwPlatformSetTls(_GLFWtls* tls, void* value)
{
    assert(tls->posix.allocated == GLFW_TRUE);
    pthread_setspecific(tls->posix.key, value);
}

GLFWbool _glfwPlatformCreateMutex(_GLFWmutex* mutex)
{
    assert(mutex->posix.allocated == GLFW_FALSE);

    if (pthread_mutex_init(&mutex->posix.handle, NULL) != 0)
    {
        _glfwInputError(GLFW_PLATFORM_ERROR, "POSIX: Failed to create mutex");
        return GLFW_FALSE;
    }

    return mutex->posix.allocated = GLFW_TRUE;
}

void _glfwPlatformDestroyMutex(_GLFWmutex* mutex)
{
    if (mutex->posix.allocated)
        pthread_mutex_destroy(&mutex->posix.handle);
    memset(mutex, 0, sizeof(_GLFWmutex));
}

void _glfwPlatformLockMutex(_GLFWmutex* mutex)
{
    assert(mutex->posix.allocated == GLFW_TRUE);
    pthread_mutex_lock(&mutex->posix.handle);
}

void _glfwPlatformUnlockMutex(_GLFWmutex* mutex)
{
    assert(mutex->posix.allocated == GLFW_TRUE);
    pthread_mutex_unlock(&mutex->posix.handle);
}

