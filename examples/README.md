# How to execute the examples

## Desktops

```sh
go run -tags=example github.com/hajimehoshi/ebiten/examples/rotate
```

## Android

Install [gomobile](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) first.

```sh
gomobile install -tags=example github.com/hajimehoshi/ebiten/examples/rotate
```

## iOS

```sh
gomobile build -target=ios -tags=example -work github.com/hajimehoshi/ebiten/examples/rotate
```

Then, open the `WORK` directory, open `main.xcodeproj`, add signing, and run the project.
