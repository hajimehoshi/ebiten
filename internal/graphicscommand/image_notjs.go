//go:build !js
// +build !js

package graphicscommand

import (
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
	// Screen image cannot be dumped.
	if i.screen {
		return nil
	}

	path = i.DumpName(path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := i.DumpTo(graphicsDriver, blackbg, rect, f); err != nil {
		return err
	}

	return nil
}
