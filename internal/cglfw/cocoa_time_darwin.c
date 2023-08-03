// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2009-2016 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

#include "internal.h"

#include <mach/mach_time.h>


//////////////////////////////////////////////////////////////////////////
//////                       GLFW internal API                      //////
//////////////////////////////////////////////////////////////////////////

// Initialise timer
//
void _glfwInitTimerNS(void)
{
    mach_timebase_info_data_t info;
    mach_timebase_info(&info);

    _glfw.timer.ns.frequency = (info.denom * 1e9) / info.numer;
}


//////////////////////////////////////////////////////////////////////////
//////                       GLFW platform API                      //////
//////////////////////////////////////////////////////////////////////////

uint64_t _glfwPlatformGetTimerValue(void)
{
    return mach_absolute_time();
}

uint64_t _glfwPlatformGetTimerFrequency(void)
{
    return _glfw.timer.ns.frequency;
}

