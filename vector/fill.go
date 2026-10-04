// Copyright 2025 The Ebitengine Authors
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

package vector

import (
	"fmt"
	"image"
	"runtime"
	"slices"
	"sync"
	"weak"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/imagebridge"
)

// FillRule is the rule whether an overlapped region is rendered or not.
//
// The number of overlaps is counted with a limited precision, so a region with too many
// overlapping triangles can be rendered incorrectly.
type FillRule int

const (
	// FillRuleNonZero means that triangles are rendered based on the non-zero rule.
	// If and only if the number of overlaps is not 0, the region is rendered.
	FillRuleNonZero FillRule = iota

	// FillRuleEvenOdd means that triangles are rendered based on the even-odd rule.
	// If and only if the number of overlaps is odd, the region is rendered.
	FillRuleEvenOdd
)

var (
	// theCallbackTokens and theFillPathsStates are keyed by weak pointers not to keep the destination images alive.
	// When a destination image is collected before being used again, releaseFillPathsState removes the entries.
	theCallbackTokens      = map[weak.Pointer[ebiten.Image]]int64{}
	theFillPathsStates     = map[weak.Pointer[ebiten.Image]]*fillPathsState{}
	theFillPathsStatesPool = sync.Pool{
		New: func() any {
			return &fillPathsState{}
		},
	}
	theFillPathM sync.Mutex

	theImageBridge = imagebridge.Get[*ebiten.Image]()
)

// FillOptions is options to fill a path.
type FillOptions struct {
	// FillRule is the rule whether an overlapped region is rendered or not.
	// The default (zero) value is FillRuleNonZero.
	FillRule FillRule
}

// DrawPathOptions is options to draw a path.
type DrawPathOptions struct {
	// AntiAlias is whether the path is drawn with anti-aliasing.
	// The default (zero) value is false.
	AntiAlias bool

	// Clip restricts the drawing to the given region.
	// The default (zero) value is nil, which imposes no restriction.
	// Paths, nodes, and slices may be modified after the drawing call returns.
	// For a nil drawing path or one without sub-paths, [DrawPathOptions.Clip] is ignored.
	// Otherwise, typed nil clip pointers, invalid [ClipSet.Operation] or [PathClip.FillRule]
	// values, and cyclic clip definitions cause a panic.
	Clip Clip

	// ColorScale is the color scale to apply to the path.
	// The default (zero) value is identity, which is (1, 1, 1, 1) (white).
	ColorScale ebiten.ColorScale

	// Blend is the blend mode to apply to the path.
	// Pixels the drawing does not cover are left unchanged for every blend mode.
	// The default (zero) value is ebiten.BlendSourceOver.
	Blend ebiten.Blend
}

// FillPath fills the specified path with the specified options.
//
// An invalid FillRule in fillOptions causes a panic.
func FillPath(dst *ebiten.Image, path *Path, fillOptions *FillOptions, drawPathOptions *DrawPathOptions) {
	if path == nil || len(path.subPaths) == 0 {
		return
	}

	if drawPathOptions == nil {
		drawPathOptions = &DrawPathOptions{}
	}
	if fillOptions == nil {
		fillOptions = &FillOptions{}
	}

	var hasEmptyClip bool
	var clipState *clipPreparationState
	if drawPathOptions.Clip != nil {
		clipState = theClipPreparationStatesPool.Get().(*clipPreparationState)
		defer clipState.release()
		validateClip(drawPathOptions.Clip, clipState.visitStates)
		hasEmptyClip = isEmptyClip(drawPathOptions.Clip, clipState.emptyClips)
	}

	bounds := dst.Bounds()
	if hasEmptyClip {
		switch fillOptions.FillRule {
		case FillRuleNonZero, FillRuleEvenOdd:
			return
		}
	}

	// Get the original image if dst is a sub-image to integrate the callbacks.
	dst = theImageBridge.OriginalImage(dst)

	theFillPathM.Lock()
	defer theFillPathM.Unlock()

	key := weak.Make(dst)

	// Remove the previous registered callbacks.
	if token, ok := theCallbackTokens[key]; ok {
		theImageBridge.RemoveUsage(dst, token)
	}
	delete(theCallbackTokens, key)

	s, ok := theFillPathsStates[key]
	if !ok {
		s = theFillPathsStatesPool.Get().(*fillPathsState)
		theFillPathsStates[key] = s
		s.cleanup = runtime.AddCleanup(dst, releaseFillPathsState, key)
	}
	if s.antialias != drawPathOptions.AntiAlias || s.blend != drawPathOptions.Blend || s.fillRule != fillOptions.FillRule {
		s.fillPaths(dst)
		s.reset()
	}
	s.antialias = drawPathOptions.AntiAlias
	s.blend = drawPathOptions.Blend
	s.fillRule = fillOptions.FillRule
	idx := s.addPath(path, bounds, drawPathOptions.ColorScale)
	s.drawPathIndices = append(s.drawPathIndices, idx)
	if drawPathOptions.Clip != nil {
		s.addClip(idx, path, drawPathOptions.Clip, bounds, clipState.pathIndices, clipState.visitStates)
	}

	// Use an independent callback function to avoid unexpected captures.
	theCallbackTokens[key] = theImageBridge.AddUsage(dst, fillPathCallback)
}

func fillPathCallback(dst *ebiten.Image) {
	if theImageBridge.OriginalImage(dst) != dst {
		panic("vector: dst must be the original image")
	}

	theFillPathM.Lock()
	defer theFillPathM.Unlock()

	key := weak.Make(dst)

	// Remove the callback not to call this twice.
	if token, ok := theCallbackTokens[key]; ok {
		theImageBridge.RemoveUsage(dst, token)
	}
	delete(theCallbackTokens, key)

	s, ok := theFillPathsStates[key]
	if !ok {
		panic("vector: fillPathsState must exist here")
	}
	s.fillPaths(dst)
	s.reset()
	delete(theFillPathsStates, key)
	s.cleanup.Stop()
	s.cleanup = runtime.Cleanup{}
	theFillPathsStatesPool.Put(s)
}

// releaseFillPathsState discards the state for a destination image that was collected before being used again.
func releaseFillPathsState(key weak.Pointer[ebiten.Image]) {
	theFillPathM.Lock()
	defer theFillPathM.Unlock()

	// The destination image is already collected, and its usage callbacks are gone with it.
	delete(theCallbackTokens, key)

	s, ok := theFillPathsStates[key]
	if !ok {
		return
	}
	delete(theFillPathsStates, key)
	s.cleanup = runtime.Cleanup{}
	s.reset()
	theFillPathsStatesPool.Put(s)
}

type offsetAndColor struct {
	offsetX    float32
	offsetY    float32
	colorR     float32
	colorG     float32
	colorB     float32
	colorA     float32
	imageIndex int
}

var (
	offsetAndColorsNonAA = []offsetAndColor{
		{
			offsetX: 0,
			offsetY: 0,
			colorR:  1,
			colorG:  0,
			colorB:  0,
			colorA:  0,
		},
	}

	// https://learn.microsoft.com/en-us/windows/win32/api/d3d11/ne-d3d11-d3d11_standard_multisample_quality_levels
	offsetAndColorsAA = []offsetAndColor{
		{
			offsetX:    1.0 / 16.0,
			offsetY:    -3.0 / 16.0,
			colorR:     1,
			colorG:     0,
			colorB:     0,
			colorA:     0,
			imageIndex: 0,
		},
		{
			offsetX:    -1.0 / 16.0,
			offsetY:    3.0 / 16.0,
			colorR:     0,
			colorG:     1,
			colorB:     0,
			colorA:     0,
			imageIndex: 0,
		},
		{
			offsetX:    5.0 / 16.0,
			offsetY:    1.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     1,
			colorA:     0,
			imageIndex: 0,
		},
		{
			offsetX:    -3.0 / 16.0,
			offsetY:    -5.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     0,
			colorA:     1,
			imageIndex: 0,
		},
		{
			offsetX:    -5.0 / 16.0,
			offsetY:    5.0 / 16.0,
			colorR:     1,
			colorG:     0,
			colorB:     0,
			colorA:     0,
			imageIndex: 1,
		},
		{
			offsetX:    -7.0 / 16.0,
			offsetY:    -1.0 / 16.0,
			colorR:     0,
			colorG:     1,
			colorB:     0,
			colorA:     0,
			imageIndex: 1,
		},
		{
			offsetX:    3.0 / 16.0,
			offsetY:    7.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     1,
			colorA:     0,
			imageIndex: 1,
		},
		{
			offsetX:    7.0 / 16.0,
			offsetY:    -7.0 / 16.0,
			colorR:     0,
			colorG:     0,
			colorB:     0,
			colorA:     1,
			imageIndex: 1,
		},
	}
)

// theAtlas manages the atlas for stencil buffer images.
// theAtlas is a singleton to avoid unnecessary texture allocations.
//
// theAtlas methods are used only at fillPathsState.fillPaths, and should be protected by theFillPathM.
var theAtlas atlas

type fillPathsState struct {
	drawPathIndices         []int
	drawPathIndexToClipPlan map[int]*clipPlan

	// paths contains both drawing paths and clip paths.
	paths  []*Path
	colors []ebiten.ColorScale
	bounds []image.Rectangle

	vertices []ebiten.Vertex
	indices  []uint32

	antialias bool
	blend     ebiten.Blend
	fillRule  FillRule

	// cleanup removes the entries for the destination image from theCallbackTokens and theFillPathsStates
	// when the image is collected.
	cleanup runtime.Cleanup
}

func (f *fillPathsState) reset() {
	f.drawPathIndices = f.drawPathIndices[:0]
	for _, plan := range f.drawPathIndexToClipPlan {
		plan.release()
	}
	clear(f.drawPathIndexToClipPlan)
	for _, p := range f.paths {
		p.Reset()
	}
	f.paths = f.paths[:0]
	f.bounds = f.bounds[:0]
	f.colors = slices.Delete(f.colors, 0, len(f.colors))
}

const invalidPathIndex = -1

// addPath adds a snapshot of path and returns its index.
// A nil path returns invalidPathIndex without adding a path.
func (f *fillPathsState) addPath(path *Path, bounds image.Rectangle, clr ebiten.ColorScale) int {
	if path == nil {
		return invalidPathIndex
	}

	idx := len(f.paths)
	f.paths = slices.Grow(f.paths, 1)[:idx+1]
	if f.paths[idx] == nil {
		f.paths[idx] = &Path{}
	}
	dst := f.paths[idx]
	dst.addSubPaths(len(path.subPaths))
	for i, subPath := range path.subPaths {
		dst.subPaths[i].start = subPath.start
		dst.subPaths[i].closed = subPath.closed
		dst.subPaths[i].invalid = subPath.invalid
		dst.subPaths[i].ops = slices.Grow(dst.subPaths[i].ops, len(subPath.ops))[:len(subPath.ops)]
		copy(dst.subPaths[i].ops, subPath.ops)
	}
	f.bounds = append(f.bounds, bounds)
	f.colors = append(f.colors, clr)
	return idx
}

// fillPaths renders the paths added by addPath with their colors onto dst.
//
// fillPaths callers must be protected by theFillPathM.
func (f *fillPathsState) fillPaths(dst *ebiten.Image) {
	if len(f.paths) != len(f.colors) {
		panic("vector: the number of paths and colors must be the same")
	}

	vs := f.vertices[:0]
	is := f.indices[:0]
	defer func() {
		f.vertices = vs
		f.indices = is
	}()

	theAtlas.setPaths(dst.Bounds(), f.paths, f.bounds, f.antialias)
	var hasClip bool
	for i, plan := range f.drawPathIndexToClipPlan {
		plan.discardMissingStencils(f.antialias)
		hasClip = hasClip || plan.kind != clipBatchKindEmpty && theAtlas.stencilBufferImageAt(i, f.antialias, 0) != nil
	}

	offsetAndColors := offsetAndColorsNonAA
	if f.antialias {
		offsetAndColors = offsetAndColorsAA
	}

	// First, render the polygons roughly.
	for i, path := range f.paths {
		if path == nil {
			continue
		}

		for _, oac := range offsetAndColors {
			vs = vs[:0]
			is = is[:0]

			stencilBufferImage := theAtlas.stencilBufferImageAt(i, f.antialias, oac.imageIndex)
			if stencilBufferImage == nil {
				continue
			}
			pp := theAtlas.pathRenderingPositionAt(i)
			dstOffsetX := float64(-pp.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-pp.X))
			dstOffsetY := float64(-pp.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-pp.Y))

			offsetX := float64(oac.offsetX) + dstOffsetX
			offsetY := float64(oac.offsetY) + dstOffsetY
			for i := range path.subPaths {
				subPath := &path.subPaths[i]
				if !subPath.isValid() {
					continue
				}

				// Add an origin point. Any position works in theory.
				// Use the sub-path's start point. Using one of the sub-path's points can reduce triangles.
				// Also, this point should be close to the other points and then triangle overlaps are reduced.
				// TODO: Use a better position like the center of the sub-path.
				originIdx := uint32(len(vs))
				cur := subPath.start
				vs = append(vs, ebiten.Vertex{
					DstX:   float32(cur.x + offsetX),
					DstY:   float32(cur.y + offsetY),
					ColorR: oac.colorR,
					ColorG: oac.colorG,
					ColorB: oac.colorB,
					ColorA: oac.colorA,
				})

				for _, op := range subPath.ops {
					switch op.typ {
					case opTypeLineTo:
						idx := uint32(len(vs))
						vs = append(vs,
							ebiten.Vertex{
								DstX:   float32(cur.x + offsetX),
								DstY:   float32(cur.y + offsetY),
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							},
							ebiten.Vertex{
								DstX:   float32(op.p1.x + offsetX),
								DstY:   float32(op.p1.y + offsetY),
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							})
						is = append(is, idx, originIdx, idx+1)
						cur = op.p1
					case opTypeQuadTo:
						idx := uint32(len(vs))
						vs = append(vs,
							ebiten.Vertex{
								DstX:   float32(cur.x + offsetX),
								DstY:   float32(cur.y + offsetY),
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							},
							ebiten.Vertex{
								DstX:   float32(op.p2.x + offsetX),
								DstY:   float32(op.p2.y + offsetY),
								ColorR: oac.colorR,
								ColorG: oac.colorG,
								ColorB: oac.colorB,
								ColorA: oac.colorA,
							})
						is = append(is, idx, originIdx, idx+1)
						cur = op.p2
					}
				}
				// If the sub-path is not closed, add a supplementary line.
				if !subPath.closed {
					idx := uint32(len(vs))
					vs = append(vs,
						ebiten.Vertex{
							DstX:   float32(cur.x + offsetX),
							DstY:   float32(cur.y + offsetY),
							ColorR: oac.colorR,
							ColorG: oac.colorG,
							ColorB: oac.colorB,
							ColorA: oac.colorA,
						},
						ebiten.Vertex{
							DstX:   float32(subPath.start.x + offsetX),
							DstY:   float32(subPath.start.y + offsetY),
							ColorR: oac.colorR,
							ColorG: oac.colorG,
							ColorB: oac.colorB,
							ColorA: oac.colorA,
						})
					is = append(is, idx, originIdx, idx+1)
				}
			}
			op := &ebiten.DrawTrianglesShaderOptions{}
			op.Blend = ebiten.BlendLighter
			shader, err := ensureStencilBufferShaders()
			if err != nil {
				panic(fmt.Sprintf("vector: failed to create stencil buffer shader: %v", err))
			}
			stencilBufferImage.DrawTrianglesShader32(vs, is, shader, op)
		}
	}

	// Second, render the bezier curves.
	for i, path := range f.paths {
		if path == nil {
			continue
		}

		for _, oac := range offsetAndColors {
			vs = vs[:0]
			is = is[:0]

			stencilBufferImage := theAtlas.stencilBufferImageAt(i, f.antialias, oac.imageIndex)
			if stencilBufferImage == nil {
				continue
			}
			pp := theAtlas.pathRenderingPositionAt(i)
			dstOffsetX := float64(-pp.X + stencilBufferImage.Bounds().Min.X - max(0, dst.Bounds().Min.X-pp.X))
			dstOffsetY := float64(-pp.Y + stencilBufferImage.Bounds().Min.Y - max(0, dst.Bounds().Min.Y-pp.Y))
			offsetX := float64(oac.offsetX) + dstOffsetX
			offsetY := float64(oac.offsetY) + dstOffsetY
			for i := range path.subPaths {
				subPath := &path.subPaths[i]
				if !subPath.isValid() {
					continue
				}

				cur := subPath.start
				for _, op := range subPath.ops {
					switch op.typ {
					case opTypeLineTo:
						cur = op.p1
					case opTypeQuadTo:
						idx := uint32(len(vs))
						vs = append(vs,
							ebiten.Vertex{
								DstX:    float32(cur.x + offsetX),
								DstY:    float32(cur.y + offsetY),
								ColorR:  oac.colorR,
								ColorG:  oac.colorG,
								ColorB:  oac.colorB,
								ColorA:  oac.colorA,
								Custom0: 0, // u for Loop-Blinn algorithm
								Custom1: 0, // v for Loop-Blinn algorithm
							},
							ebiten.Vertex{
								DstX:    float32(op.p1.x + offsetX),
								DstY:    float32(op.p1.y + offsetY),
								ColorR:  oac.colorR,
								ColorG:  oac.colorG,
								ColorB:  oac.colorB,
								ColorA:  oac.colorA,
								Custom0: 0.5,
								Custom1: 0,
							},
							ebiten.Vertex{
								DstX:    float32(op.p2.x + offsetX),
								DstY:    float32(op.p2.y + offsetY),
								ColorR:  oac.colorR,
								ColorG:  oac.colorG,
								ColorB:  oac.colorB,
								ColorA:  oac.colorA,
								Custom0: 1,
								Custom1: 1,
							})
						is = append(is, idx, idx+1, idx+2)
						cur = op.p2
					}
				}
			}
			op := &ebiten.DrawTrianglesShaderOptions{}
			op.Blend = ebiten.BlendLighter
			shader, err := ensureStencilBufferBezierShader()
			if err != nil {
				panic(fmt.Sprintf("vector: failed to create stencil buffer bezier shader: %v", err))
			}
			stencilBufferImage.DrawTrianglesShader32(vs, is, shader, op)
		}
	}

	var clippedImages []*ebiten.Image
	var maskUnclipped bool
	if hasClip {
		clippedImages = make([]*ebiten.Image, len(f.drawPathIndices))
		maskUnclipped = f.canMaskUnclippedPaths()
	}

	// Render the stencil buffer with the specified color.
	var clipChunkEnd int
	for j, i := range f.drawPathIndices {
		if len(clippedImages) > 0 && j == clipChunkEnd {
			clipChunkEnd = f.prepareClipImages(clippedImages, j, maskUnclipped)
		}
		path := f.paths[i]
		if path == nil {
			continue
		}

		stencilImage := theAtlas.stencilBufferImageAt(i, f.antialias, 0)
		if stencilImage == nil {
			continue
		}
		if plan := f.drawPathIndexToClipPlan[i]; plan != nil && plan.kind == clipBatchKindEmpty {
			switch f.fillRule {
			case FillRuleNonZero, FillRuleEvenOdd:
				continue
			default:
				panic(fmt.Sprintf("vector: invalid fill rule: %d", f.fillRule))
			}
		}
		var clipped *ebiten.Image
		if len(clippedImages) > 0 {
			clipped = clippedImages[j]
			if clipped != nil {
				stencilImage = clipped
			}
		}
		srcRegion := stencilImage.Bounds()
		if clipped != nil && f.antialias {
			srcRegion.Max.X = srcRegion.Min.X + srcRegion.Dx()/clipSamplePlanesAA
		}

		var offsetX, offsetY float32
		if clipped != nil && f.antialias {
			offsetX = float32(srcRegion.Dx())
		} else if f.antialias {
			stencilImage1 := theAtlas.stencilBufferImageAt(i, f.antialias, 1)
			offsetX = float32(stencilImage1.Bounds().Min.X - stencilImage.Bounds().Min.X)
			offsetY = float32(stencilImage1.Bounds().Min.Y - stencilImage.Bounds().Min.Y)
		}

		pp := theAtlas.pathRenderingPositionAt(i)

		vs = vs[:0]
		is = is[:0]
		dstOffsetX := max(0, dst.Bounds().Min.X-pp.X)
		dstOffsetY := max(0, dst.Bounds().Min.Y-pp.Y)
		var clrR, clrG, clrB, clrA float32
		clrR = f.colors[i].R()
		clrG = f.colors[i].G()
		clrB = f.colors[i].B()
		clrA = f.colors[i].A()
		vs = append(vs,
			ebiten.Vertex{
				DstX:    float32(pp.X + dstOffsetX),
				DstY:    float32(pp.Y + dstOffsetY),
				SrcX:    float32(srcRegion.Min.X),
				SrcY:    float32(srcRegion.Min.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			},
			ebiten.Vertex{
				DstX:    float32(pp.X + srcRegion.Dx() + dstOffsetX),
				DstY:    float32(pp.Y + dstOffsetY),
				SrcX:    float32(srcRegion.Max.X),
				SrcY:    float32(srcRegion.Min.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			},
			ebiten.Vertex{
				DstX:    float32(pp.X + dstOffsetX),
				DstY:    float32(pp.Y + srcRegion.Dy() + dstOffsetY),
				SrcX:    float32(srcRegion.Min.X),
				SrcY:    float32(srcRegion.Max.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			},
			ebiten.Vertex{
				DstX:    float32(pp.X + srcRegion.Dx() + dstOffsetX),
				DstY:    float32(pp.Y + srcRegion.Dy() + dstOffsetY),
				SrcX:    float32(srcRegion.Max.X),
				SrcY:    float32(srcRegion.Max.Y),
				ColorR:  clrR,
				ColorG:  clrG,
				ColorB:  clrB,
				ColorA:  clrA,
				Custom0: offsetX,
				Custom1: offsetY,
			})
		is = append(is, 0, 1, 2, 1, 2, 3)

		op := &ebiten.DrawTrianglesShaderOptions{}
		op.Blend = f.blend
		op.Images[0] = stencilImage
		var shader *ebiten.Shader
		switch f.fillRule {
		case FillRuleNonZero:
			var err error
			shader, err = ensureStencilBufferNonZeroShader(f.antialias)
			if err != nil {
				panic(fmt.Sprintf("vector: failed to create stencil buffer non-zero shader: %v", err))
			}
		case FillRuleEvenOdd:
			var err error
			shader, err = ensureStencilBufferEvenOddShader(f.antialias)
			if err != nil {
				panic(fmt.Sprintf("vector: failed to create stencil buffer even-odd shader: %v", err))
			}
		default:
			panic(fmt.Sprintf("vector: invalid fill rule: %d", f.fillRule))
		}
		if clipped != nil {
			shader = ensureClipResolveShader(f.antialias)
		}
		dst2 := dst
		var recycle bool
		if dst.Bounds() != f.bounds[i] {
			dst2 = dst.RecyclableSubImage(f.bounds[i])
			recycle = true
		}
		dst2.DrawTrianglesShader32(vs, is, shader, op)
		if recycle {
			dst2.Recycle()
		}
		if clipped != nil {
			theClipImages.put(clipped, f.drawPathIndexToClipPlan[i] == nil || f.drawPathIndexToClipPlan[i].kind != clipBatchKindNone)
			clippedImages[j] = nil
		}
	}
}
