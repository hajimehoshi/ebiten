// Copyright 2022 The Ebiten Authors
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

package gamepad

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type ID int

const (
	hatCentered  = 0
	hatUp        = 1
	hatRight     = 2
	hatDown      = 4
	hatLeft      = 8
	hatRightUp   = hatRight | hatUp
	hatRightDown = hatRight | hatDown
	hatLeftUp    = hatLeft | hatUp
	hatLeftDown  = hatLeft | hatDown
)

type gamepads struct {
	inited   bool
	gamepads []*Gamepad
	m        sync.Mutex

	native nativeGamepads
}

type nativeGamepads interface {
	init(gamepads *gamepads) error
	update(gamepads *gamepads) error
}

var theGamepads = gamepads{
	native: newNativeGamepadsImpl(),
}

// AppendGamepadIDs is concurrent-safe.
func AppendGamepadIDs(ids []ID) []ID {
	return theGamepads.appendGamepadIDs(ids)
}

// Update is concurrent-safe.
func Update() error {
	return theGamepads.update()
}

// Get is concurrent-safe.
func Get(id ID) *Gamepad {
	return theGamepads.get(id)
}

func SetNativeWindow(nativeWindow uintptr) {
	theGamepads.setNativeWindow(nativeWindow)
}

func (g *gamepads) appendGamepadIDs(ids []ID) []ID {
	g.m.Lock()
	defer g.m.Unlock()

	for i, gp := range g.gamepads {
		if gp != nil {
			ids = append(ids, ID(i))
		}
	}
	return ids
}

func (g *gamepads) update() error {
	g.m.Lock()
	defer g.m.Unlock()

	if !g.inited {
		if err := g.native.init(g); err != nil {
			return err
		}
		g.inited = true
	}

	if err := g.native.update(g); err != nil {
		return err
	}

	// A gamepad can be detected even though there are not. Apparently, some special devices are
	// recognized as gamepads by OSes. In this case, the number of the 'buttons' can exceed the
	// maximum. Skip such devices as a tentative solution (#1173, #2039).
	g.remove(func(gamepad *Gamepad) bool {
		return gamepad.ButtonCount() > ButtonCount
	})

	for _, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		if err := gp.update(g); err != nil {
			return err
		}
	}
	return nil
}

func (g *gamepads) get(id ID) *Gamepad {
	g.m.Lock()
	defer g.m.Unlock()

	if id < 0 || int(id) >= len(g.gamepads) {
		return nil
	}
	return g.gamepads[id]
}

func (g *gamepads) find(cond func(*Gamepad) bool) *Gamepad {
	for _, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		if cond(gp) {
			return gp
		}
	}
	return nil
}

func (g *gamepads) add(name, sdlID string) *Gamepad {
	for i, gp := range g.gamepads {
		if gp == nil {
			gp := &Gamepad{
				name:  name,
				sdlID: sdlID,
			}
			g.gamepads[i] = gp
			return gp
		}
	}

	gp := &Gamepad{
		name:  name,
		sdlID: sdlID,
	}
	g.gamepads = append(g.gamepads, gp)
	return gp
}

func (g *gamepads) remove(cond func(*Gamepad) bool) {
	for i, gp := range g.gamepads {
		if gp == nil {
			continue
		}
		if cond(gp) {
			g.gamepads[i] = nil
		}
	}
}

func (g *gamepads) setNativeWindow(nativeWindow uintptr) {
	g.m.Lock()
	defer g.m.Unlock()

	var n any = g.native
	if n, ok := n.(interface{ setNativeWindow(uintptr) }); ok {
		n.setNativeWindow(nativeWindow)
	}
}

type Gamepad struct {
	name  string
	sdlID string
	m     sync.Mutex

	native nativeGamepad
}

type mappingInput interface {
	Pressed() bool
	Value() float64 // Normalized to range: 0..1.
}

type axisMappingInput struct {
	g    nativeGamepad
	axis int
}

func (a axisMappingInput) Pressed() bool {
	return a.g.axisValue(a.axis) > gamepaddb.ButtonPressedThreshold
}

func (a axisMappingInput) Value() float64 {
	return a.g.axisValue(a.axis)*0.5 + 0.5
}

type buttonMappingInput struct {
	g      nativeGamepad
	button int
}

func (b buttonMappingInput) Pressed() bool {
	return b.g.isButtonPressed(b.button)
}

func (b buttonMappingInput) Value() float64 {
	return b.g.buttonValue(b.button)
}

type hatMappingInput struct {
	g         nativeGamepad
	hat       int
	direction int
}

func (h hatMappingInput) Pressed() bool {
	return h.g.hatState(h.hat)&h.direction != 0
}

func (h hatMappingInput) Value() float64 {
	if h.Pressed() {
		return 1
	}
	return 0
}

type nativeGamepad interface {
	update(gamepads *gamepads) error
	hasOwnStandardLayoutMapping() bool
	standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput
	standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput
	axisCount() int
	buttonCount() int
	hatCount() int
	axisValue(axis int) float64
	buttonValue(button int) float64
	isButtonPressed(button int) bool
	hatState(hat int) int
	vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64)
}

func (g *Gamepad) update(gamepads *gamepads) error {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.update(gamepads)
}

// Name is concurrent-safe.
func (g *Gamepad) Name() string {
	// This is immutable and doesn't have to be protected by a mutex.
	if name := gamepaddb.Name(g.sdlID); name != "" {
		return name
	}
	return g.name
}

// SDLID is concurrent-safe.
func (g *Gamepad) SDLID() string {
	// This is immutable and doesn't have to be protected by a mutex.
	return g.sdlID
}

// AxisCount is concurrent-safe.
func (g *Gamepad) AxisCount() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.axisCount()
}

// ButtonCount is concurrent-safe.
func (g *Gamepad) ButtonCount() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.buttonCount()
}

// HatCount is concurrent-safe.
func (g *Gamepad) HatCount() int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.hatCount()
}

// Axis is concurrent-safe.
func (g *Gamepad) Axis(axis int) float64 {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.axisValue(axis)
}

// Button is concurrent-safe.
func (g *Gamepad) Button(button int) bool {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.isButtonPressed(button)
}

// Hat is concurrent-safe.
func (g *Gamepad) Hat(hat int) int {
	g.m.Lock()
	defer g.m.Unlock()

	return g.native.hatState(hat)
}

// IsStandardLayoutAvailable is concurrent-safe.
func (g *Gamepad) IsStandardLayoutAvailable() bool {
	g.m.Lock()
	defer g.m.Unlock()

	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return true
	}
	return g.native.hasOwnStandardLayoutMapping()
}

// IsStandardAxisAvailable is concurrent safe.
func (g *Gamepad) IsStandardAxisAvailable(axis gamepaddb.StandardAxis) bool {
	g.m.Lock()
	defer g.m.Unlock()

	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.HasStandardAxis(g.sdlID, axis)
	}
	return g.native.standardAxisInOwnMapping(axis) != nil
}

// IsStandardButtonAvailable is concurrent safe.
func (g *Gamepad) IsStandardButtonAvailable(button gamepaddb.StandardButton) bool {
	g.m.Lock()
	defer g.m.Unlock()

	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		return gamepaddb.HasStandardButton(g.sdlID, button)
	}
	return g.native.standardButtonInOwnMapping(button) != nil
}

// StandardAxisValue is concurrent-safe.
func (g *Gamepad) StandardAxisValue(axis gamepaddb.StandardAxis) float64 {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		// StandardAxisValue invokes g.Axis, g.Button, or g.Hat so this cannot be locked.
		return gamepaddb.StandardAxisValue(g.sdlID, axis, g)
	}

	g.m.Lock()
	defer g.m.Unlock()

	if m := g.native.standardAxisInOwnMapping(axis); m != nil {
		return m.Value()*2 - 1
	}
	return 0
}

// StandardButtonValue is concurrent-safe.
func (g *Gamepad) StandardButtonValue(button gamepaddb.StandardButton) float64 {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		// StandardButtonValue invokes g.Axis, g.Button, or g.Hat so this cannot be locked.
		return gamepaddb.StandardButtonValue(g.sdlID, button, g)
	}

	g.m.Lock()
	defer g.m.Unlock()

	if m := g.native.standardButtonInOwnMapping(button); m != nil {
		return m.Value()
	}
	return 0
}

// IsStandardButtonPressed is concurrent-safe.
func (g *Gamepad) IsStandardButtonPressed(button gamepaddb.StandardButton) bool {
	if gamepaddb.HasStandardLayoutMapping(g.sdlID) {
		// IsStandardButtonPressed invokes g.Axis, g.Button, or g.Hat so this cannot be locked.
		return gamepaddb.IsStandardButtonPressed(g.sdlID, button, g)
	}

	g.m.Lock()
	defer g.m.Unlock()

	if m := g.native.standardButtonInOwnMapping(button); m != nil {
		return m.Pressed()
	}
	return false
}

// Vibrate is concurrent-safe.
func (g *Gamepad) Vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	g.m.Lock()
	defer g.m.Unlock()

	g.native.vibrate(duration, strongMagnitude, weakMagnitude)
}

type MappingItem struct {
	StandardType                 gamepaddb.MappingType
	StandardIndex                int
	PhysicalType                 gamepaddb.MappingType
	PhysicalIndex                int
	PhysicalToStandardAxisScale  float64
	PhysicalToStandardAxisOffset float64
}

func (g *Gamepad) Mapping() []MappingItem {
	g.m.Lock()
	defer g.m.Unlock()

	if g.native.hasOwnStandardLayoutMapping() {
		var mapping []MappingItem

		buttonCount := g.native.buttonCount() + g.native.hatCount()*4
		if buttonCount > int(gamepaddb.StandardButtonMax)+1 {
			buttonCount = int(gamepaddb.StandardButtonMax) + 1
		}
		for i := 0; i < buttonCount; i++ {
			mapping = append(mapping, MappingItem{
				StandardType:  gamepaddb.MappingTypeButton,
				StandardIndex: i,
				PhysicalType:  gamepaddb.MappingTypeButton,
				PhysicalIndex: i,
			})
		}

		axisCount := g.native.axisCount()
		if axisCount > int(gamepaddb.StandardAxisMax)+1 {
			axisCount = int(gamepaddb.StandardAxisMax) + 1
		}
		for i := 0; i < axisCount; i++ {
			mapping = append(mapping, MappingItem{
				StandardType:                gamepaddb.MappingTypeAxis,
				StandardIndex:               i,
				PhysicalType:                gamepaddb.MappingTypeAxis,
				PhysicalIndex:               i,
				PhysicalToStandardAxisScale: 1,
			})
		}

		return mapping
	}

	var mapping []MappingItem
	buttons, axes := gamepaddb.UnsafeMapping(g.sdlID)
	for button, item := range buttons {
		item := g.convertMappingItem(item)
		item.StandardType = gamepaddb.MappingTypeButton
		item.StandardIndex = int(button)
		mapping = append(mapping, item)
	}
	for axis, item := range axes {
		item := g.convertMappingItem(item)
		item.StandardType = gamepaddb.MappingTypeAxis
		item.StandardIndex = int(axis)
		mapping = append(mapping, item)
	}

	return mapping
}

func (g *Gamepad) convertMappingItem(item gamepaddb.MappingItem) MappingItem {
	var r MappingItem
	switch item.Type {
	case gamepaddb.MappingTypeButton:
		r.PhysicalType = gamepaddb.MappingTypeButton
		r.PhysicalIndex = item.Index
	case gamepaddb.MappingTypeAxis:
		r.PhysicalType = gamepaddb.MappingTypeAxis
		r.PhysicalIndex = item.Index
		r.PhysicalToStandardAxisScale = item.AxisScale
		r.PhysicalToStandardAxisOffset = item.AxisOffset
	case gamepaddb.MappingTypeHat:
		// Convert a hat value to a button value.
		r.PhysicalType = gamepaddb.MappingTypeButton
		var offset int
		switch item.HatState {
		case hatUp:
			offset = 0
		case hatRight:
			offset = 1
		case hatDown:
			offset = 2
		case hatLeft:
			offset = 3
		}
		r.PhysicalIndex = g.native.buttonCount() + 4*item.Index + offset
	}
	return r
}
