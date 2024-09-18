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
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

// Metadata represents a font face's metadata.
type Metadata struct {
	Family  string
	Style   Style
	Weight  Weight
	Stretch Stretch
}

func metadataFromLoader(l *opentype.Loader) Metadata {
	d, _ := font.Describe(l, nil)
	return Metadata{
		Family:  d.Family,
		Style:   Style(d.Aspect.Style),
		Weight:  Weight(d.Aspect.Weight),
		Stretch: Stretch(d.Aspect.Stretch),
	}
}

type Style uint8

const (
	StyleNormal Style = Style(font.StyleNormal)
	StyleItalic Style = Style(font.StyleItalic)
)

type Weight float32

const (
	WeightThin       Weight = Weight(font.WeightThin)
	WeightExtraLight Weight = Weight(font.WeightExtraLight)
	WeightLight      Weight = Weight(font.WeightLight)
	WeightNormal     Weight = Weight(font.WeightNormal)
	WeightMedium     Weight = Weight(font.WeightMedium)
	WeightSemibold   Weight = Weight(font.WeightSemibold)
	WeightBold       Weight = Weight(font.WeightBold)
	WeightExtraBold  Weight = Weight(font.WeightExtraBold)
	WeightBlack      Weight = Weight(font.WeightBlack)
)

type Stretch float32

const (
	StretchUltraCondensed Stretch = Stretch(font.StretchUltraCondensed)
	StretchExtraCondensed Stretch = Stretch(font.StretchExtraCondensed)
	StretchCondensed      Stretch = Stretch(font.StretchCondensed)
	StretchSemiCondensed  Stretch = Stretch(font.StretchSemiCondensed)
	StretchNormal         Stretch = Stretch(font.StretchNormal)
	StretchSemiExpanded   Stretch = Stretch(font.StretchSemiExpanded)
	StretchExpanded       Stretch = Stretch(font.StretchExpanded)
	StretchExtraExpanded  Stretch = Stretch(font.StretchExtraExpanded)
	StretchUltraExpanded  Stretch = Stretch(font.StretchUltraExpanded)
)
