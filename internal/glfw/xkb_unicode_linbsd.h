// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2014 Jonas Ådahl <jadahl@gmail.com>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

#include <stdint.h>

#define GLFW_INVALID_CODEPOINT 0xffffffffu

uint32_t _glfwKeySym2Unicode(unsigned int keysym);
