# Ebiten (海老天) v1.0.0-alpha

* A simple 2D game library in Go
* Works on
  * Mac OS X
  * Linux (maybe)
  * Windows (possibly)
* [API Docs](http://godoc.org/github.com/hajimehoshi/ebiten)

## Features

* 2D Graphics
* Input (Mouse, Keyboard)

## Example

* example/mosaic - Mosaics an image
* example/perspective - See an image in a perspective view
* example/rotate - Rotates an image
* etc.

## Install on Mac OS X

```
:; brew install glew
:; brew install glfw3 # or homebrew/versions/glfw3
:; go get -u github.com/hajimehoshi/ebiten
```

## Execute the example

```
:; cd $GOHOME/src/github.com/hajimehoshi/ebiten/example
:; go run blocks/main.go
```

### Benchmark the example

```
:; cd $GOHOME/src/github.com/hajimehoshi/ebiten/example
:; go build -o=example blocks/main.go
:; ./example -cpuprofile=cpu.out
:; go tool pprof ./example cpu.out
```

## Versioning

* We adopted [Semantic Versioning](http://semver.org/)

## License

```
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
