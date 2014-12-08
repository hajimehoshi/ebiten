package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/example/blocks"
	"github.com/hajimehoshi/ebiten/ui"
	"github.com/hajimehoshi/ebiten/ui/glfw"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

func init() {
	runtime.LockOSThread()
}

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	u := new(glfw.UI)
	game := blocks.NewGame()
	if err := ui.Run(u, game, blocks.ScreenWidth, blocks.ScreenHeight, 2, "Blocks (Ebiten Demo)", 60); err != nil {
		log.Fatal(err)
	}
}
