// Copyright 2013 The Ebiten Authors
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

// Code generated by genkeys.go using 'go generate'. DO NOT EDIT.

package ebiten

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// A MidiKey represents a midi key.
// These keys use C4 = 60 as the middle C as per: https://www.midi.org/forum/830-midi-octave-and-note-numbering-standard#reply-1086
type MidiKey uint8

// MidiKeys.
const (
	NoteC_1      MidiKey = MidiKey(ui.NoteC_1)
	NoteCSharp_1 MidiKey = MidiKey(ui.NoteCSharp_1)
	NoteDFlat_1  MidiKey = MidiKey(ui.NoteDFlat_1)
	NoteD_1      MidiKey = MidiKey(ui.NoteD_1)
	NoteDSharp_1 MidiKey = MidiKey(ui.NoteDSharp_1)
	NoteEFlat_1  MidiKey = MidiKey(ui.NoteEFlat_1)
	NoteE_1      MidiKey = MidiKey(ui.NoteE_1)
	NoteF_1      MidiKey = MidiKey(ui.NoteF_1)
	NoteFSharp_1 MidiKey = MidiKey(ui.NoteFSharp_1)
	NoteGFlat_1  MidiKey = MidiKey(ui.NoteGFlat_1)
	NoteG_1      MidiKey = MidiKey(ui.NoteG_1)
	NoteGSharp_1 MidiKey = MidiKey(ui.NoteGSharp_1)
	NoteAFlat_1  MidiKey = MidiKey(ui.NoteAFlat_1)
	NoteA_1      MidiKey = MidiKey(ui.NoteA_1)
	NoteASharp_1 MidiKey = MidiKey(ui.NoteASharp_1)
	NoteBFlat_1  MidiKey = MidiKey(ui.NoteBFlat_1)
	NoteB_1      MidiKey = MidiKey(ui.NoteB_1)
	NoteC0       MidiKey = MidiKey(ui.NoteC0)
	NoteCSharp0  MidiKey = MidiKey(ui.NoteCSharp0)
	NoteDFlat0   MidiKey = MidiKey(ui.NoteDFlat0)
	NoteD0       MidiKey = MidiKey(ui.NoteD0)
	NoteDSharp0  MidiKey = MidiKey(ui.NoteDSharp0)
	NoteEFlat0   MidiKey = MidiKey(ui.NoteEFlat0)
	NoteE0       MidiKey = MidiKey(ui.NoteE0)
	NoteF0       MidiKey = MidiKey(ui.NoteF0)
	NoteFSharp0  MidiKey = MidiKey(ui.NoteFSharp0)
	NoteGFlat0   MidiKey = MidiKey(ui.NoteGFlat0)
	NoteG0       MidiKey = MidiKey(ui.NoteG0)
	NoteGSharp0  MidiKey = MidiKey(ui.NoteGSharp0)
	NoteAFlat0   MidiKey = MidiKey(ui.NoteAFlat0)
	NoteA0       MidiKey = MidiKey(ui.NoteA0)
	NoteASharp0  MidiKey = MidiKey(ui.NoteASharp0)
	NoteBFlat0   MidiKey = MidiKey(ui.NoteBFlat0)
	NoteB0       MidiKey = MidiKey(ui.NoteB0)
	NoteC1       MidiKey = MidiKey(ui.NoteC1)
	NoteCSharp1  MidiKey = MidiKey(ui.NoteCSharp1)
	NoteDFlat1   MidiKey = MidiKey(ui.NoteDFlat1)
	NoteD1       MidiKey = MidiKey(ui.NoteD1)
	NoteDSharp1  MidiKey = MidiKey(ui.NoteDSharp1)
	NoteEFlat1   MidiKey = MidiKey(ui.NoteEFlat1)
	NoteE1       MidiKey = MidiKey(ui.NoteE1)
	NoteF1       MidiKey = MidiKey(ui.NoteF1)
	NoteFSharp1  MidiKey = MidiKey(ui.NoteFSharp1)
	NoteGFlat1   MidiKey = MidiKey(ui.NoteGFlat1)
	NoteG1       MidiKey = MidiKey(ui.NoteG1)
	NoteGSharp1  MidiKey = MidiKey(ui.NoteGSharp1)
	NoteAFlat1   MidiKey = MidiKey(ui.NoteAFlat1)
	NoteA1       MidiKey = MidiKey(ui.NoteA1)
	NoteASharp1  MidiKey = MidiKey(ui.NoteASharp1)
	NoteBFlat1   MidiKey = MidiKey(ui.NoteBFlat1)
	NoteB1       MidiKey = MidiKey(ui.NoteB1)
	NoteC2       MidiKey = MidiKey(ui.NoteC2)
	NoteCSharp2  MidiKey = MidiKey(ui.NoteCSharp2)
	NoteDFlat2   MidiKey = MidiKey(ui.NoteDFlat2)
	NoteD2       MidiKey = MidiKey(ui.NoteD2)
	NoteDSharp2  MidiKey = MidiKey(ui.NoteDSharp2)
	NoteEFlat2   MidiKey = MidiKey(ui.NoteEFlat2)
	NoteE2       MidiKey = MidiKey(ui.NoteE2)
	NoteF2       MidiKey = MidiKey(ui.NoteF2)
	NoteFSharp2  MidiKey = MidiKey(ui.NoteFSharp2)
	NoteGFlat2   MidiKey = MidiKey(ui.NoteGFlat2)
	NoteG2       MidiKey = MidiKey(ui.NoteG2)
	NoteGSharp2  MidiKey = MidiKey(ui.NoteGSharp2)
	NoteAFlat2   MidiKey = MidiKey(ui.NoteAFlat2)
	NoteA2       MidiKey = MidiKey(ui.NoteA2)
	NoteASharp2  MidiKey = MidiKey(ui.NoteASharp2)
	NoteBFlat2   MidiKey = MidiKey(ui.NoteBFlat2)
	NoteB2       MidiKey = MidiKey(ui.NoteB2)
	NoteC3       MidiKey = MidiKey(ui.NoteC3)
	NoteCSharp3  MidiKey = MidiKey(ui.NoteCSharp3)
	NoteDFlat3   MidiKey = MidiKey(ui.NoteDFlat3)
	NoteD3       MidiKey = MidiKey(ui.NoteD3)
	NoteDSharp3  MidiKey = MidiKey(ui.NoteDSharp3)
	NoteEFlat3   MidiKey = MidiKey(ui.NoteEFlat3)
	NoteE3       MidiKey = MidiKey(ui.NoteE3)
	NoteF3       MidiKey = MidiKey(ui.NoteF3)
	NoteFSharp3  MidiKey = MidiKey(ui.NoteFSharp3)
	NoteGFlat3   MidiKey = MidiKey(ui.NoteGFlat3)
	NoteG3       MidiKey = MidiKey(ui.NoteG3)
	NoteGSharp3  MidiKey = MidiKey(ui.NoteGSharp3)
	NoteAFlat3   MidiKey = MidiKey(ui.NoteAFlat3)
	NoteA3       MidiKey = MidiKey(ui.NoteA3)
	NoteASharp3  MidiKey = MidiKey(ui.NoteASharp3)
	NoteBFlat3   MidiKey = MidiKey(ui.NoteBFlat3)
	NoteB3       MidiKey = MidiKey(ui.NoteB3)
	NoteC4       MidiKey = MidiKey(ui.NoteC4)
	NoteCSharp4  MidiKey = MidiKey(ui.NoteCSharp4)
	NoteDFlat4   MidiKey = MidiKey(ui.NoteDFlat4)
	NoteD4       MidiKey = MidiKey(ui.NoteD4)
	NoteDSharp4  MidiKey = MidiKey(ui.NoteDSharp4)
	NoteEFlat4   MidiKey = MidiKey(ui.NoteEFlat4)
	NoteE4       MidiKey = MidiKey(ui.NoteE4)
	NoteF4       MidiKey = MidiKey(ui.NoteF4)
	NoteFSharp4  MidiKey = MidiKey(ui.NoteFSharp4)
	NoteGFlat4   MidiKey = MidiKey(ui.NoteGFlat4)
	NoteG4       MidiKey = MidiKey(ui.NoteG4)
	NoteGSharp4  MidiKey = MidiKey(ui.NoteGSharp4)
	NoteAFlat4   MidiKey = MidiKey(ui.NoteAFlat4)
	NoteA4       MidiKey = MidiKey(ui.NoteA4)
	NoteASharp4  MidiKey = MidiKey(ui.NoteASharp4)
	NoteBFlat4   MidiKey = MidiKey(ui.NoteBFlat4)
	NoteB4       MidiKey = MidiKey(ui.NoteB4)
	NoteC5       MidiKey = MidiKey(ui.NoteC5)
	NoteCSharp5  MidiKey = MidiKey(ui.NoteCSharp5)
	NoteDFlat5   MidiKey = MidiKey(ui.NoteDFlat5)
	NoteD5       MidiKey = MidiKey(ui.NoteD5)
	NoteDSharp5  MidiKey = MidiKey(ui.NoteDSharp5)
	NoteEFlat5   MidiKey = MidiKey(ui.NoteEFlat5)
	NoteE5       MidiKey = MidiKey(ui.NoteE5)
	NoteF5       MidiKey = MidiKey(ui.NoteF5)
	NoteFSharp5  MidiKey = MidiKey(ui.NoteFSharp5)
	NoteGFlat5   MidiKey = MidiKey(ui.NoteGFlat5)
	NoteG5       MidiKey = MidiKey(ui.NoteG5)
	NoteGSharp5  MidiKey = MidiKey(ui.NoteGSharp5)
	NoteAFlat5   MidiKey = MidiKey(ui.NoteAFlat5)
	NoteA5       MidiKey = MidiKey(ui.NoteA5)
	NoteASharp5  MidiKey = MidiKey(ui.NoteASharp5)
	NoteBFlat5   MidiKey = MidiKey(ui.NoteBFlat5)
	NoteB5       MidiKey = MidiKey(ui.NoteB5)
	NoteC6       MidiKey = MidiKey(ui.NoteC6)
	NoteCSharp6  MidiKey = MidiKey(ui.NoteCSharp6)
	NoteDFlat6   MidiKey = MidiKey(ui.NoteDFlat6)
	NoteD6       MidiKey = MidiKey(ui.NoteD6)
	NoteDSharp6  MidiKey = MidiKey(ui.NoteDSharp6)
	NoteEFlat6   MidiKey = MidiKey(ui.NoteEFlat6)
	NoteE6       MidiKey = MidiKey(ui.NoteE6)
	NoteF6       MidiKey = MidiKey(ui.NoteF6)
	NoteFSharp6  MidiKey = MidiKey(ui.NoteFSharp6)
	NoteGFlat6   MidiKey = MidiKey(ui.NoteGFlat6)
	NoteG6       MidiKey = MidiKey(ui.NoteG6)
	NoteGSharp6  MidiKey = MidiKey(ui.NoteGSharp6)
	NoteAFlat6   MidiKey = MidiKey(ui.NoteAFlat6)
	NoteA6       MidiKey = MidiKey(ui.NoteA6)
	NoteASharp6  MidiKey = MidiKey(ui.NoteASharp6)
	NoteBFlat6   MidiKey = MidiKey(ui.NoteBFlat6)
	NoteB6       MidiKey = MidiKey(ui.NoteB6)
	NoteC7       MidiKey = MidiKey(ui.NoteC7)
	NoteCSharp7  MidiKey = MidiKey(ui.NoteCSharp7)
	NoteDFlat7   MidiKey = MidiKey(ui.NoteDFlat7)
	NoteD7       MidiKey = MidiKey(ui.NoteD7)
	NoteDSharp7  MidiKey = MidiKey(ui.NoteDSharp7)
	NoteEFlat7   MidiKey = MidiKey(ui.NoteEFlat7)
	NoteE7       MidiKey = MidiKey(ui.NoteE7)
	NoteF7       MidiKey = MidiKey(ui.NoteF7)
	NoteFSharp7  MidiKey = MidiKey(ui.NoteFSharp7)
	NoteGFlat7   MidiKey = MidiKey(ui.NoteGFlat7)
	NoteG7       MidiKey = MidiKey(ui.NoteG7)
	NoteGSharp7  MidiKey = MidiKey(ui.NoteGSharp7)
	NoteAFlat7   MidiKey = MidiKey(ui.NoteAFlat7)
	NoteA7       MidiKey = MidiKey(ui.NoteA7)
	NoteASharp7  MidiKey = MidiKey(ui.NoteASharp7)
	NoteBFlat7   MidiKey = MidiKey(ui.NoteBFlat7)
	NoteB7       MidiKey = MidiKey(ui.NoteB7)
	NoteC8       MidiKey = MidiKey(ui.NoteC8)
	NoteCSharp8  MidiKey = MidiKey(ui.NoteCSharp8)
	NoteDFlat8   MidiKey = MidiKey(ui.NoteDFlat8)
	NoteD8       MidiKey = MidiKey(ui.NoteD8)
	NoteDSharp8  MidiKey = MidiKey(ui.NoteDSharp8)
	NoteEFlat8   MidiKey = MidiKey(ui.NoteEFlat8)
	NoteE8       MidiKey = MidiKey(ui.NoteE8)
	NoteF8       MidiKey = MidiKey(ui.NoteF8)
	NoteFSharp8  MidiKey = MidiKey(ui.NoteFSharp8)
	NoteGFlat8   MidiKey = MidiKey(ui.NoteGFlat8)
	NoteG8       MidiKey = MidiKey(ui.NoteG8)
	NoteGSharp8  MidiKey = MidiKey(ui.NoteGSharp8)
	NoteAFlat8   MidiKey = MidiKey(ui.NoteAFlat8)
	NoteA8       MidiKey = MidiKey(ui.NoteA8)
	NoteASharp8  MidiKey = MidiKey(ui.NoteASharp8)
	NoteBFlat8   MidiKey = MidiKey(ui.NoteBFlat8)
	NoteB8       MidiKey = MidiKey(ui.NoteB8)
	NoteC9       MidiKey = MidiKey(ui.NoteC9)
	NoteCSharp9  MidiKey = MidiKey(ui.NoteCSharp9)
	NoteDFlat9   MidiKey = MidiKey(ui.NoteDFlat9)
	NoteD9       MidiKey = MidiKey(ui.NoteD9)
	NoteDSharp9  MidiKey = MidiKey(ui.NoteDSharp9)
	NoteEFlat9   MidiKey = MidiKey(ui.NoteEFlat9)
	NoteE9       MidiKey = MidiKey(ui.NoteE9)
	NoteF9       MidiKey = MidiKey(ui.NoteF9)
	NoteFSharp9  MidiKey = MidiKey(ui.NoteFSharp9)
	NoteGFlat9   MidiKey = MidiKey(ui.NoteGFlat9)
	NoteG9       MidiKey = MidiKey(ui.NoteG9)
)

func (k MidiKey) isValid() bool {
	switch k {
	case NoteC_1:
		return true
	case NoteCSharp_1:
		return true
	case NoteDFlat_1:
		return true
	case NoteD_1:
		return true
	case NoteDSharp_1:
		return true
	case NoteEFlat_1:
		return true
	case NoteE_1:
		return true
	case NoteF_1:
		return true
	case NoteFSharp_1:
		return true
	case NoteGFlat_1:
		return true
	case NoteG_1:
		return true
	case NoteGSharp_1:
		return true
	case NoteAFlat_1:
		return true
	case NoteA_1:
		return true
	case NoteASharp_1:
		return true
	case NoteBFlat_1:
		return true
	case NoteB_1:
		return true
	case NoteC0:
		return true
	case NoteCSharp0:
		return true
	case NoteDFlat0:
		return true
	case NoteD0:
		return true
	case NoteDSharp0:
		return true
	case NoteEFlat0:
		return true
	case NoteE0:
		return true
	case NoteF0:
		return true
	case NoteFSharp0:
		return true
	case NoteGFlat0:
		return true
	case NoteG0:
		return true
	case NoteGSharp0:
		return true
	case NoteAFlat0:
		return true
	case NoteA0:
		return true
	case NoteASharp0:
		return true
	case NoteBFlat0:
		return true
	case NoteB0:
		return true
	case NoteC1:
		return true
	case NoteCSharp1:
		return true
	case NoteDFlat1:
		return true
	case NoteD1:
		return true
	case NoteDSharp1:
		return true
	case NoteEFlat1:
		return true
	case NoteE1:
		return true
	case NoteF1:
		return true
	case NoteFSharp1:
		return true
	case NoteGFlat1:
		return true
	case NoteG1:
		return true
	case NoteGSharp1:
		return true
	case NoteAFlat1:
		return true
	case NoteA1:
		return true
	case NoteASharp1:
		return true
	case NoteBFlat1:
		return true
	case NoteB1:
		return true
	case NoteC2:
		return true
	case NoteCSharp2:
		return true
	case NoteDFlat2:
		return true
	case NoteD2:
		return true
	case NoteDSharp2:
		return true
	case NoteEFlat2:
		return true
	case NoteE2:
		return true
	case NoteF2:
		return true
	case NoteFSharp2:
		return true
	case NoteGFlat2:
		return true
	case NoteG2:
		return true
	case NoteGSharp2:
		return true
	case NoteAFlat2:
		return true
	case NoteA2:
		return true
	case NoteASharp2:
		return true
	case NoteBFlat2:
		return true
	case NoteB2:
		return true
	case NoteC3:
		return true
	case NoteCSharp3:
		return true
	case NoteDFlat3:
		return true
	case NoteD3:
		return true
	case NoteDSharp3:
		return true
	case NoteEFlat3:
		return true
	case NoteE3:
		return true
	case NoteF3:
		return true
	case NoteFSharp3:
		return true
	case NoteGFlat3:
		return true
	case NoteG3:
		return true
	case NoteGSharp3:
		return true
	case NoteAFlat3:
		return true
	case NoteA3:
		return true
	case NoteASharp3:
		return true
	case NoteBFlat3:
		return true
	case NoteB3:
		return true
	case NoteC4:
		return true
	case NoteCSharp4:
		return true
	case NoteDFlat4:
		return true
	case NoteD4:
		return true
	case NoteDSharp4:
		return true
	case NoteEFlat4:
		return true
	case NoteE4:
		return true
	case NoteF4:
		return true
	case NoteFSharp4:
		return true
	case NoteGFlat4:
		return true
	case NoteG4:
		return true
	case NoteGSharp4:
		return true
	case NoteAFlat4:
		return true
	case NoteA4:
		return true
	case NoteASharp4:
		return true
	case NoteBFlat4:
		return true
	case NoteB4:
		return true
	case NoteC5:
		return true
	case NoteCSharp5:
		return true
	case NoteDFlat5:
		return true
	case NoteD5:
		return true
	case NoteDSharp5:
		return true
	case NoteEFlat5:
		return true
	case NoteE5:
		return true
	case NoteF5:
		return true
	case NoteFSharp5:
		return true
	case NoteGFlat5:
		return true
	case NoteG5:
		return true
	case NoteGSharp5:
		return true
	case NoteAFlat5:
		return true
	case NoteA5:
		return true
	case NoteASharp5:
		return true
	case NoteBFlat5:
		return true
	case NoteB5:
		return true
	case NoteC6:
		return true
	case NoteCSharp6:
		return true
	case NoteDFlat6:
		return true
	case NoteD6:
		return true
	case NoteDSharp6:
		return true
	case NoteEFlat6:
		return true
	case NoteE6:
		return true
	case NoteF6:
		return true
	case NoteFSharp6:
		return true
	case NoteGFlat6:
		return true
	case NoteG6:
		return true
	case NoteGSharp6:
		return true
	case NoteAFlat6:
		return true
	case NoteA6:
		return true
	case NoteASharp6:
		return true
	case NoteBFlat6:
		return true
	case NoteB6:
		return true
	case NoteC7:
		return true
	case NoteCSharp7:
		return true
	case NoteDFlat7:
		return true
	case NoteD7:
		return true
	case NoteDSharp7:
		return true
	case NoteEFlat7:
		return true
	case NoteE7:
		return true
	case NoteF7:
		return true
	case NoteFSharp7:
		return true
	case NoteGFlat7:
		return true
	case NoteG7:
		return true
	case NoteGSharp7:
		return true
	case NoteAFlat7:
		return true
	case NoteA7:
		return true
	case NoteASharp7:
		return true
	case NoteBFlat7:
		return true
	case NoteB7:
		return true
	case NoteC8:
		return true
	case NoteCSharp8:
		return true
	case NoteDFlat8:
		return true
	case NoteD8:
		return true
	case NoteDSharp8:
		return true
	case NoteEFlat8:
		return true
	case NoteE8:
		return true
	case NoteF8:
		return true
	case NoteFSharp8:
		return true
	case NoteGFlat8:
		return true
	case NoteG8:
		return true
	case NoteGSharp8:
		return true
	case NoteAFlat8:
		return true
	case NoteA8:
		return true
	case NoteASharp8:
		return true
	case NoteBFlat8:
		return true
	case NoteB8:
		return true
	case NoteC9:
		return true
	case NoteCSharp9:
		return true
	case NoteDFlat9:
		return true
	case NoteD9:
		return true
	case NoteDSharp9:
		return true
	case NoteEFlat9:
		return true
	case NoteE9:
		return true
	case NoteF9:
		return true
	case NoteFSharp9:
		return true
	case NoteGFlat9:
		return true
	case NoteG9:
		return true

	default:
		return false
	}
}

// String returns a string representing the midi key.
//
// If k is an undefined key, String returns an empty string.
func (k MidiKey) String() string {
	switch k {
	case NoteC_1:
		return "C-1"
	case NoteCSharp_1:
		return "C#-1"
	case NoteDFlat_1:
		return "Db-1"
	case NoteD_1:
		return "D-1"
	case NoteDSharp_1:
		return "D#-1"
	case NoteEFlat_1:
		return "Eb-1"
	case NoteE_1:
		return "E-1"
	case NoteF_1:
		return "F-1"
	case NoteFSharp_1:
		return "F#-1"
	case NoteGFlat_1:
		return "Gb-1"
	case NoteG_1:
		return "G-1"
	case NoteGSharp_1:
		return "G#-1"
	case NoteAFlat_1:
		return "Ab-1"
	case NoteA_1:
		return "A-1"
	case NoteASharp_1:
		return "A#-1"
	case NoteBFlat_1:
		return "Bb-1"
	case NoteB_1:
		return "B-1"
	case NoteC0:
		return "C0"
	case NoteCSharp0:
		return "C#0"
	case NoteDFlat0:
		return "Db0"
	case NoteD0:
		return "D0"
	case NoteDSharp0:
		return "D#0"
	case NoteEFlat0:
		return "Eb0"
	case NoteE0:
		return "E0"
	case NoteF0:
		return "F0"
	case NoteFSharp0:
		return "F#0"
	case NoteGFlat0:
		return "Gb0"
	case NoteG0:
		return "G0"
	case NoteGSharp0:
		return "G#0"
	case NoteAFlat0:
		return "Ab0"
	case NoteA0:
		return "A0"
	case NoteASharp0:
		return "A#0"
	case NoteBFlat0:
		return "Bb0"
	case NoteB0:
		return "B0"
	case NoteC1:
		return "C1"
	case NoteCSharp1:
		return "C#1"
	case NoteDFlat1:
		return "Db1"
	case NoteD1:
		return "D1"
	case NoteDSharp1:
		return "D#1"
	case NoteEFlat1:
		return "Eb1"
	case NoteE1:
		return "E1"
	case NoteF1:
		return "F1"
	case NoteFSharp1:
		return "F#1"
	case NoteGFlat1:
		return "Gb1"
	case NoteG1:
		return "G1"
	case NoteGSharp1:
		return "G#1"
	case NoteAFlat1:
		return "Ab1"
	case NoteA1:
		return "A1"
	case NoteASharp1:
		return "A#1"
	case NoteBFlat1:
		return "Bb1"
	case NoteB1:
		return "B1"
	case NoteC2:
		return "C2"
	case NoteCSharp2:
		return "C#2"
	case NoteDFlat2:
		return "Db2"
	case NoteD2:
		return "D2"
	case NoteDSharp2:
		return "D#2"
	case NoteEFlat2:
		return "Eb2"
	case NoteE2:
		return "E2"
	case NoteF2:
		return "F2"
	case NoteFSharp2:
		return "F#2"
	case NoteGFlat2:
		return "Gb2"
	case NoteG2:
		return "G2"
	case NoteGSharp2:
		return "G#2"
	case NoteAFlat2:
		return "Ab2"
	case NoteA2:
		return "A2"
	case NoteASharp2:
		return "A#2"
	case NoteBFlat2:
		return "Bb2"
	case NoteB2:
		return "B2"
	case NoteC3:
		return "C3"
	case NoteCSharp3:
		return "C#3"
	case NoteDFlat3:
		return "Db3"
	case NoteD3:
		return "D3"
	case NoteDSharp3:
		return "D#3"
	case NoteEFlat3:
		return "Eb3"
	case NoteE3:
		return "E3"
	case NoteF3:
		return "F3"
	case NoteFSharp3:
		return "F#3"
	case NoteGFlat3:
		return "Gb3"
	case NoteG3:
		return "G3"
	case NoteGSharp3:
		return "G#3"
	case NoteAFlat3:
		return "Ab3"
	case NoteA3:
		return "A3"
	case NoteASharp3:
		return "A#3"
	case NoteBFlat3:
		return "Bb3"
	case NoteB3:
		return "B3"
	case NoteC4:
		return "C4"
	case NoteCSharp4:
		return "C#4"
	case NoteDFlat4:
		return "Db4"
	case NoteD4:
		return "D4"
	case NoteDSharp4:
		return "D#4"
	case NoteEFlat4:
		return "Eb4"
	case NoteE4:
		return "E4"
	case NoteF4:
		return "F4"
	case NoteFSharp4:
		return "F#4"
	case NoteGFlat4:
		return "Gb4"
	case NoteG4:
		return "G4"
	case NoteGSharp4:
		return "G#4"
	case NoteAFlat4:
		return "Ab4"
	case NoteA4:
		return "A4"
	case NoteASharp4:
		return "A#4"
	case NoteBFlat4:
		return "Bb4"
	case NoteB4:
		return "B4"
	case NoteC5:
		return "C5"
	case NoteCSharp5:
		return "C#5"
	case NoteDFlat5:
		return "Db5"
	case NoteD5:
		return "D5"
	case NoteDSharp5:
		return "D#5"
	case NoteEFlat5:
		return "Eb5"
	case NoteE5:
		return "E5"
	case NoteF5:
		return "F5"
	case NoteFSharp5:
		return "F#5"
	case NoteGFlat5:
		return "Gb5"
	case NoteG5:
		return "G5"
	case NoteGSharp5:
		return "G#5"
	case NoteAFlat5:
		return "Ab5"
	case NoteA5:
		return "A5"
	case NoteASharp5:
		return "A#5"
	case NoteBFlat5:
		return "Bb5"
	case NoteB5:
		return "B5"
	case NoteC6:
		return "C6"
	case NoteCSharp6:
		return "C#6"
	case NoteDFlat6:
		return "Db6"
	case NoteD6:
		return "D6"
	case NoteDSharp6:
		return "D#6"
	case NoteEFlat6:
		return "Eb6"
	case NoteE6:
		return "E6"
	case NoteF6:
		return "F6"
	case NoteFSharp6:
		return "F#6"
	case NoteGFlat6:
		return "Gb6"
	case NoteG6:
		return "G6"
	case NoteGSharp6:
		return "G#6"
	case NoteAFlat6:
		return "Ab6"
	case NoteA6:
		return "A6"
	case NoteASharp6:
		return "A#6"
	case NoteBFlat6:
		return "Bb6"
	case NoteB6:
		return "B6"
	case NoteC7:
		return "C7"
	case NoteCSharp7:
		return "C#7"
	case NoteDFlat7:
		return "Db7"
	case NoteD7:
		return "D7"
	case NoteDSharp7:
		return "D#7"
	case NoteEFlat7:
		return "Eb7"
	case NoteE7:
		return "E7"
	case NoteF7:
		return "F7"
	case NoteFSharp7:
		return "F#7"
	case NoteGFlat7:
		return "Gb7"
	case NoteG7:
		return "G7"
	case NoteGSharp7:
		return "G#7"
	case NoteAFlat7:
		return "Ab7"
	case NoteA7:
		return "A7"
	case NoteASharp7:
		return "A#7"
	case NoteBFlat7:
		return "Bb7"
	case NoteB7:
		return "B7"
	case NoteC8:
		return "C8"
	case NoteCSharp8:
		return "C#8"
	case NoteDFlat8:
		return "Db8"
	case NoteD8:
		return "D8"
	case NoteDSharp8:
		return "D#8"
	case NoteEFlat8:
		return "Eb8"
	case NoteE8:
		return "E8"
	case NoteF8:
		return "F8"
	case NoteFSharp8:
		return "F#8"
	case NoteGFlat8:
		return "Gb8"
	case NoteG8:
		return "G8"
	case NoteGSharp8:
		return "G#8"
	case NoteAFlat8:
		return "Ab8"
	case NoteA8:
		return "A8"
	case NoteASharp8:
		return "A#8"
	case NoteBFlat8:
		return "Bb8"
	case NoteB8:
		return "B8"
	case NoteC9:
		return "C9"
	case NoteCSharp9:
		return "C#9"
	case NoteDFlat9:
		return "Db9"
	case NoteD9:
		return "D9"
	case NoteDSharp9:
		return "D#9"
	case NoteEFlat9:
		return "Eb9"
	case NoteE9:
		return "E9"
	case NoteF9:
		return "F9"
	case NoteFSharp9:
		return "F#9"
	case NoteGFlat9:
		return "Gb9"
	case NoteG9:
		return "G9"
	}
	return ""
}

func midiKeyNameToMidiKey(name string) (MidiKey, bool) {
	switch strings.ToLower(name) {
	case "c-1", "notec_1":
		return NoteC_1, true
	case "c#-1", "notecsharp_1":
		return NoteCSharp_1, true
	case "db-1", "notedflat_1":
		return NoteDFlat_1, true
	case "d-1", "noted_1":
		return NoteD_1, true
	case "d#-1", "notedsharp_1":
		return NoteDSharp_1, true
	case "eb-1", "noteeflat_1":
		return NoteEFlat_1, true
	case "e-1", "notee_1":
		return NoteE_1, true
	case "f-1", "notef_1":
		return NoteF_1, true
	case "f#-1", "notefsharp_1":
		return NoteFSharp_1, true
	case "gb-1", "notegflat_1":
		return NoteGFlat_1, true
	case "g-1", "noteg_1":
		return NoteG_1, true
	case "g#-1", "notegsharp_1":
		return NoteGSharp_1, true
	case "ab-1", "noteaflat_1":
		return NoteAFlat_1, true
	case "a-1", "notea_1":
		return NoteA_1, true
	case "a#-1", "noteasharp_1":
		return NoteASharp_1, true
	case "bb-1", "notebflat_1":
		return NoteBFlat_1, true
	case "b-1", "noteb_1":
		return NoteB_1, true
	case "c0", "notec0":
		return NoteC0, true
	case "c#0", "notecsharp0":
		return NoteCSharp0, true
	case "db0", "notedflat0":
		return NoteDFlat0, true
	case "d0", "noted0":
		return NoteD0, true
	case "d#0", "notedsharp0":
		return NoteDSharp0, true
	case "eb0", "noteeflat0":
		return NoteEFlat0, true
	case "e0", "notee0":
		return NoteE0, true
	case "f0", "notef0":
		return NoteF0, true
	case "f#0", "notefsharp0":
		return NoteFSharp0, true
	case "gb0", "notegflat0":
		return NoteGFlat0, true
	case "g0", "noteg0":
		return NoteG0, true
	case "g#0", "notegsharp0":
		return NoteGSharp0, true
	case "ab0", "noteaflat0":
		return NoteAFlat0, true
	case "a0", "notea0":
		return NoteA0, true
	case "a#0", "noteasharp0":
		return NoteASharp0, true
	case "bb0", "notebflat0":
		return NoteBFlat0, true
	case "b0", "noteb0":
		return NoteB0, true
	case "c1", "notec1":
		return NoteC1, true
	case "c#1", "notecsharp1":
		return NoteCSharp1, true
	case "db1", "notedflat1":
		return NoteDFlat1, true
	case "d1", "noted1":
		return NoteD1, true
	case "d#1", "notedsharp1":
		return NoteDSharp1, true
	case "eb1", "noteeflat1":
		return NoteEFlat1, true
	case "e1", "notee1":
		return NoteE1, true
	case "f1", "notef1":
		return NoteF1, true
	case "f#1", "notefsharp1":
		return NoteFSharp1, true
	case "gb1", "notegflat1":
		return NoteGFlat1, true
	case "g1", "noteg1":
		return NoteG1, true
	case "g#1", "notegsharp1":
		return NoteGSharp1, true
	case "ab1", "noteaflat1":
		return NoteAFlat1, true
	case "a1", "notea1":
		return NoteA1, true
	case "a#1", "noteasharp1":
		return NoteASharp1, true
	case "bb1", "notebflat1":
		return NoteBFlat1, true
	case "b1", "noteb1":
		return NoteB1, true
	case "c2", "notec2":
		return NoteC2, true
	case "c#2", "notecsharp2":
		return NoteCSharp2, true
	case "db2", "notedflat2":
		return NoteDFlat2, true
	case "d2", "noted2":
		return NoteD2, true
	case "d#2", "notedsharp2":
		return NoteDSharp2, true
	case "eb2", "noteeflat2":
		return NoteEFlat2, true
	case "e2", "notee2":
		return NoteE2, true
	case "f2", "notef2":
		return NoteF2, true
	case "f#2", "notefsharp2":
		return NoteFSharp2, true
	case "gb2", "notegflat2":
		return NoteGFlat2, true
	case "g2", "noteg2":
		return NoteG2, true
	case "g#2", "notegsharp2":
		return NoteGSharp2, true
	case "ab2", "noteaflat2":
		return NoteAFlat2, true
	case "a2", "notea2":
		return NoteA2, true
	case "a#2", "noteasharp2":
		return NoteASharp2, true
	case "bb2", "notebflat2":
		return NoteBFlat2, true
	case "b2", "noteb2":
		return NoteB2, true
	case "c3", "notec3":
		return NoteC3, true
	case "c#3", "notecsharp3":
		return NoteCSharp3, true
	case "db3", "notedflat3":
		return NoteDFlat3, true
	case "d3", "noted3":
		return NoteD3, true
	case "d#3", "notedsharp3":
		return NoteDSharp3, true
	case "eb3", "noteeflat3":
		return NoteEFlat3, true
	case "e3", "notee3":
		return NoteE3, true
	case "f3", "notef3":
		return NoteF3, true
	case "f#3", "notefsharp3":
		return NoteFSharp3, true
	case "gb3", "notegflat3":
		return NoteGFlat3, true
	case "g3", "noteg3":
		return NoteG3, true
	case "g#3", "notegsharp3":
		return NoteGSharp3, true
	case "ab3", "noteaflat3":
		return NoteAFlat3, true
	case "a3", "notea3":
		return NoteA3, true
	case "a#3", "noteasharp3":
		return NoteASharp3, true
	case "bb3", "notebflat3":
		return NoteBFlat3, true
	case "b3", "noteb3":
		return NoteB3, true
	case "c4", "notec4":
		return NoteC4, true
	case "c#4", "notecsharp4":
		return NoteCSharp4, true
	case "db4", "notedflat4":
		return NoteDFlat4, true
	case "d4", "noted4":
		return NoteD4, true
	case "d#4", "notedsharp4":
		return NoteDSharp4, true
	case "eb4", "noteeflat4":
		return NoteEFlat4, true
	case "e4", "notee4":
		return NoteE4, true
	case "f4", "notef4":
		return NoteF4, true
	case "f#4", "notefsharp4":
		return NoteFSharp4, true
	case "gb4", "notegflat4":
		return NoteGFlat4, true
	case "g4", "noteg4":
		return NoteG4, true
	case "g#4", "notegsharp4":
		return NoteGSharp4, true
	case "ab4", "noteaflat4":
		return NoteAFlat4, true
	case "a4", "notea4":
		return NoteA4, true
	case "a#4", "noteasharp4":
		return NoteASharp4, true
	case "bb4", "notebflat4":
		return NoteBFlat4, true
	case "b4", "noteb4":
		return NoteB4, true
	case "c5", "notec5":
		return NoteC5, true
	case "c#5", "notecsharp5":
		return NoteCSharp5, true
	case "db5", "notedflat5":
		return NoteDFlat5, true
	case "d5", "noted5":
		return NoteD5, true
	case "d#5", "notedsharp5":
		return NoteDSharp5, true
	case "eb5", "noteeflat5":
		return NoteEFlat5, true
	case "e5", "notee5":
		return NoteE5, true
	case "f5", "notef5":
		return NoteF5, true
	case "f#5", "notefsharp5":
		return NoteFSharp5, true
	case "gb5", "notegflat5":
		return NoteGFlat5, true
	case "g5", "noteg5":
		return NoteG5, true
	case "g#5", "notegsharp5":
		return NoteGSharp5, true
	case "ab5", "noteaflat5":
		return NoteAFlat5, true
	case "a5", "notea5":
		return NoteA5, true
	case "a#5", "noteasharp5":
		return NoteASharp5, true
	case "bb5", "notebflat5":
		return NoteBFlat5, true
	case "b5", "noteb5":
		return NoteB5, true
	case "c6", "notec6":
		return NoteC6, true
	case "c#6", "notecsharp6":
		return NoteCSharp6, true
	case "db6", "notedflat6":
		return NoteDFlat6, true
	case "d6", "noted6":
		return NoteD6, true
	case "d#6", "notedsharp6":
		return NoteDSharp6, true
	case "eb6", "noteeflat6":
		return NoteEFlat6, true
	case "e6", "notee6":
		return NoteE6, true
	case "f6", "notef6":
		return NoteF6, true
	case "f#6", "notefsharp6":
		return NoteFSharp6, true
	case "gb6", "notegflat6":
		return NoteGFlat6, true
	case "g6", "noteg6":
		return NoteG6, true
	case "g#6", "notegsharp6":
		return NoteGSharp6, true
	case "ab6", "noteaflat6":
		return NoteAFlat6, true
	case "a6", "notea6":
		return NoteA6, true
	case "a#6", "noteasharp6":
		return NoteASharp6, true
	case "bb6", "notebflat6":
		return NoteBFlat6, true
	case "b6", "noteb6":
		return NoteB6, true
	case "c7", "notec7":
		return NoteC7, true
	case "c#7", "notecsharp7":
		return NoteCSharp7, true
	case "db7", "notedflat7":
		return NoteDFlat7, true
	case "d7", "noted7":
		return NoteD7, true
	case "d#7", "notedsharp7":
		return NoteDSharp7, true
	case "eb7", "noteeflat7":
		return NoteEFlat7, true
	case "e7", "notee7":
		return NoteE7, true
	case "f7", "notef7":
		return NoteF7, true
	case "f#7", "notefsharp7":
		return NoteFSharp7, true
	case "gb7", "notegflat7":
		return NoteGFlat7, true
	case "g7", "noteg7":
		return NoteG7, true
	case "g#7", "notegsharp7":
		return NoteGSharp7, true
	case "ab7", "noteaflat7":
		return NoteAFlat7, true
	case "a7", "notea7":
		return NoteA7, true
	case "a#7", "noteasharp7":
		return NoteASharp7, true
	case "bb7", "notebflat7":
		return NoteBFlat7, true
	case "b7", "noteb7":
		return NoteB7, true
	case "c8", "notec8":
		return NoteC8, true
	case "c#8", "notecsharp8":
		return NoteCSharp8, true
	case "db8", "notedflat8":
		return NoteDFlat8, true
	case "d8", "noted8":
		return NoteD8, true
	case "d#8", "notedsharp8":
		return NoteDSharp8, true
	case "eb8", "noteeflat8":
		return NoteEFlat8, true
	case "e8", "notee8":
		return NoteE8, true
	case "f8", "notef8":
		return NoteF8, true
	case "f#8", "notefsharp8":
		return NoteFSharp8, true
	case "gb8", "notegflat8":
		return NoteGFlat8, true
	case "g8", "noteg8":
		return NoteG8, true
	case "g#8", "notegsharp8":
		return NoteGSharp8, true
	case "ab8", "noteaflat8":
		return NoteAFlat8, true
	case "a8", "notea8":
		return NoteA8, true
	case "a#8", "noteasharp8":
		return NoteASharp8, true
	case "bb8", "notebflat8":
		return NoteBFlat8, true
	case "b8", "noteb8":
		return NoteB8, true
	case "c9", "notec9":
		return NoteC9, true
	case "c#9", "notecsharp9":
		return NoteCSharp9, true
	case "db9", "notedflat9":
		return NoteDFlat9, true
	case "d9", "noted9":
		return NoteD9, true
	case "d#9", "notedsharp9":
		return NoteDSharp9, true
	case "eb9", "noteeflat9":
		return NoteEFlat9, true
	case "e9", "notee9":
		return NoteE9, true
	case "f9", "notef9":
		return NoteF9, true
	case "f#9", "notefsharp9":
		return NoteFSharp9, true
	case "gb9", "notegflat9":
		return NoteGFlat9, true
	case "g9", "noteg9":
		return NoteG9, true
	}
	return 0, false
}
