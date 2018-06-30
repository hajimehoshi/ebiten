// Copyright 2018 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build example jsgo

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/internal/testflock"
)

var noErr = errors.New("clean")

type benchmark struct {
	name   string
	fn     func(b *testing.B, screen *ebiten.Image)
	result testing.BenchmarkResult
	output []string
}

const (
	MAX_TRIANGLES = 1000
	TRIANGLE_SIZE = 20
)

var (
	width  = 640
	height = 480
	// images to use during the benchmarks
	px26  *ebiten.Image
	px104 *ebiten.Image
	px416 *ebiten.Image
	box   *ebiten.Image

	benchIdx    = 0
	results     []testing.BenchmarkResult
	setup       sync.Once
	benchFunc   func(b *testing.B, screen *ebiten.Image)
	benchGo     = make(chan struct{})
	benchDone   = make(chan bool)
	vertices    = make([]ebiten.Vertex, 0, MAX_TRIANGLES*3)
	indices     = make([]uint16, 0, MAX_TRIANGLES*3)
	rainbow     = make([][3]float32, 120)
	rainbowBase = []color.RGBA{
		{240, 0, 0, 255},
		{240, 90, 0, 255},
		{220, 220, 0, 255},
		{0, 200, 0, 255},
		{0, 0, 255, 255},
		{180, 0, 200, 255},
	}
)

func main() {
	// You can't specify minimum bench time, or whether you want allocation,
	// directly in code, but because benchmarking per-frame is weird, we really
	// want to control those. So what if we faked up command line parameters
	// which have special meaning to the testing package?
	cpuprofile := flag.String("cpuprofile", "", "file to store CPU profile data in")
	newArgs := []string{os.Args[0], "-test.benchmem", "-test.v"}
	timeGiven := false
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-test.benchtime") {
			timeGiven = true
			break
		}
	}
	if !timeGiven {
		newArgs = append(newArgs, "-test.benchtime=240ms")
	}
	os.Args = append(newArgs, os.Args[1:]...)
	flag.Parse()
	testflock.Lock()
	defer testflock.Unlock()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalf("can't create %s: %s", *cpuprofile, err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	// log memory usage, too
	defer func() {
		f, err := os.Create("heap-profile.dat")
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't create heap-profile.dat: %s", err)
		} else {
			pprof.Lookup("heap").WriteTo(f, 0)
		}
		f, err = os.Create("alloc-profile.dat")
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't create alloc-profile.dat: %s", err)
		} else {
			pprof.Lookup("allocs").WriteTo(f, 0)
		}
	}()

	// Run an Ebiten process so that (*Image).At is available.
	f := func(screen *ebiten.Image) error {
		return runBenchmarks(screen)
	}
	if err := ebiten.Run(f, width, height, 1, "Test"); err != nil && err != noErr {
		panic(err)
	}
	for _, b := range benchList {
		fmt.Println(b.name)
		fmt.Println(b.result)
		fmt.Println(b.result.MemString())
		for i := 0; i < len(b.output); i++ {
			fmt.Println(b.output[i])
		}
	}
}

var benchList = []benchmark{
	{
		// the warmup function is because some allocations happen only during
		// initial setup, and can throw timing severely off.
		name: "warmup (ignore)",
		fn: func(b *testing.B, screen *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			for i := 0; i < b.N; i++ {
				op.GeoM.Reset()
				op.GeoM.Translate(float64(i*width/b.N), float64(i*height/b.N))
				screen.DrawImage(px26, op)
			}
		},
	},
	{
		name: "draw26",
		fn: func(b *testing.B, screen *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			for i := 0; i < b.N; i++ {
				op.GeoM.Reset()
				op.GeoM.Translate(float64(i*width/b.N), float64(i*height/b.N))
				screen.DrawImage(px26, op)
			}
		},
	},
	{
		name: "draw104",
		fn: func(b *testing.B, screen *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			for i := 0; i < b.N; i++ {
				op.GeoM.Reset()
				op.GeoM.Translate(float64(i*width/b.N), float64(i*height/b.N))
				screen.DrawImage(px104, op)
			}
		},
	},
	{
		name: "draw416",
		fn: func(b *testing.B, screen *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			for i := 0; i < b.N; i++ {
				op.GeoM.Reset()
				op.GeoM.Translate(float64(i*width/b.N), float64(i*height/b.N))
				screen.DrawImage(px416, op)
			}
		},
	},
	{
		name: "draw26color1",
		fn: func(b *testing.B, screen *ebiten.Image) {
			ops := []*ebiten.DrawImageOptions{&ebiten.DrawImageOptions{}, &ebiten.DrawImageOptions{}}
			ops[0].ColorM.Scale(1.0, 0.2, 0.2, 1.0)
			ops[1].ColorM.Scale(0.2, 1.0, 0.2, 1.0)
			for i := 0; i < b.N; i++ {
				idx := i
				ops[idx%2].GeoM.Reset()
				ops[idx%2].GeoM.Translate(float64(i*width/b.N), float64(i*height/b.N))
				screen.DrawImage(px26, ops[idx%2])
			}
		},
	},
	{
		name: "draw26color10",
		fn: func(b *testing.B, screen *ebiten.Image) {
			ops := []*ebiten.DrawImageOptions{&ebiten.DrawImageOptions{}, &ebiten.DrawImageOptions{}}
			ops[0].ColorM.Scale(1.0, 0.2, 0.2, 1.0)
			ops[1].ColorM.Scale(0.2, 1.0, 0.2, 1.0)
			for i := 0; i < b.N; i++ {
				idx := i / 10
				ops[idx%2].GeoM.Reset()
				ops[idx%2].GeoM.Translate(float64(i*width/b.N), float64(i*height/b.N))
				screen.DrawImage(px26, ops[idx%2])
			}
		},
	},
	{
		name: "triangles100",
		fn:   drawNTrianglesFunc(100),
	},
	{
		name: "triangles500",
		fn:   drawNTrianglesFunc(500),
	},
	{
		name: "triangles1000",
		fn:   drawNTrianglesFunc(1000),
	},
}

func interpolate(into [][3]float32, from, to color.RGBA) {
	r0, g0, b0 := int(from.R), int(from.G), int(from.B)
	r1, g1, b1 := int(to.R), int(to.G), int(to.B)
	n := len(into)

	for i := 0; i < n; i++ {
		inv := n - i
		r := (r0*inv + r1*i) / n
		g := (g0*inv + g1*i) / n
		b := (b0*inv + b1*i) / n

		into[i] = [3]float32{float32(r) / 255, float32(g) / 255, float32(b) / 255}
	}
}

func runBenchmarks(screen *ebiten.Image) error {
	var setupErr error

	// We need to run this exactly once, but it has to happen inside an
	// actual update loop so ebiten is fully up and running.
	setup.Do(func() {
		eimg, _, err := openEbitenImage()
		if err != nil {
			setupErr = fmt.Errorf("can't open image: %s", err)
			return
		}
		px26 = eimg
		box, err = makeWhiteBox()
		if err != nil {
			setupErr = fmt.Errorf("can't create white box: %s", err)
			return
		}

		w, h := eimg.Size()
		img, err := ebiten.NewImage(w*4, h*4, ebiten.FilterNearest)
		if err != nil {
			setupErr = fmt.Errorf("can't create new image: %s", err)
			return
		}
		px104 = img

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(4, 4)
		px104.DrawImage(px26, op)

		img, err = ebiten.NewImage(w*16, h*16, ebiten.FilterNearest)
		if err != nil {
			setupErr = fmt.Errorf("can't create new image: %s", err)
			return
		}
		px416 = img
		op.GeoM.Scale(4, 4)
		px416.DrawImage(px26, op)

		// initialize a pretty rainbow palette
		prev := rainbowBase[0]
		scale := len(rainbow) / len(rainbowBase)
		for idx, next := range rainbowBase[1:] {
			offset := idx * scale
			interpolate(rainbow[offset:offset+scale], prev, next)
			prev = next
		}
		interpolate(rainbow[(len(rainbowBase)-1)*scale:], prev, rainbowBase[0])
	})
	if setupErr != nil {
		return setupErr
	}

	if benchFunc != nil {
		benchGo <- struct{}{}
		if <-benchDone {
			benchFunc = nil
		}
	} else {
		if benchIdx >= len(benchList) {
			return noErr
		}
		benchFunc = benchList[benchIdx].fn
		go func(idx int) {
			benchList[idx].result = testing.Benchmark(func(b *testing.B) {
				// ebiten will start/stop the benchmark's timer during the
				// actual frame update, including time spent calling the user
				// update function, but also any rendering calls during the
				// update.
				b.StopTimer()
				// ensure that the change to the copy of the interface
				// inside ebiten happens during the next frame, not the
				// frame this goroutine started in.
				<-benchGo
				ebiten.SetBenchmark(b)
				benchDone <- false
				var elapsed time.Duration
				// run for at least 60 frames, so the FPS value should mean something
				for i := 0; i < 60; i++ {
					<-benchGo
					t0 := time.Now()
					benchFunc(b, screen)
					t1 := time.Now()
					elapsed += t1.Sub(t0)
					benchDone <- false
				}
				out := fmt.Sprintf("    N %5d, FPS: %5.2f, time %10v/60 frames", b.N, ebiten.CurrentFPS(), elapsed)
				benchList[idx].output = append(benchList[idx].output, out)
				<-benchGo
				ebiten.SetBenchmark(nil)
				benchDone <- false
			})
			<-benchGo
			benchDone <- true
		}(benchIdx)
		benchIdx++
	}
	return nil
}

func openEbitenImage() (*ebiten.Image, image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		return nil, nil, err
	}

	eimg, err := ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		return nil, nil, err
	}
	return eimg, img, nil
}

func makeWhiteBox() (*ebiten.Image, error) {
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 16, Y: 16}})
	col := color.RGBA{255, 255, 255, 255}
	for r := 0; r < 16; r++ {
		for c := 0; c < 16; c++ {
			img.Set(c, r, col)
		}
	}
	whiteBox, err := ebiten.NewImageFromImage(img, ebiten.FilterDefault)
	return whiteBox, err
}

func drawSomeTriangles(screen *ebiten.Image, count int) {
	triop := &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	points := []image.Point{
		{X: 0, Y: 0},
		{X: TRIANGLE_SIZE, Y: 0},
		{X: 0, Y: TRIANGLE_SIZE},
	}
	srcs := []image.Point{
		{X: 0, Y: 16},
		{X: 0, Y: 0},
		{X: 16, Y: 0},
	}
	for i := 0; i < count; i++ {
		vs := vertices[i*3 : (i+1)*3]
		rgb := rainbow[i%len(rainbow)]
		for j := 0; j < 3; j++ {
			vs[j] = ebiten.Vertex{
				DstX: float32(points[j].X), DstY: float32(points[j].Y),
				SrcX: float32(srcs[j].X), SrcY: float32(srcs[j].Y),
				ColorR: rgb[0], ColorG: rgb[1], ColorB: rgb[2],
				ColorA: 0.2,
			}
		}
		// after each point, we want to make a new point.
		switch i % 2 {
		case 0: // move X over
			// note order changes
			points[0] = points[2]
			points[2] = image.Point{points[0].X + TRIANGLE_SIZE, points[0].Y}
		case 1:
			points[0] = points[1]
			points[1] = image.Point{points[0].X + TRIANGLE_SIZE, points[0].Y}
		}
		if points[0].X >= width {
			points[0].X -= width
			points[1].X -= width
			points[2].X -= width
			points[0].Y += TRIANGLE_SIZE
			points[1].Y += TRIANGLE_SIZE
			points[2].Y += TRIANGLE_SIZE
		}
		// loop when we've filled the screen
		if points[0].Y >= height {
			points[0].Y -= height
			points[1].Y -= height
			points[2].Y -= height
		}
		vertices = append(vertices, vs...)
		indices = append(indices, uint16(i*3), uint16(i*3+1), uint16(i*3+2))
	}
	screen.DrawTriangles(vertices, indices, box, triop)
	vertices = vertices[:0]
	indices = indices[:0]
}

func drawNTrianglesFunc(count int) func(b *testing.B, screen *ebiten.Image) {
	return func(b *testing.B, screen *ebiten.Image) {
		for loop := 0; loop < b.N; loop++ {
			drawSomeTriangles(screen, count)
		}
	}
}
