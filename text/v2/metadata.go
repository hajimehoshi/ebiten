// Copyright 2023 The Ebitengine Authors
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

package text

import (
	"github.com/go-text/typesetting/opentype/api/metadata"
	"github.com/go-text/typesetting/opentype/loader"
)

// Metadata represents a font face's metadata.
type Metadata struct {
	Family  string
	Style   Style
	Weight  Weight
	Stretch Stretch
}

func metadataFromLoader(l *loader.Loader) Metadata {
	f, a, _ := metadata.Describe(l, nil)
	return Metadata{
		Family:  f,
		Style:   Style(a.Style),
		Weight:  Weight(a.Weight),
		Stretch: Stretch(a.Stretch),
	}
}

type Style uint8

const (
	StyleNormal Style = Style(metadata.StyleNormal)
	StyleItalic Style = Style(metadata.StyleItalic)
)

type Weight float32

const (
	WeightThin       Weight = Weight(metadata.WeightThin)
	WeightExtraLight Weight = Weight(metadata.WeightExtraLight)
	WeightLight      Weight = Weight(metadata.WeightLight)
	WeightNormal     Weight = Weight(metadata.WeightNormal)
	WeightMedium     Weight = Weight(metadata.WeightMedium)
	WeightSemibold   Weight = Weight(metadata.WeightSemibold)
	WeightBold       Weight = Weight(metadata.WeightBold)
	WeightExtraBold  Weight = Weight(metadata.WeightExtraBold)
	WeightBlack      Weight = Weight(metadata.WeightBlack)
)

type Stretch float32

const (
	StretchUltraCondensed Stretch = Stretch(metadata.StretchUltraCondensed)
	StretchExtraCondensed Stretch = Stretch(metadata.StretchExtraCondensed)
	StretchCondensed      Stretch = Stretch(metadata.StretchCondensed)
	StretchSemiCondensed  Stretch = Stretch(metadata.StretchSemiCondensed)
	StretchNormal         Stretch = Stretch(metadata.StretchNormal)
	StretchSemiExpanded   Stretch = Stretch(metadata.StretchSemiExpanded)
	StretchExpanded       Stretch = Stretch(metadata.StretchExpanded)
	StretchExtraExpanded  Stretch = Stretch(metadata.StretchExtraExpanded)
	StretchUltraExpanded  Stretch = Stretch(metadata.StretchUltraExpanded)
)
