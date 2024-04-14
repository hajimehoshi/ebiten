// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

#include "internal_unix.h"

#include <assert.h>
#include <float.h>
#include <math.h>
#include <stdlib.h>
#include <string.h>

// Internal key state used for sticky keys
#define _GLFW_STICK 3


//////////////////////////////////////////////////////////////////////////
//////                         GLFW event API                       //////
//////////////////////////////////////////////////////////////////////////

// Notifies shared code of a physical key event
//
void _glfwInputKey(_GLFWwindow* window, int key, int scancode, int action, int mods)
{
    if (key >= 0 && key <= GLFW_KEY_LAST)
    {
        GLFWbool repeated = GLFW_FALSE;

        if (action == GLFW_RELEASE && window->keys[key] == GLFW_RELEASE)
            return;

        if (action == GLFW_PRESS && window->keys[key] == GLFW_PRESS)
            repeated = GLFW_TRUE;

        if (action == GLFW_RELEASE && window->stickyKeys)
            window->keys[key] = _GLFW_STICK;
        else
            window->keys[key] = (char) action;

        if (repeated)
            action = GLFW_REPEAT;
    }

    if (!window->lockKeyMods)
        mods &= ~(GLFW_MOD_CAPS_LOCK | GLFW_MOD_NUM_LOCK);

    if (window->callbacks.key)
        window->callbacks.key((GLFWwindow*) window, key, scancode, action, mods);
}

// Notifies shared code of a Unicode codepoint input event
// The 'plain' parameter determines whether to emit a regular character event
//
void _glfwInputChar(_GLFWwindow* window, uint32_t codepoint, int mods, GLFWbool plain)
{
    if (codepoint < 32 || (codepoint > 126 && codepoint < 160))
        return;

    if (!window->lockKeyMods)
        mods &= ~(GLFW_MOD_CAPS_LOCK | GLFW_MOD_NUM_LOCK);

    if (window->callbacks.charmods)
        window->callbacks.charmods((GLFWwindow*) window, codepoint, mods);

    if (plain)
    {
        if (window->callbacks.character)
            window->callbacks.character((GLFWwindow*) window, codepoint);
    }
}

// Notifies shared code of a scroll event
//
void _glfwInputScroll(_GLFWwindow* window, double xoffset, double yoffset)
{
    if (window->callbacks.scroll)
        window->callbacks.scroll((GLFWwindow*) window, xoffset, yoffset);
}

// Notifies shared code of a mouse button click event
//
void _glfwInputMouseClick(_GLFWwindow* window, int button, int action, int mods)
{
    if (button < 0 || button > GLFW_MOUSE_BUTTON_LAST)
        return;

    if (!window->lockKeyMods)
        mods &= ~(GLFW_MOD_CAPS_LOCK | GLFW_MOD_NUM_LOCK);

    if (action == GLFW_RELEASE && window->stickyMouseButtons)
        window->mouseButtons[button] = _GLFW_STICK;
    else
        window->mouseButtons[button] = (char) action;

    if (window->callbacks.mouseButton)
        window->callbacks.mouseButton((GLFWwindow*) window, button, action, mods);
}

// Notifies shared code of a cursor motion event
// The position is specified in content area relative screen coordinates
//
void _glfwInputCursorPos(_GLFWwindow* window, double xpos, double ypos)
{
    if (window->virtualCursorPosX == xpos && window->virtualCursorPosY == ypos)
        return;

    window->virtualCursorPosX = xpos;
    window->virtualCursorPosY = ypos;

    if (window->callbacks.cursorPos)
        window->callbacks.cursorPos((GLFWwindow*) window, xpos, ypos);
}

// Notifies shared code of a cursor enter/leave event
//
void _glfwInputCursorEnter(_GLFWwindow* window, GLFWbool entered)
{
    if (window->callbacks.cursorEnter)
        window->callbacks.cursorEnter((GLFWwindow*) window, entered);
}

// Notifies shared code of files or directories dropped on a window
//
void _glfwInputDrop(_GLFWwindow* window, int count, const char** paths)
{
    if (window->callbacks.drop)
        window->callbacks.drop((GLFWwindow*) window, count, paths);
}


//////////////////////////////////////////////////////////////////////////
//////                       GLFW internal API                      //////
//////////////////////////////////////////////////////////////////////////

// Center the cursor in the content area of the specified window
//
void _glfwCenterCursorInContentArea(_GLFWwindow* window)
{
    int width, height;

    _glfwPlatformGetWindowSize(window, &width, &height);
    _glfwPlatformSetCursorPos(window, width / 2.0, height / 2.0);
}


//////////////////////////////////////////////////////////////////////////
//////                        GLFW public API                       //////
//////////////////////////////////////////////////////////////////////////

GLFWAPI int glfwGetInputMode(GLFWwindow* handle, int mode)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(0);

    switch (mode)
    {
        case GLFW_CURSOR:
            return window->cursorMode;
        case GLFW_STICKY_KEYS:
            return window->stickyKeys;
        case GLFW_STICKY_MOUSE_BUTTONS:
            return window->stickyMouseButtons;
        case GLFW_LOCK_KEY_MODS:
            return window->lockKeyMods;
        case GLFW_RAW_MOUSE_MOTION:
            return window->rawMouseMotion;
    }

    _glfwInputError(GLFW_INVALID_ENUM, "Invalid input mode 0x%08X", mode);
    return 0;
}

GLFWAPI void glfwSetInputMode(GLFWwindow* handle, int mode, int value)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT();

    if (mode == GLFW_CURSOR)
    {
        if (value != GLFW_CURSOR_NORMAL &&
            value != GLFW_CURSOR_HIDDEN &&
            value != GLFW_CURSOR_DISABLED)
        {
            _glfwInputError(GLFW_INVALID_ENUM,
                            "Invalid cursor mode 0x%08X",
                            value);
            return;
        }

        if (window->cursorMode == value)
            return;

        window->cursorMode = value;

        _glfwPlatformGetCursorPos(window,
                                  &window->virtualCursorPosX,
                                  &window->virtualCursorPosY);
        _glfwPlatformSetCursorMode(window, value);
    }
    else if (mode == GLFW_STICKY_KEYS)
    {
        value = value ? GLFW_TRUE : GLFW_FALSE;
        if (window->stickyKeys == value)
            return;

        if (!value)
        {
            int i;

            // Release all sticky keys
            for (i = 0;  i <= GLFW_KEY_LAST;  i++)
            {
                if (window->keys[i] == _GLFW_STICK)
                    window->keys[i] = GLFW_RELEASE;
            }
        }

        window->stickyKeys = value;
    }
    else if (mode == GLFW_STICKY_MOUSE_BUTTONS)
    {
        value = value ? GLFW_TRUE : GLFW_FALSE;
        if (window->stickyMouseButtons == value)
            return;

        if (!value)
        {
            int i;

            // Release all sticky mouse buttons
            for (i = 0;  i <= GLFW_MOUSE_BUTTON_LAST;  i++)
            {
                if (window->mouseButtons[i] == _GLFW_STICK)
                    window->mouseButtons[i] = GLFW_RELEASE;
            }
        }

        window->stickyMouseButtons = value;
    }
    else if (mode == GLFW_LOCK_KEY_MODS)
    {
        window->lockKeyMods = value ? GLFW_TRUE : GLFW_FALSE;
    }
    else if (mode == GLFW_RAW_MOUSE_MOTION)
    {
        if (!_glfwPlatformRawMouseMotionSupported())
        {
            _glfwInputError(GLFW_PLATFORM_ERROR,
                            "Raw mouse motion is not supported on this system");
            return;
        }

        value = value ? GLFW_TRUE : GLFW_FALSE;
        if (window->rawMouseMotion == value)
            return;

        window->rawMouseMotion = value;
        _glfwPlatformSetRawMouseMotion(window, value);
    }
    else
        _glfwInputError(GLFW_INVALID_ENUM, "Invalid input mode 0x%08X", mode);
}

GLFWAPI int glfwRawMouseMotionSupported(void)
{
    _GLFW_REQUIRE_INIT_OR_RETURN(GLFW_FALSE);
    return _glfwPlatformRawMouseMotionSupported();
}

GLFWAPI const char* glfwGetKeyName(int key, int scancode)
{
    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);

    if (key != GLFW_KEY_UNKNOWN)
    {
        if (key < GLFW_KEY_SPACE || key > GLFW_KEY_LAST)
        {
            _glfwInputError(GLFW_INVALID_ENUM, "Invalid key %i", key);
            return NULL;
        }

        if (key != GLFW_KEY_KP_EQUAL &&
            (key < GLFW_KEY_KP_0 || key > GLFW_KEY_KP_ADD) &&
            (key < GLFW_KEY_APOSTROPHE || key > GLFW_KEY_WORLD_2))
        {
            return NULL;
        }

        scancode = _glfwPlatformGetKeyScancode(key);
    }

    return _glfwPlatformGetScancodeName(scancode);
}

GLFWAPI int glfwGetKeyScancode(int key)
{
    _GLFW_REQUIRE_INIT_OR_RETURN(0);

    if (key < GLFW_KEY_SPACE || key > GLFW_KEY_LAST)
    {
        _glfwInputError(GLFW_INVALID_ENUM, "Invalid key %i", key);
        return -1;
    }

    return _glfwPlatformGetKeyScancode(key);
}

GLFWAPI int glfwGetKey(GLFWwindow* handle, int key)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(GLFW_RELEASE);

    if (key < GLFW_KEY_SPACE || key > GLFW_KEY_LAST)
    {
        _glfwInputError(GLFW_INVALID_ENUM, "Invalid key %i", key);
        return GLFW_RELEASE;
    }

    if (window->keys[key] == _GLFW_STICK)
    {
        // Sticky mode: release key now
        window->keys[key] = GLFW_RELEASE;
        return GLFW_PRESS;
    }

    return (int) window->keys[key];
}

GLFWAPI int glfwGetMouseButton(GLFWwindow* handle, int button)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(GLFW_RELEASE);

    if (button < GLFW_MOUSE_BUTTON_1 || button > GLFW_MOUSE_BUTTON_LAST)
    {
        _glfwInputError(GLFW_INVALID_ENUM, "Invalid mouse button %i", button);
        return GLFW_RELEASE;
    }

    if (window->mouseButtons[button] == _GLFW_STICK)
    {
        // Sticky mode: release mouse button now
        window->mouseButtons[button] = GLFW_RELEASE;
        return GLFW_PRESS;
    }

    return (int) window->mouseButtons[button];
}

GLFWAPI void glfwGetCursorPos(GLFWwindow* handle, double* xpos, double* ypos)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    if (xpos)
        *xpos = 0;
    if (ypos)
        *ypos = 0;

    _GLFW_REQUIRE_INIT();

    if (window->cursorMode == GLFW_CURSOR_DISABLED)
    {
        if (xpos)
            *xpos = window->virtualCursorPosX;
        if (ypos)
            *ypos = window->virtualCursorPosY;
    }
    else
        _glfwPlatformGetCursorPos(window, xpos, ypos);
}

GLFWAPI void glfwSetCursorPos(GLFWwindow* handle, double xpos, double ypos)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT();

    if (xpos != xpos || xpos < -DBL_MAX || xpos > DBL_MAX ||
        ypos != ypos || ypos < -DBL_MAX || ypos > DBL_MAX)
    {
        _glfwInputError(GLFW_INVALID_VALUE,
                        "Invalid cursor position %f %f",
                        xpos, ypos);
        return;
    }

    if (!_glfwPlatformWindowFocused(window))
        return;

    if (window->cursorMode == GLFW_CURSOR_DISABLED)
    {
        // Only update the accumulated position if the cursor is disabled
        window->virtualCursorPosX = xpos;
        window->virtualCursorPosY = ypos;
    }
    else
    {
        // Update system cursor position
        _glfwPlatformSetCursorPos(window, xpos, ypos);
    }
}

GLFWAPI GLFWcursor* glfwCreateCursor(const GLFWimage* image, int xhot, int yhot)
{
    _GLFWcursor* cursor;

    assert(image != NULL);
    assert(image->pixels != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);

    if (image->width <= 0 || image->height <= 0)
    {
        _glfwInputError(GLFW_INVALID_VALUE, "Invalid image dimensions for cursor");
        return NULL;
    }

    cursor = calloc(1, sizeof(_GLFWcursor));
    cursor->next = _glfw.cursorListHead;
    _glfw.cursorListHead = cursor;

    if (!_glfwPlatformCreateCursor(cursor, image, xhot, yhot))
    {
        glfwDestroyCursor((GLFWcursor*) cursor);
        return NULL;
    }

    return (GLFWcursor*) cursor;
}

GLFWAPI GLFWcursor* glfwCreateStandardCursor(int shape)
{
    _GLFWcursor* cursor;

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);

    if (shape != GLFW_ARROW_CURSOR &&
        shape != GLFW_IBEAM_CURSOR &&
        shape != GLFW_CROSSHAIR_CURSOR &&
        shape != GLFW_HAND_CURSOR &&
        shape != GLFW_HRESIZE_CURSOR &&
        shape != GLFW_VRESIZE_CURSOR &&
        shape != GLFW_RESIZE_NWSE_CURSOR &&
        shape != GLFW_RESIZE_NESW_CURSOR &&
        shape != GLFW_RESIZE_ALL_CURSOR &&
        shape != GLFW_NOT_ALLOWED_CURSOR)
    {
        _glfwInputError(GLFW_INVALID_ENUM, "Invalid standard cursor 0x%08X", shape);
        return NULL;
    }

    cursor = calloc(1, sizeof(_GLFWcursor));
    cursor->next = _glfw.cursorListHead;
    _glfw.cursorListHead = cursor;

    if (!_glfwPlatformCreateStandardCursor(cursor, shape))
    {
        glfwDestroyCursor((GLFWcursor*) cursor);
        return NULL;
    }

    return (GLFWcursor*) cursor;
}

GLFWAPI void glfwDestroyCursor(GLFWcursor* handle)
{
    _GLFWcursor* cursor = (_GLFWcursor*) handle;

    _GLFW_REQUIRE_INIT();

    if (cursor == NULL)
        return;

    // Make sure the cursor is not being used by any window
    {
        _GLFWwindow* window;

        for (window = _glfw.windowListHead;  window;  window = window->next)
        {
            if (window->cursor == cursor)
                glfwSetCursor((GLFWwindow*) window, NULL);
        }
    }

    _glfwPlatformDestroyCursor(cursor);

    // Unlink cursor from global linked list
    {
        _GLFWcursor** prev = &_glfw.cursorListHead;

        while (*prev != cursor)
            prev = &((*prev)->next);

        *prev = cursor->next;
    }

    free(cursor);
}

GLFWAPI void glfwSetCursor(GLFWwindow* windowHandle, GLFWcursor* cursorHandle)
{
    _GLFWwindow* window = (_GLFWwindow*) windowHandle;
    _GLFWcursor* cursor = (_GLFWcursor*) cursorHandle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT();

    window->cursor = cursor;

    _glfwPlatformSetCursor(window, cursor);
}

GLFWAPI GLFWkeyfun glfwSetKeyCallback(GLFWwindow* handle, GLFWkeyfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.key, cbfun);
    return cbfun;
}

GLFWAPI GLFWcharfun glfwSetCharCallback(GLFWwindow* handle, GLFWcharfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.character, cbfun);
    return cbfun;
}

GLFWAPI GLFWcharmodsfun glfwSetCharModsCallback(GLFWwindow* handle, GLFWcharmodsfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.charmods, cbfun);
    return cbfun;
}

GLFWAPI GLFWmousebuttonfun glfwSetMouseButtonCallback(GLFWwindow* handle,
                                                      GLFWmousebuttonfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.mouseButton, cbfun);
    return cbfun;
}

GLFWAPI GLFWcursorposfun glfwSetCursorPosCallback(GLFWwindow* handle,
                                                  GLFWcursorposfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.cursorPos, cbfun);
    return cbfun;
}

GLFWAPI GLFWcursorenterfun glfwSetCursorEnterCallback(GLFWwindow* handle,
                                                      GLFWcursorenterfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.cursorEnter, cbfun);
    return cbfun;
}

GLFWAPI GLFWscrollfun glfwSetScrollCallback(GLFWwindow* handle,
                                            GLFWscrollfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.scroll, cbfun);
    return cbfun;
}

GLFWAPI GLFWdropfun glfwSetDropCallback(GLFWwindow* handle, GLFWdropfun cbfun)
{
    _GLFWwindow* window = (_GLFWwindow*) handle;
    assert(window != NULL);

    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    _GLFW_SWAP_POINTERS(window->callbacks.drop, cbfun);
    return cbfun;
}

GLFWAPI void glfwSetClipboardString(GLFWwindow* handle, const char* string)
{
    assert(string != NULL);

    _GLFW_REQUIRE_INIT();
    _glfwPlatformSetClipboardString(string);
}

GLFWAPI const char* glfwGetClipboardString(GLFWwindow* handle)
{
    _GLFW_REQUIRE_INIT_OR_RETURN(NULL);
    return _glfwPlatformGetClipboardString();
}
