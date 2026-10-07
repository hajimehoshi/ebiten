// Copyright 2026 The Ebitengine Authors
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
	"cmp"
	"fmt"
	"image"
	"image/color"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

// ClipOperation specifies how a [ClipSet] combines its regions.
type ClipOperation int

const (
	// ClipOperationUnion keeps points inside any operand.
	ClipOperationUnion ClipOperation = iota

	// ClipOperationIntersection keeps points inside every operand.
	ClipOperationIntersection
)

// Clip defines a clipping region. Pointers to [PathClip] and [ClipSet] implement [Clip].
// [Clip] values may be shared and reused.
type Clip interface {
	isClip()
}

// PathClip defines a region by filling a [Path].
// The zero value contains no points. A nil [PathClip] pointer causes a panic.
type PathClip struct {
	// Path is in destination coordinates. A nil path contains no points.
	Path *Path

	// FillRule determines which points are inside [Path].
	// The default (zero) value is [FillRuleNonZero].
	// Values other than [FillRuleNonZero] and [FillRuleEvenOdd] cause a panic.
	FillRule FillRule
}

func (*PathClip) isClip() {}

// ClipSet combines the regions defined by its [ClipSet.Children].
// The zero value is an empty union. A nil [ClipSet] pointer causes a panic.
type ClipSet struct {
	// Operation combines [ClipSet.Children]. An intersection without children contains every point.
	// The default (zero) value is [ClipOperationUnion].
	// Values other than [ClipOperationUnion] and [ClipOperationIntersection] cause a panic.
	Operation ClipOperation

	// Children specifies the regions to combine. A nil child contains no points.
	// The same clip may appear multiple times.
	// A set that directly or indirectly contains itself causes a panic.
	Children []Clip
}

func (*ClipSet) isClip() {}

type clipVisitState uint8

const (
	clipVisitStateUnvisited clipVisitState = iota
	clipVisitStateVisiting
	clipVisitStateVisited
)

type clipPreparationState struct {
	visitStates map[Clip]clipVisitState
	emptyClips  map[Clip]bool
	pathIndices map[*Path]int
}

var theClipPreparationStatesPool = sync.Pool{
	New: func() any {
		return &clipPreparationState{
			visitStates: map[Clip]clipVisitState{},
			emptyClips:  map[Clip]bool{},
			pathIndices: map[*Path]int{},
		}
	},
}

func (s *clipPreparationState) release() {
	clear(s.visitStates)
	clear(s.emptyClips)
	clear(s.pathIndices)
	theClipPreparationStatesPool.Put(s)
}

func validateClip(clip Clip, visitStates map[Clip]clipVisitState) {
	if clip == nil {
		return
	}
	var children []Clip
	switch clip := clip.(type) {
	case *PathClip:
		if clip == nil {
			panic("vector: nil PathClip")
		}
		switch clip.FillRule {
		case FillRuleNonZero, FillRuleEvenOdd:
		default:
			panic(fmt.Sprintf("vector: invalid fill rule: %d", clip.FillRule))
		}
	case *ClipSet:
		if clip == nil {
			panic("vector: nil ClipSet")
		}
		switch clip.Operation {
		case ClipOperationUnion, ClipOperationIntersection:
		default:
			panic(fmt.Sprintf("vector: invalid clip operation: %d", clip.Operation))
		}
		children = clip.Children
	default:
		panic(fmt.Sprintf("vector: invalid clip type: %T", clip))
	}
	switch visitStates[clip] {
	case clipVisitStateVisiting:
		panic("vector: cyclic clip definition")
	case clipVisitStateVisited:
		return
	}
	visitStates[clip] = clipVisitStateVisiting
	for _, child := range children {
		validateClip(child, visitStates)
	}
	visitStates[clip] = clipVisitStateVisited
}

// isEmptyClip reports whether a validated clip is provably empty.
func isEmptyClip(clip Clip, emptyClips map[Clip]bool) bool {
	if clip == nil {
		return true
	}
	if empty, ok := emptyClips[clip]; ok {
		return empty
	}
	var empty bool
	switch clip := clip.(type) {
	case *PathClip:
		empty = clip.Path == nil || len(clip.Path.subPaths) == 0
	case *ClipSet:
		switch clip.Operation {
		case ClipOperationUnion:
			empty = true
			for _, child := range clip.Children {
				if !isEmptyClip(child, emptyClips) {
					empty = false
					break
				}
			}
		case ClipOperationIntersection:
			for _, child := range clip.Children {
				if isEmptyClip(child, emptyClips) {
					empty = true
					break
				}
			}
		}
	}
	emptyClips[clip] = empty
	return empty
}

// addClip attaches a plan for a validated clip to the drawing at drawPathIndex.
// drawPath is the original path registered at drawPathIndex.
func (f *fillPathsState) addClip(drawPathIndex int, drawPath *Path, clip Clip, bounds image.Rectangle, pathIndices map[*Path]int, visitStates map[Clip]clipVisitState) {
	plan := theClipPlansPool.Get().(*clipPlan)
	clipBounds := bounds.Intersect(drawPath.Bounds())
	pathIndices[drawPath] = drawPathIndex
	clear(visitStates)
	f.registerClipPaths(clip, clipBounds, pathIndices, visitStates)
	plan.build(clip, pathIndices)
	if f.drawPathIndexToClipPlan == nil {
		f.drawPathIndexToClipPlan = map[int]*clipPlan{}
	}
	f.drawPathIndexToClipPlan[drawPathIndex] = plan
}

// registerClipPaths adds paths from a validated clip to f and records their indices.
func (f *fillPathsState) registerClipPaths(clip Clip, bounds image.Rectangle, pathIndices map[*Path]int, visitStates map[Clip]clipVisitState) {
	if clip == nil || visitStates[clip] == clipVisitStateVisited {
		return
	}
	visitStates[clip] = clipVisitStateVisited
	switch clip := clip.(type) {
	case *PathClip:
		if clip.Path == nil {
			return
		}
		if _, ok := pathIndices[clip.Path]; ok {
			return
		}
		pathIndices[clip.Path] = f.addPath(clip.Path, bounds, ebiten.ColorScale{})
	case *ClipSet:
		for _, child := range clip.Children {
			f.registerClipPaths(child, bounds, pathIndices, visitStates)
		}
	}
}

type clipElement struct {
	// pathIndex indexes fillPathsState.paths, or is invalidPathIndex.
	pathIndex int
	fillRule  FillRule
}

type clipNode struct {
	// pathIndex indexes fillPathsState.paths, or is invalidPathIndex.
	pathIndex int
	fillRule  FillRule
	operation ClipOperation
	children  []int
	uses      int
	height    int
}

// clipPlan describes the clips applied to a drawing.
type clipPlan struct {
	nodes            []clipNode
	childNodeIndices []int
	rootNodeIndex    int
	kind             clipBatchKind
	leaves           []clipElement
	seen             []bool
	clipNodeIndices  map[Clip]int
}

var theClipPlansPool = sync.Pool{
	New: func() any {
		return &clipPlan{
			clipNodeIndices: map[Clip]int{},
		}
	},
}

func (p *clipPlan) release() {
	clear(p.clipNodeIndices)
	clear(p.nodes)
	p.nodes = p.nodes[:0]
	p.childNodeIndices = p.childNodeIndices[:0]
	p.rootNodeIndex = 0
	p.kind = clipBatchKindNone
	p.leaves = p.leaves[:0]
	p.seen = p.seen[:0]
	theClipPlansPool.Put(p)
}

type clipBatchKind uint8

const (
	clipBatchKindNone clipBatchKind = iota
	clipBatchKindEmpty
	clipBatchKindUnion
	clipBatchKindIntersection
)

func (p *clipPlan) appendBatchLeaves(nodeIndex int, operation ClipOperation) bool {
	if p.seen[nodeIndex] {
		return true
	}
	p.seen[nodeIndex] = true
	node := &p.nodes[nodeIndex]
	if len(node.children) == 0 {
		if node.pathIndex != invalidPathIndex {
			p.leaves = append(p.leaves, clipElement{
				pathIndex: node.pathIndex,
				fillRule:  node.fillRule,
			})
			return true
		}
		return node.operation == operation
	}
	if len(node.children) == 1 {
		return p.appendBatchLeaves(node.children[0], operation)
	}
	if node.operation != operation {
		return false
	}
	for _, child := range node.children {
		if !p.appendBatchLeaves(child, operation) {
			return false
		}
	}
	return true
}

// build constructs a plan for a validated clip with paths registered in pathIndices.
func (p *clipPlan) build(clip Clip, pathIndices map[*Path]int) {
	p.rootNodeIndex = p.appendClip(clip, pathIndices)
	// Graph identities are needed only while appending nodes.
	clear(p.clipNodeIndices)
	p.seen = slices.Grow(p.seen[:0], len(p.nodes))[:len(p.nodes)]
	clear(p.seen)
	if p.appendBatchLeaves(p.rootNodeIndex, ClipOperationIntersection) {
		p.kind = clipBatchKindIntersection
		return
	}
	p.leaves = p.leaves[:0]
	clear(p.seen)
	if p.appendBatchLeaves(p.rootNodeIndex, ClipOperationUnion) {
		p.kind = clipBatchKindUnion
		if len(p.leaves) == 0 {
			p.kind = clipBatchKindEmpty
		}
		return
	}
	p.kind = clipBatchKindNone
	p.leaves = p.leaves[:0]
}

// appendClips appends validated clips with registered paths and returns their node indices.
func (p *clipPlan) appendClips(clips []Clip, pathIndices map[*Path]int) []int {
	start := len(p.childNodeIndices)
	end := start + len(clips)
	p.childNodeIndices = slices.Grow(p.childNodeIndices, len(clips))[:end]
	for i, clip := range clips {
		// Recursion can grow the buffer, so resolve its region after appending the child.
		nodeIndex := p.appendClip(clip, pathIndices)
		p.childNodeIndices[start+i] = nodeIndex
	}
	return p.childNodeIndices[start:end:end]
}

// appendClip appends a validated clip with registered paths and returns its node index.
func (p *clipPlan) appendClip(clip Clip, pathIndices map[*Path]int) int {
	idx, ok := p.clipNodeIndices[clip]
	if !ok {
		node := clipNode{
			pathIndex: invalidPathIndex,
		}
		switch clip := clip.(type) {
		case *PathClip:
			node.fillRule = clip.FillRule
			if clip.Path != nil {
				node.pathIndex, ok = pathIndices[clip.Path]
				if !ok {
					panic("vector: clip path must be registered")
				}
			}
		case *ClipSet:
			node.operation = clip.Operation
			node.children = p.appendClips(clip.Children, pathIndices)
			for _, child := range node.children {
				node.height = max(node.height, p.nodes[child].height+1)
			}
			// Deeper subtrees precede shallower ones so their masks can be adopted.
			slices.SortFunc(node.children, func(a, b int) int {
				if aHeight, bHeight := p.nodes[a].height, p.nodes[b].height; aHeight != bHeight {
					return cmp.Compare(bHeight, aHeight)
				}
				aPath := p.nodes[a].pathIndex != invalidPathIndex
				bPath := p.nodes[b].pathIndex != invalidPathIndex
				if aPath != bPath {
					if aPath {
						return 1
					}
					return -1
				}
				return cmp.Compare(b, a)
			})
		}
		// Each clip is appended once, after its children.
		idx = len(p.nodes)
		p.nodes = append(p.nodes, node)
		p.clipNodeIndices[clip] = idx
	}
	p.nodes[idx].uses++
	return idx
}

// theClipImages and clipImageAgingHookRegistered must be accessed under theFillPathM.
var (
	theClipImages                clipImagePool
	clipImageAgingHookRegistered bool
)

// clipImagePixelBudget is the allocated-pixel budget for mask preparation and normal idle images.
// All idle images expire after the unused tick limit. Without updates, images never age.
const clipImagePixelBudget = 4 * 1024 * 1024

const clipImageMaxUnusedTicks = 60

// clipImageMaxReuseAreaRatio is the maximum image area relative to a rounded request.
const clipImageMaxReuseAreaRatio = 2

const clipImageIndexNone = -1

const (
	clipSamplePlanesNonAA = 1
	clipSamplePlanesAA    = 2
)

type clipImage struct {
	image   *ebiten.Image
	lastUse int64
	batched bool
}

type clipImagePool struct {
	images          []clipImage
	pixels          int64
	oversizedImages []clipImage
	oversizedPixels int64
	maxImageSize    int
}

func (p *clipImagePool) allocationSize(width, height int) image.Point {
	if p.maxImageSize == 0 {
		p.maxImageSize = ebiten.MaxImageSize()
	}
	limit := p.maxImageSize
	return image.Point{
		X: max(width, min(roundUpAtlasSize(width), limit)),
		Y: max(height, min(roundUpAtlasSize(height), limit)),
	}
}

func (p *clipImagePool) selectImage(width, height int, batched bool) (oversized bool, index int, size image.Point) {
	size = p.allocationSize(width, height)
	maxArea := clipImageMaxReuseAreaRatio * int64(size.X) * int64(size.Y)
	// Each request searches its own size group.
	oversized = int64(size.X)*int64(size.Y) > clipImagePixelBudget
	count := len(p.images)
	if oversized {
		count = len(p.oversizedImages)
	}
	index = clipImageIndexNone
	var bestArea int64
	for i := range count {
		var entry clipImage
		if oversized {
			entry = p.oversizedImages[i]
		} else {
			entry = p.images[i]
		}
		// Intermediate masks and batched results have different render dependencies.
		if entry.batched != batched {
			continue
		}
		b := entry.image.Bounds()
		if b.Dx() < width || b.Dy() < height {
			continue
		}
		area := int64(b.Dx()) * int64(b.Dy())
		if area > maxArea {
			continue
		}
		if index == clipImageIndexNone || area <= bestArea {
			index = i
			bestArea = area
			size = b.Size()
		}
	}
	return oversized, index, size
}

func (p *clipImagePool) imageSize(width, height int, batched bool) image.Point {
	_, _, size := p.selectImage(width, height, batched)
	return size
}

func (p *clipImagePool) get(width, height int, batched bool) *ebiten.Image {
	oversized, index, size := p.selectImage(width, height, batched)
	var img *ebiten.Image
	if index != clipImageIndexNone {
		pixels := int64(size.X) * int64(size.Y)
		if oversized {
			img = p.oversizedImages[index].image
			p.oversizedImages = slices.Delete(p.oversizedImages, index, index+1)
			p.oversizedPixels -= pixels
		} else {
			img = p.images[index].image
			p.images = slices.Delete(p.images, index, index+1)
			p.pixels -= pixels
		}
	} else {
		img = ebiten.NewImage(size.X, size.Y)
		if !clipImageAgingHookRegistered {
			hook.AppendHookOnBeforeUpdate(releaseUnusedClipImages)
			clipImageAgingHookRegistered = true
		}
	}
	sub := img.RecyclableSubImage(image.Rect(0, 0, width, height))
	sub.Clear()
	return sub
}

func (p *clipImagePool) put(img *ebiten.Image, batched bool) {
	original := theImageBridge.OriginalImage(img)
	img.Recycle()
	b := original.Bounds()
	pixels := int64(b.Dx()) * int64(b.Dy())
	if pixels > clipImagePixelBudget {
		p.oversizedImages = append(p.oversizedImages, clipImage{
			image:   original,
			lastUse: ebiten.Tick(),
			batched: batched,
		})
		p.oversizedPixels += pixels
		return
	}
	for p.pixels+pixels > clipImagePixelBudget {
		evicted := p.images[0].image
		b := evicted.Bounds()
		p.images = slices.Delete(p.images, 0, 1)
		p.pixels -= int64(b.Dx()) * int64(b.Dy())
		evicted.Deallocate()
	}
	p.images = append(p.images, clipImage{
		image:   original,
		lastUse: ebiten.Tick(),
		batched: batched,
	})
	p.pixels += pixels
}

func releaseUnusedClipImages() error {
	theFillPathM.Lock()
	defer theFillPathM.Unlock()

	theClipImages.releaseUnused(ebiten.Tick())
	return nil
}

func (p *clipImagePool) releaseUnused(tick int64) {
	for len(p.images) > 0 {
		entry := p.images[0]
		if tick-entry.lastUse <= clipImageMaxUnusedTicks {
			break
		}
		b := entry.image.Bounds()
		p.images = slices.Delete(p.images, 0, 1)
		p.pixels -= int64(b.Dx()) * int64(b.Dy())
		entry.image.Deallocate()
	}
	p.releaseUnusedOversized(tick)
}

func (p *clipImagePool) releaseUnusedOversized(tick int64) {
	for i := 0; i < len(p.oversizedImages); {
		entry := p.oversizedImages[i]
		if tick-entry.lastUse <= clipImageMaxUnusedTicks {
			i++
			continue
		}
		b := entry.image.Bounds()
		p.oversizedImages = slices.Delete(p.oversizedImages, i, i+1)
		p.oversizedPixels -= int64(b.Dx()) * int64(b.Dy())
		entry.image.Deallocate()
	}
}

var (
	clipUnionBlend = ebiten.Blend{
		BlendOperationRGB:   ebiten.BlendOperationMax,
		BlendOperationAlpha: ebiten.BlendOperationMax,
	}
	clipIntersectionBlend = ebiten.Blend{
		BlendOperationRGB:   ebiten.BlendOperationMin,
		BlendOperationAlpha: ebiten.BlendOperationMin,
	}
)

func releaseClipMask(masks []*ebiten.Image, uses []int, nodeIndex int) {
	uses[nodeIndex]--
	if uses[nodeIndex] == 0 {
		theClipImages.put(masks[nodeIndex], false)
		masks[nodeIndex] = nil
	}
}

type clipMaskState struct {
	position image.Point
	width    int
	height   int
	planes   int
	masks    []*ebiten.Image
	uses     []int
}

func (f *fillPathsState) buildClipMask(plan *clipPlan, nodeIndex int, state *clipMaskState) *ebiten.Image {
	if mask := state.masks[nodeIndex]; mask != nil {
		return mask
	}
	node := &plan.nodes[nodeIndex]
	if node.pathIndex != invalidPathIndex {
		mask := theClipImages.get(state.width*state.planes, state.height, false)
		f.decodeClipLeaf(mask, state.position, state.width, state.height, state.planes, clipElement{
			pathIndex: node.pathIndex,
			fillRule:  node.fillRule,
		}, clipUnionBlend)
		state.masks[nodeIndex] = mask
		return mask
	}
	blend := clipUnionBlend
	if node.operation == ClipOperationIntersection {
		blend = clipIntersectionBlend
	}
	var mask *ebiten.Image
	for _, child := range node.children {
		childNode := &plan.nodes[child]
		if len(childNode.children) == 0 && childNode.pathIndex != invalidPathIndex && state.masks[child] == nil && state.uses[child] == 1 {
			pathBlend := blend
			if mask == nil {
				mask = theClipImages.get(state.width*state.planes, state.height, false)
				pathBlend = clipUnionBlend
			}
			f.decodeClipLeaf(mask, state.position, state.width, state.height, state.planes, clipElement{
				pathIndex: childNode.pathIndex,
				fillRule:  childNode.fillRule,
			}, pathBlend)
			state.uses[child]--
			continue
		}
		childMask := f.buildClipMask(plan, child, state)
		if mask == nil {
			if state.uses[child] == 1 {
				mask = childMask
				state.masks[child] = nil
				state.uses[child] = 0
			} else {
				mask = theClipImages.get(state.width*state.planes, state.height, false)
				mask.DrawImage(childMask, &ebiten.DrawImageOptions{
					Blend: ebiten.BlendCopy,
				})
				releaseClipMask(state.masks, state.uses, child)
			}
		} else {
			mask.DrawImage(childMask, &ebiten.DrawImageOptions{
				Blend: blend,
			})
			releaseClipMask(state.masks, state.uses, child)
		}
	}

	if mask == nil {
		mask = theClipImages.get(state.width*state.planes, state.height, false)
		if node.operation == ClipOperationIntersection {
			mask.Fill(color.White)
		}
	}
	state.masks[nodeIndex] = mask
	return mask
}

func (f *fillPathsState) clipSamples(pathIndex int, plan *clipPlan) *ebiten.Image {
	b := theAtlas.stencilBufferImageAt(pathIndex, f.antialias, 0).Bounds()
	state := clipMaskState{
		position: theAtlas.pathRenderingPositionAt(pathIndex),
		width:    b.Dx(),
		height:   b.Dy(),
		planes:   clipSamplePlanesNonAA,
		masks:    make([]*ebiten.Image, len(plan.nodes)),
		uses:     make([]int, len(plan.nodes)),
	}
	if f.antialias {
		state.planes = clipSamplePlanesAA
	}
	for i, node := range plan.nodes {
		state.uses[i] = node.uses
	}
	result := f.buildClipMask(plan, plan.rootNodeIndex, &state)
	if state.uses[plan.rootNodeIndex] != 1 {
		panic("vector: root clip mask must have exactly one remaining use")
	}
	state.masks[plan.rootNodeIndex] = nil
	state.uses[plan.rootNodeIndex] = 0
	f.decodeClipLeaf(result, state.position, state.width, state.height, state.planes, clipElement{
		pathIndex: pathIndex,
		fillRule:  f.fillRule,
	}, clipIntersectionBlend)
	return result
}

func (p *clipPlan) discardMissingStencils(antialias bool) {
	switch p.kind {
	case clipBatchKindIntersection:
		for _, leaf := range p.leaves {
			if theAtlas.stencilBufferImageAt(leaf.pathIndex, antialias, 0) == nil {
				p.kind = clipBatchKindEmpty
				return
			}
		}
	case clipBatchKindUnion:
		for _, leaf := range p.leaves {
			if theAtlas.stencilBufferImageAt(leaf.pathIndex, antialias, 0) != nil {
				return
			}
		}
		p.kind = clipBatchKindEmpty
	}
}

func (f *fillPathsState) canMaskUnclippedPaths() bool {
	if len(f.drawPathIndexToClipPlan) == len(f.drawPathIndices) {
		return false
	}
	planes := clipSamplePlanesNonAA
	if f.antialias {
		planes = clipSamplePlanesAA
	}
	var pixels int64
	var runs int
	var lastClipped bool
	var hasShapeGroup bool
	var extraCommands int
	var blockUnclipped bool
	var blockFlat bool
	for _, i := range f.drawPathIndices {
		plan := f.drawPathIndexToClipPlan[i]
		if plan != nil && plan.kind == clipBatchKindEmpty {
			continue
		}
		stencil := theAtlas.stencilBufferImageAt(i, f.antialias, 0)
		if stencil == nil {
			continue
		}
		clipped := plan != nil
		if runs == 0 || clipped != lastClipped {
			runs++
		}
		lastClipped = clipped
		if plan != nil && plan.kind == clipBatchKindNone {
			if blockUnclipped && !blockFlat {
				extraCommands++
			}
			blockUnclipped = false
			blockFlat = false
		} else {
			blockUnclipped = blockUnclipped || !clipped
			blockFlat = blockFlat || clipped
		}
		if plan != nil {
			switch plan.kind {
			case clipBatchKindIntersection:
				hasShapeGroup = true
			case clipBatchKindUnion:
				for _, leaf := range plan.leaves {
					hasShapeGroup = hasShapeGroup || leaf.fillRule == f.fillRule && theAtlas.stencilBufferImageAt(leaf.pathIndex, f.antialias, 0) != nil
				}
			}
		}
		b := stencil.Bounds()
		size := theClipImages.allocationSize(b.Dx()*planes, b.Dy())
		area := clipImageMaxReuseAreaRatio * int64(size.X) * int64(size.Y)
		if area > clipImagePixelBudget-pixels {
			return false
		}
		pixels += area
	}
	if blockUnclipped && !blockFlat {
		extraCommands++
	}
	if !hasShapeGroup {
		extraCommands++
	}
	// Reuse is bounded per request, so all results fit one preparation chunk.
	// The final-draw runs saved must exceed the added clear and decode groups.
	return runs > extraCommands+1
}

func (f *fillPathsState) prepareClipImages(images []*ebiten.Image, start int, includeUnclipped bool) int {
	planes := clipSamplePlanesNonAA
	if f.antialias {
		planes = clipSamplePlanesAA
	}
	var pixels int64
	end := start
	for ; end < len(f.drawPathIndices); end++ {
		i := f.drawPathIndices[end]
		plan := f.drawPathIndexToClipPlan[i]
		if plan == nil && !includeUnclipped || plan != nil && plan.kind == clipBatchKindEmpty {
			continue
		}
		stencil := theAtlas.stencilBufferImageAt(i, f.antialias, 0)
		if stencil == nil {
			continue
		}
		b := stencil.Bounds()
		batched := plan == nil || plan.kind != clipBatchKindNone
		size := theClipImages.imageSize(b.Dx()*planes, b.Dy(), batched)
		area := int64(size.X) * int64(size.Y)
		if pixels > 0 && area > clipImagePixelBudget-pixels {
			break
		}
		// Admission includes allocation padding. Shared children can affect which
		// image supplies the result, so charge its actual allocated region.
		// An oversized result is admitted alone.
		if plan != nil && plan.kind == clipBatchKindNone {
			images[end] = f.clipSamples(i, plan)
		} else {
			images[end] = theClipImages.get(b.Dx()*planes, b.Dy(), true)
		}
		allocated := theImageBridge.OriginalImage(images[end]).Bounds()
		pixels += int64(allocated.Dx()) * int64(allocated.Dy())
	}
	f.drawBatchedClipMasks(images, start, end, planes)
	return end
}

func (f *fillPathsState) drawBatchedClipMasks(images []*ebiten.Image, start, end, planes int) {
	// Mask regions are independent, and each region's unions precede its intersections.
	for _, pass := range []struct {
		blend     ebiten.Blend
		shapeKind clipBatchKind
	}{
		{
			blend:     clipUnionBlend,
			shapeKind: clipBatchKindIntersection,
		},
		{
			blend:     clipIntersectionBlend,
			shapeKind: clipBatchKindUnion,
		},
	} {
		for _, rule := range []FillRule{FillRuleNonZero, FillRuleEvenOdd} {
			for j := start; j < end; j++ {
				if images[j] == nil {
					continue
				}
				i := f.drawPathIndices[j]
				plan := f.drawPathIndexToClipPlan[i]
				kind := clipBatchKindIntersection
				if plan != nil {
					kind = plan.kind
				}
				if kind == clipBatchKindNone || kind == clipBatchKindEmpty {
					continue
				}
				position := theAtlas.pathRenderingPositionAt(i)
				b := theAtlas.stencilBufferImageAt(i, f.antialias, 0).Bounds()
				if kind == pass.shapeKind {
					if f.fillRule == rule {
						f.decodeClipLeaf(images[j], position, b.Dx(), b.Dy(), planes, clipElement{
							pathIndex: i,
							fillRule:  f.fillRule,
						}, pass.blend)
					}
					continue
				}
				if plan == nil {
					continue
				}
				for _, element := range plan.leaves {
					if element.fillRule == rule {
						f.decodeClipLeaf(images[j], position, b.Dx(), b.Dy(), planes, element, pass.blend)
					}
				}
			}
		}
	}
}

func (f *fillPathsState) decodeClipLeaf(img *ebiten.Image, position image.Point, width, height, planes int, element clipElement, blend ebiten.Blend) {
	// Each mask channel stores an independent binary coverage sample.
	if element.pathIndex == invalidPathIndex {
		if blend.BlendOperationRGB == ebiten.BlendOperationMin {
			img.Clear()
		}
		return
	}
	for plane := range planes {
		src := theAtlas.stencilBufferImageAt(element.pathIndex, f.antialias, plane)
		if src == nil {
			if blend.BlendOperationRGB == ebiten.BlendOperationMin {
				img.Clear()
			}
			continue
		}
		pp := theAtlas.pathRenderingPositionAt(element.pathIndex)
		x := float32(src.Bounds().Min.X + position.X - pp.X)
		y := float32(src.Bounds().Min.Y + position.Y - pp.Y)
		dx := float32(plane * width)
		w, h := float32(width), float32(height)
		// Custom attributes carry the sample position and bounds in source-local coordinates.
		localX := x - float32(src.Bounds().Min.X)
		localY := y - float32(src.Bounds().Min.Y)
		sourceWidth := float32(src.Bounds().Dx())
		sourceHeight := float32(src.Bounds().Dy())
		vertices := []ebiten.Vertex{
			{
				DstX:    dx,
				SrcX:    x,
				SrcY:    y,
				Custom0: localX,
				Custom1: localY,
				Custom2: sourceWidth,
				Custom3: sourceHeight,
			},
			{
				DstX:    dx + w,
				SrcX:    x + w,
				SrcY:    y,
				Custom0: localX + w,
				Custom1: localY,
				Custom2: sourceWidth,
				Custom3: sourceHeight,
			},
			{
				DstX:    dx,
				DstY:    h,
				SrcX:    x,
				SrcY:    y + h,
				Custom0: localX,
				Custom1: localY + h,
				Custom2: sourceWidth,
				Custom3: sourceHeight,
			},
			{
				DstX:    dx + w,
				DstY:    h,
				SrcX:    x + w,
				SrcY:    y + h,
				Custom0: localX + w,
				Custom1: localY + h,
				Custom2: sourceWidth,
				Custom3: sourceHeight,
			},
		}
		op := &ebiten.DrawTrianglesShaderOptions{}
		op.Blend = blend
		op.Images[0] = src
		img.DrawTrianglesShader32(vertices, []uint32{0, 1, 2, 1, 2, 3}, ensureClipDecodeShader(element.fillRule), op)
	}
}

type clipShaders struct {
	decodeNonZero *ebiten.Shader
	decodeEvenOdd *ebiten.Shader
	resolve       *ebiten.Shader
	resolveAA     *ebiten.Shader
}

var theClipShaders clipShaders

func ensureClipDecodeShader(fillRule FillRule) *ebiten.Shader {
	shader := &theClipShaders.decodeNonZero
	source := clipDecodeNonZeroShaderSrc
	if fillRule == FillRuleEvenOdd {
		shader = &theClipShaders.decodeEvenOdd
		source = clipDecodeEvenOddShaderSrc
	}
	if *shader != nil {
		return *shader
	}
	s, err := ebiten.NewShader([]byte(source))
	if err != nil {
		panic(fmt.Sprintf("vector: failed to create clip decode shader: %v", err))
	}
	*shader = s
	return s
}

func ensureClipResolveShader(antialias bool) *ebiten.Shader {
	shader := &theClipShaders.resolve
	source := clipResolveShaderSrc
	if antialias {
		shader = &theClipShaders.resolveAA
		source = clipResolveAAShaderSrc
	}
	if *shader != nil {
		return *shader
	}
	s, err := ebiten.NewShader([]byte(source))
	if err != nil {
		panic(fmt.Sprintf("vector: failed to create clip resolve shader: %v", err))
	}
	*shader = s
	return s
}

//ebitengine:shadersource
const clipDecodeNonZeroShaderSrc = `//kage:unit pixels
package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	if custom.x < 0 || custom.y < 0 || custom.x >= custom.z || custom.y >= custom.w {
		return vec4(0)
	}
	// Each stencil byte contains two four-bit winding counts.
	c := ivec4(floor(imageSrc0UnsafeAt(src0Pos)*255 + 0.5))
	w := abs((c >> 4) - (c & 0x0F))
	return min(vec4(w), 1)
}
`

//ebitengine:shadersource
const clipDecodeEvenOddShaderSrc = `//kage:unit pixels
package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	if custom.x < 0 || custom.y < 0 || custom.x >= custom.z || custom.y >= custom.w {
		return vec4(0)
	}
	// Each stencil byte contains two four-bit winding counts.
	c := ivec4(floor(imageSrc0UnsafeAt(src0Pos)*255 + 0.5))
	w := abs((c >> 4) - (c & 0x0F))
	return vec4(w % 2)
}
`

//ebitengine:shadersource
const clipResolveShaderSrc = `//kage:unit pixels
package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(src0Pos)
	v := c.r
	// Pixels outside the clipped shape leave the destination untouched.
	if v == 0 {
		discard()
	}
	return v * color
}
`

//ebitengine:shadersource
const clipResolveAAShaderSrc = `//kage:unit pixels
package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	// The two RGBA planes contain eight independent samples.
	c := imageSrc0UnsafeAt(src0Pos)
	c += imageSrc0UnsafeAt(src0Pos + custom.xy)
	v := dot(c, vec4(1.0/8.0))
	// Pixels outside the clipped shape leave the destination untouched.
	if v == 0 {
		discard()
	}
	return v * color
}
`
