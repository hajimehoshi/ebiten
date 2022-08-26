//go:build !js
// +build !js

package graphicscommand

import (
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
	data, err := i.DumpBytes(graphicsDriver, blackbg, rect)
	if err != nil {
		return err
	}

	// screen image
	if data == nil {
		return nil
	}

	path = i.FormatPath(path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	return nil
}
