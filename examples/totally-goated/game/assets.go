package game

import (
	"bytes"
	"image"
	_ "image/png"
	"io/fs"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

var assetsFS fs.FS

func InitAssets(fsys fs.FS) {
	assetsFS = fsys
	loadTileAssets()
	loadAudioAssets()
	loadBellAsset()
	loadPowerUpAssets()
}

func loadImageFromFS(path string) *ebiten.Image {
	data, err := fs.ReadFile(assetsFS, path)
	if err != nil {
		log.Fatalf("load image %s: %v", path, err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("decode image %s: %v", path, err)
	}
	return ebiten.NewImageFromImage(img)
}
