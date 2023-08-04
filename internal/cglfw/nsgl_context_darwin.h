// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2009-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

// NOTE: Many Cocoa enum values have been renamed and we need to build across
//       SDK versions where one is unavailable or deprecated.
//       We use the newer names in code and replace them with the older names if
//       the base SDK does not provide the newer names.

#if MAC_OS_X_VERSION_MAX_ALLOWED < 101400
 #define NSOpenGLContextParameterSwapInterval NSOpenGLCPSwapInterval
 #define NSOpenGLContextParameterSurfaceOpacity NSOpenGLCPSurfaceOpacity
#endif

#define _GLFW_PLATFORM_CONTEXT_STATE            _GLFWcontextNSGL nsgl
#define _GLFW_PLATFORM_LIBRARY_CONTEXT_STATE    _GLFWlibraryNSGL nsgl

#include <stdatomic.h>


// NSGL-specific per-context data
//
typedef struct _GLFWcontextNSGL
{
    id                pixelFormat;
    id                object;
} _GLFWcontextNSGL;

// NSGL-specific global data
//
typedef struct _GLFWlibraryNSGL
{
    // dlopen handle for OpenGL.framework (for glfwGetProcAddress)
    CFBundleRef     framework;
} _GLFWlibraryNSGL;


GLFWbool _glfwInitNSGL(void);
void _glfwTerminateNSGL(void);
GLFWbool _glfwCreateContextNSGL(_GLFWwindow* window,
                                const _GLFWctxconfig* ctxconfig,
                                const _GLFWfbconfig* fbconfig);
void _glfwDestroyContextNSGL(_GLFWwindow* window);

