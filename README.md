# Ebitengine (v2)

[![Go Reference](https://pkg.go.dev/badge/github.com/hajimehoshi/ebiten/v2.svg)](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
[![Build Status](https://github.com/hajimehoshi/ebiten/actions/workflows/test.yml/badge.svg)](https://github.com/hajimehoshi/ebiten/actions?query=workflow%3Atest)

**A dead simple 2D game engine for Go**

Ebitengine (formerly known as Ebiten) is an open source game engine for the Go programming language. Ebitengine's simple API allows you to quickly and easily develop 2D games that can be deployed across multiple platforms.

* [Website (ebitengine.org)](https://ebitengine.org)
* [API Reference](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
* [Cheat Sheet](https://ebitengine.org/en/documents/cheatsheet.html)
* [Awesome Ebitengine](https://github.com/sedyh/awesome-ebitengine)

![Overview](https://ebitengine.org/images/overview2.png)

## Platforms

* [Windows](https://ebitengine.org/en/documents/install.html?os=windows) (No Cgo required!)
* [macOS](https://ebitengine.org/en/documents/install.html?os=darwin)
* [Linux](https://ebitengine.org/en/documents/install.html?os=linux)
* [FreeBSD](https://ebitengine.org/en/documents/install.html?os=freebsd)
* [Android](https://ebitengine.org/en/documents/mobile.html)
* [iOS](https://ebitengine.org/en/documents/mobile.html)
* [WebAssembly](https://ebitengine.org/en/documents/webassembly.html)
* Nintendo Switch
* Xbox (Xbox support is limited and not available to everyone. Negotiations are currently underway to make it accessible to all.)

For installation on desktops, see [the installation instruction](https://ebitengine.org/en/documents/install.html).

## Features

* 2D Graphics (Geometry and color transformation by matrices, Various composition modes, Offscreen rendering, Text rendering, Automatic batches, Automatic texture atlas, Custom shaders)
* Input (Mouse, Keyboard, Gamepads, Touches)
* Audio (Ogg/Vorbis, MP3, WAV, PCM)

## Packages

* [ebiten](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
  * [audio](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio)
    * [mp3](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/mp3)
    * [vorbis](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/vorbis)
    * [wav](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/wav)
  * [colorm](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/colorm)
  * [ebitenutil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/ebitenutil)
  * [inpututil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/inpututil)
  * [mobile](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/mobile)
  * [text/v2](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/text/v2)
  * [vector](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/vector)
  * [exp/textinput](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/exp/textinput)

## Community

- [Discord](https://discord.gg/3tVdM5H8cC)
- `#ebitengine` channel in [Gophers Slack](https://blog.gopheracademy.com/gophers-slack-community/)
- [GitHub Discussion](https://github.com/hajimehoshi/ebiten/discussions)
- [`r/ebitengine` in Reddit](https://www.reddit.com/r/ebitengine/)

## License

Ebitengine is licensed under Apache license version 2.0. See [LICENSE](LICENSE) file.

[The Ebitengine logo](https://ebitengine.org/images/logo.png) by Hajime Hoshi is licensed under [the Creative Commons Attribution 4.0](https://creativecommons.org/licenses/by/4.0/).

### GLFW

https://github.com/glfw/glfw


```
Copyright (c) 2002-2006 Marcus Geelnard

Copyright (c) 2006-2019 Camilla Löwy

This software is provided 'as-is', without any express or implied
warranty. In no event will the authors be held liable for any damages
arising from the use of this software.

Permission is granted to anyone to use this software for any purpose,
including commercial applications, and to alter it and redistribute it
freely, subject to the following restrictions:

1. The origin of this software must not be misrepresented; you must not
   claim that you wrote the original software. If you use this software
   in a product, an acknowledgment in the product documentation would
   be appreciated but is not required.

2. Altered source versions must be plainly marked as such, and must not
   be misrepresented as being the original software.

3. This notice may not be removed or altered from any source
   distribution.
```

### Go

https://cs.opensource.google/go/go


```
Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### go-gl/gl

https://github.com/go-gl/gl


```
The MIT License (MIT)

Copyright (c) 2014 Eric Woroshow

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### go-gl/glfw

https://github.com/go-gl/glfw


```
Copyright (c) 2012 The glfw3-go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
