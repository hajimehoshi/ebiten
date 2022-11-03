# How to execute the examples

## Desktops

```sh
go run github.com/hajimehoshi/ebiten/v2/examples/rotate@latest
```

## Android

Install [gomobile](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) first.

```sh
gomobile install github.com/hajimehoshi/ebiten/v2/examples/rotate@latest
```

## iOS

```sh
gomobile build -target=ios -work github.com/hajimehoshi/ebiten/v2/examples/rotate@latest
```

Then, open the `WORK` directory, open `main.xcodeproj`, add signing, and run the project.
