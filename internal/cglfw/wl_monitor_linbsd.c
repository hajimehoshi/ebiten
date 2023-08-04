// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2014 Jonas Ådahl <jadahl@gmail.com>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build (freebsd || linux || netbsd || openbsd) && wayland

#include "internal.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <math.h>


static void outputHandleGeometry(void* userData,
                                 struct wl_output* output,
                                 int32_t x,
                                 int32_t y,
                                 int32_t physicalWidth,
                                 int32_t physicalHeight,
                                 int32_t subpixel,
                                 const char* make,
                                 const char* model,
                                 int32_t transform)
{
    struct _GLFWmonitor* monitor = userData;

    monitor->wl.x = x;
    monitor->wl.y = y;
    monitor->widthMM = physicalWidth;
    monitor->heightMM = physicalHeight;

    if (strlen(monitor->name) == 0)
        snprintf(monitor->name, sizeof(monitor->name), "%s %s", make, model);
}

static void outputHandleMode(void* userData,
                             struct wl_output* output,
                             uint32_t flags,
                             int32_t width,
                             int32_t height,
                             int32_t refresh)
{
    struct _GLFWmonitor* monitor = userData;
    GLFWvidmode mode;

    mode.width = width;
    mode.height = height;
    mode.redBits = 8;
    mode.greenBits = 8;
    mode.blueBits = 8;
    mode.refreshRate = (int) round(refresh / 1000.0);

    monitor->modeCount++;
    monitor->modes =
        realloc(monitor->modes, monitor->modeCount * sizeof(GLFWvidmode));
    monitor->modes[monitor->modeCount - 1] = mode;

    if (flags & WL_OUTPUT_MODE_CURRENT)
        monitor->wl.currentMode = monitor->modeCount - 1;
}

static void outputHandleDone(void* userData, struct wl_output* output)
{
    struct _GLFWmonitor* monitor = userData;

    if (monitor->widthMM <= 0 || monitor->heightMM <= 0)
    {
        // If Wayland does not provide a physical size, assume the default 96 DPI
        const GLFWvidmode* mode = &monitor->modes[monitor->wl.currentMode];
        monitor->widthMM  = (int) (mode->width * 25.4f / 96.f);
        monitor->heightMM = (int) (mode->height * 25.4f / 96.f);
    }

    for (int i = 0; i < _glfw.monitorCount; i++)
    {
        if (_glfw.monitors[i] == monitor)
            return;
    }

    _glfwInputMonitor(monitor, GLFW_CONNECTED, _GLFW_INSERT_LAST);
}

static void outputHandleScale(void* userData,
                              struct wl_output* output,
                              int32_t factor)
{
    struct _GLFWmonitor* monitor = userData;

    monitor->wl.scale = factor;

    for (_GLFWwindow* window = _glfw.windowListHead; window; window = window->next)
    {
        for (int i = 0; i < window->wl.monitorsCount; i++)
        {
            if (window->wl.monitors[i] == monitor)
            {
                _glfwUpdateContentScaleWayland(window);
                break;
            }
        }
    }
}

#ifdef WL_OUTPUT_NAME_SINCE_VERSION

void outputHandleName(void* userData, struct wl_output* wl_output, const char* name)
{
    struct _GLFWmonitor* monitor = userData;

    strncpy(monitor->name, name, sizeof(monitor->name) - 1);
}

void outputHandleDescription(void* userData,
                             struct wl_output* wl_output,
                             const char* description)
{
}

#endif // WL_OUTPUT_NAME_SINCE_VERSION

static const struct wl_output_listener outputListener =
{
    outputHandleGeometry,
    outputHandleMode,
    outputHandleDone,
    outputHandleScale,
#ifdef WL_OUTPUT_NAME_SINCE_VERSION
    outputHandleName,
    outputHandleDescription,
#endif
};


//////////////////////////////////////////////////////////////////////////
//////                       GLFW internal API                      //////
//////////////////////////////////////////////////////////////////////////

void _glfwAddOutputWayland(uint32_t name, uint32_t version)
{
    if (version < 2)
    {
        _glfwInputError(GLFW_PLATFORM_ERROR,
                        "Wayland: Unsupported output interface version");
        return;
    }

#ifdef WL_OUTPUT_NAME_SINCE_VERSION
    version = _glfw_min(version, WL_OUTPUT_NAME_SINCE_VERSION);
#else
    version = 2;
#endif

    struct wl_output* output = wl_registry_bind(_glfw.wl.registry,
                                                name,
                                                &wl_output_interface,
                                                version);
    if (!output)
        return;

    // The actual name of this output will be set in the geometry handler
    _GLFWmonitor* monitor = _glfwAllocMonitor("", 0, 0);
    monitor->wl.scale = 1;
    monitor->wl.output = output;
    monitor->wl.name = name;

    wl_output_add_listener(output, &outputListener, monitor);
}


//////////////////////////////////////////////////////////////////////////
//////                       GLFW platform API                      //////
//////////////////////////////////////////////////////////////////////////

void _glfwPlatformFreeMonitor(_GLFWmonitor* monitor)
{
    if (monitor->wl.output)
        wl_output_destroy(monitor->wl.output);
}

void _glfwPlatformGetMonitorPos(_GLFWmonitor* monitor, int* xpos, int* ypos)
{
    if (xpos)
        *xpos = monitor->wl.x;
    if (ypos)
        *ypos = monitor->wl.y;
}

void _glfwPlatformGetMonitorContentScale(_GLFWmonitor* monitor,
                                         float* xscale, float* yscale)
{
    if (xscale)
        *xscale = (float) monitor->wl.scale;
    if (yscale)
        *yscale = (float) monitor->wl.scale;
}

void _glfwPlatformGetMonitorWorkarea(_GLFWmonitor* monitor,
                                     int* xpos, int* ypos,
                                     int* width, int* height)
{
    if (xpos)
        *xpos = monitor->wl.x;
    if (ypos)
        *ypos = monitor->wl.y;
    if (width)
        *width = monitor->modes[monitor->wl.currentMode].width;
    if (height)
        *height = monitor->modes[monitor->wl.currentMode].height;
}

GLFWvidmode* _glfwPlatformGetVideoModes(_GLFWmonitor* monitor, int* found)
{
    *found = monitor->modeCount;
    return monitor->modes;
}

void _glfwPlatformGetVideoMode(_GLFWmonitor* monitor, GLFWvidmode* mode)
{
    *mode = monitor->modes[monitor->wl.currentMode];
}

GLFWbool _glfwPlatformGetGammaRamp(_GLFWmonitor* monitor, GLFWgammaramp* ramp)
{
    _glfwInputError(GLFW_PLATFORM_ERROR,
                    "Wayland: Gamma ramp access is not available");
    return GLFW_FALSE;
}

void _glfwPlatformSetGammaRamp(_GLFWmonitor* monitor,
                               const GLFWgammaramp* ramp)
{
    _glfwInputError(GLFW_PLATFORM_ERROR,
                    "Wayland: Gamma ramp access is not available");
}


//////////////////////////////////////////////////////////////////////////
//////                        GLFW native API                       //////
//////////////////////////////////////////////////////////////////////////

GLFWAPI struct wl_output* glfwGetWaylandMonitor(GLFWmonitor* handle)
{
    _GLFWmonitor* monitor = (_GLFWmonitor*) handle;
    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    return monitor->wl.output;
}

