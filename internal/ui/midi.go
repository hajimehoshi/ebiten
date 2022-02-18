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

package ui

import (
	"fmt"
)

type MidiKey uint8

const (
	NoteC_1      MidiKey = 0
	NoteCSharp_1 MidiKey = 1
	NoteDFlat_1  MidiKey = 1
	NoteD_1      MidiKey = 2
	NoteDSharp_1 MidiKey = 3
	NoteEFlat_1  MidiKey = 3
	NoteE_1      MidiKey = 4
	NoteF_1      MidiKey = 5
	NoteFSharp_1 MidiKey = 6
	NoteGFlat_1  MidiKey = 6
	NoteG_1      MidiKey = 7
	NoteGSharp_1 MidiKey = 8
	NoteAFlat_1  MidiKey = 8
	NoteA_1      MidiKey = 9
	NoteASharp_1 MidiKey = 10
	NoteBFlat_1  MidiKey = 10
	NoteB_1      MidiKey = 11
	NoteC0       MidiKey = 12
	NoteCSharp0  MidiKey = 13
	NoteDFlat0   MidiKey = 13
	NoteD0       MidiKey = 14
	NoteDSharp0  MidiKey = 15
	NoteEFlat0   MidiKey = 15
	NoteE0       MidiKey = 16
	NoteF0       MidiKey = 17
	NoteFSharp0  MidiKey = 18
	NoteGFlat0   MidiKey = 18
	NoteG0       MidiKey = 19
	NoteGSharp0  MidiKey = 20
	NoteAFlat0   MidiKey = 20
	NoteA0       MidiKey = 21
	NoteASharp0  MidiKey = 22
	NoteBFlat0   MidiKey = 22
	NoteB0       MidiKey = 23
	NoteC1       MidiKey = 24
	NoteCSharp1  MidiKey = 25
	NoteDFlat1   MidiKey = 25
	NoteD1       MidiKey = 26
	NoteDSharp1  MidiKey = 27
	NoteEFlat1   MidiKey = 27
	NoteE1       MidiKey = 28
	NoteF1       MidiKey = 29
	NoteFSharp1  MidiKey = 30
	NoteGFlat1   MidiKey = 30
	NoteG1       MidiKey = 31
	NoteGSharp1  MidiKey = 32
	NoteAFlat1   MidiKey = 32
	NoteA1       MidiKey = 33
	NoteASharp1  MidiKey = 34
	NoteBFlat1   MidiKey = 34
	NoteB1       MidiKey = 35
	NoteC2       MidiKey = 36
	NoteCSharp2  MidiKey = 37
	NoteDFlat2   MidiKey = 37
	NoteD2       MidiKey = 38
	NoteDSharp2  MidiKey = 39
	NoteEFlat2   MidiKey = 39
	NoteE2       MidiKey = 40
	NoteF2       MidiKey = 41
	NoteFSharp2  MidiKey = 42
	NoteGFlat2   MidiKey = 42
	NoteG2       MidiKey = 43
	NoteGSharp2  MidiKey = 44
	NoteAFlat2   MidiKey = 44
	NoteA2       MidiKey = 45
	NoteASharp2  MidiKey = 46
	NoteBFlat2   MidiKey = 46
	NoteB2       MidiKey = 47
	NoteC3       MidiKey = 48
	NoteCSharp3  MidiKey = 49
	NoteDFlat3   MidiKey = 49
	NoteD3       MidiKey = 50
	NoteDSharp3  MidiKey = 51
	NoteEFlat3   MidiKey = 51
	NoteE3       MidiKey = 52
	NoteF3       MidiKey = 53
	NoteFSharp3  MidiKey = 54
	NoteGFlat3   MidiKey = 54
	NoteG3       MidiKey = 55
	NoteGSharp3  MidiKey = 56
	NoteAFlat3   MidiKey = 56
	NoteA3       MidiKey = 57
	NoteASharp3  MidiKey = 58
	NoteBFlat3   MidiKey = 58
	NoteB3       MidiKey = 59
	NoteC4       MidiKey = 60
	NoteCSharp4  MidiKey = 61
	NoteDFlat4   MidiKey = 61
	NoteD4       MidiKey = 62
	NoteDSharp4  MidiKey = 63
	NoteEFlat4   MidiKey = 63
	NoteE4       MidiKey = 64
	NoteF4       MidiKey = 65
	NoteFSharp4  MidiKey = 66
	NoteGFlat4   MidiKey = 66
	NoteG4       MidiKey = 67
	NoteGSharp4  MidiKey = 68
	NoteAFlat4   MidiKey = 68
	NoteA4       MidiKey = 69
	NoteASharp4  MidiKey = 70
	NoteBFlat4   MidiKey = 70
	NoteB4       MidiKey = 71
	NoteC5       MidiKey = 72
	NoteCSharp5  MidiKey = 73
	NoteDFlat5   MidiKey = 73
	NoteD5       MidiKey = 74
	NoteDSharp5  MidiKey = 75
	NoteEFlat5   MidiKey = 75
	NoteE5       MidiKey = 76
	NoteF5       MidiKey = 77
	NoteFSharp5  MidiKey = 78
	NoteGFlat5   MidiKey = 78
	NoteG5       MidiKey = 79
	NoteGSharp5  MidiKey = 80
	NoteAFlat5   MidiKey = 80
	NoteA5       MidiKey = 81
	NoteASharp5  MidiKey = 82
	NoteBFlat5   MidiKey = 82
	NoteB5       MidiKey = 83
	NoteC6       MidiKey = 84
	NoteCSharp6  MidiKey = 85
	NoteDFlat6   MidiKey = 85
	NoteD6       MidiKey = 86
	NoteDSharp6  MidiKey = 87
	NoteEFlat6   MidiKey = 87
	NoteE6       MidiKey = 88
	NoteF6       MidiKey = 89
	NoteFSharp6  MidiKey = 90
	NoteGFlat6   MidiKey = 90
	NoteG6       MidiKey = 91
	NoteGSharp6  MidiKey = 92
	NoteAFlat6   MidiKey = 92
	NoteA6       MidiKey = 93
	NoteASharp6  MidiKey = 94
	NoteBFlat6   MidiKey = 94
	NoteB6       MidiKey = 95
	NoteC7       MidiKey = 96
	NoteCSharp7  MidiKey = 97
	NoteDFlat7   MidiKey = 97
	NoteD7       MidiKey = 98
	NoteDSharp7  MidiKey = 99
	NoteEFlat7   MidiKey = 99
	NoteE7       MidiKey = 100
	NoteF7       MidiKey = 101
	NoteFSharp7  MidiKey = 102
	NoteGFlat7   MidiKey = 102
	NoteG7       MidiKey = 103
	NoteGSharp7  MidiKey = 104
	NoteAFlat7   MidiKey = 104
	NoteA7       MidiKey = 105
	NoteASharp7  MidiKey = 106
	NoteBFlat7   MidiKey = 106
	NoteB7       MidiKey = 107
	NoteC8       MidiKey = 108
	NoteCSharp8  MidiKey = 109
	NoteDFlat8   MidiKey = 109
	NoteD8       MidiKey = 110
	NoteDSharp8  MidiKey = 111
	NoteEFlat8   MidiKey = 111
	NoteE8       MidiKey = 112
	NoteF8       MidiKey = 113
	NoteFSharp8  MidiKey = 114
	NoteGFlat8   MidiKey = 114
	NoteG8       MidiKey = 115
	NoteGSharp8  MidiKey = 116
	NoteAFlat8   MidiKey = 116
	NoteA8       MidiKey = 117
	NoteASharp8  MidiKey = 118
	NoteBFlat8   MidiKey = 118
	NoteB8       MidiKey = 119
	NoteC9       MidiKey = 120
	NoteCSharp9  MidiKey = 121
	NoteDFlat9   MidiKey = 121
	NoteD9       MidiKey = 122
	NoteDSharp9  MidiKey = 123
	NoteEFlat9   MidiKey = 123
	NoteE9       MidiKey = 124
	NoteF9       MidiKey = 125
	NoteFSharp9  MidiKey = 126
	NoteGFlat9   MidiKey = 126
	NoteG9       MidiKey = 127
)

func (k Key) String() string {
	switch k {
	case NoteC_1:
		return "NoteC_1"
	case NoteCSharp_1:
		return "NoteCSharp_1"
	case NoteDFlat_1:
		return "NoteDFlat_1"
	case NoteD_1:
		return "NoteD_1"
	case NoteDSharp_1:
		return "NoteDSharp_1"
	case NoteEFlat_1:
		return "NoteEFlat_1"
	case NoteE_1:
		return "NoteE_1"
	case NoteF_1:
		return "NoteF_1"
	case NoteFSharp_1:
		return "NoteFSharp_1"
	case NoteGFlat_1:
		return "NoteGFlat_1"
	case NoteG_1:
		return "NoteG_1"
	case NoteGSharp_1:
		return "NoteGSharp_1"
	case NoteAFlat_1:
		return "NoteAFlat_1"
	case NoteA_1:
		return "NoteA_1"
	case NoteASharp_1:
		return "NoteASharp_1"
	case NoteBFlat_1:
		return "NoteBFlat_1"
	case NoteB_1:
		return "NoteB_1"
	case NoteC0:
		return "NoteC0"
	case NoteCSharp0:
		return "NoteCSharp0"
	case NoteDFlat0:
		return "NoteDFlat0"
	case NoteD0:
		return "NoteD0"
	case NoteDSharp0:
		return "NoteDSharp0"
	case NoteEFlat0:
		return "NoteEFlat0"
	case NoteE0:
		return "NoteE0"
	case NoteF0:
		return "NoteF0"
	case NoteFSharp0:
		return "NoteFSharp0"
	case NoteGFlat0:
		return "NoteGFlat0"
	case NoteG0:
		return "NoteG0"
	case NoteGSharp0:
		return "NoteGSharp0"
	case NoteAFlat0:
		return "NoteAFlat0"
	case NoteA0:
		return "NoteA0"
	case NoteASharp0:
		return "NoteASharp0"
	case NoteBFlat0:
		return "NoteBFlat0"
	case NoteB0:
		return "NoteB0"
	case NoteC1:
		return "NoteC1"
	case NoteCSharp1:
		return "NoteCSharp1"
	case NoteDFlat1:
		return "NoteDFlat1"
	case NoteD1:
		return "NoteD1"
	case NoteDSharp1:
		return "NoteDSharp1"
	case NoteEFlat1:
		return "NoteEFlat1"
	case NoteE1:
		return "NoteE1"
	case NoteF1:
		return "NoteF1"
	case NoteFSharp1:
		return "NoteFSharp1"
	case NoteGFlat1:
		return "NoteGFlat1"
	case NoteG1:
		return "NoteG1"
	case NoteGSharp1:
		return "NoteGSharp1"
	case NoteAFlat1:
		return "NoteAFlat1"
	case NoteA1:
		return "NoteA1"
	case NoteASharp1:
		return "NoteASharp1"
	case NoteBFlat1:
		return "NoteBFlat1"
	case NoteB1:
		return "NoteB1"
	case NoteC2:
		return "NoteC2"
	case NoteCSharp2:
		return "NoteCSharp2"
	case NoteDFlat2:
		return "NoteDFlat2"
	case NoteD2:
		return "NoteD2"
	case NoteDSharp2:
		return "NoteDSharp2"
	case NoteEFlat2:
		return "NoteEFlat2"
	case NoteE2:
		return "NoteE2"
	case NoteF2:
		return "NoteF2"
	case NoteFSharp2:
		return "NoteFSharp2"
	case NoteGFlat2:
		return "NoteGFlat2"
	case NoteG2:
		return "NoteG2"
	case NoteGSharp2:
		return "NoteGSharp2"
	case NoteAFlat2:
		return "NoteAFlat2"
	case NoteA2:
		return "NoteA2"
	case NoteASharp2:
		return "NoteASharp2"
	case NoteBFlat2:
		return "NoteBFlat2"
	case NoteB2:
		return "NoteB2"
	case NoteC3:
		return "NoteC3"
	case NoteCSharp3:
		return "NoteCSharp3"
	case NoteDFlat3:
		return "NoteDFlat3"
	case NoteD3:
		return "NoteD3"
	case NoteDSharp3:
		return "NoteDSharp3"
	case NoteEFlat3:
		return "NoteEFlat3"
	case NoteE3:
		return "NoteE3"
	case NoteF3:
		return "NoteF3"
	case NoteFSharp3:
		return "NoteFSharp3"
	case NoteGFlat3:
		return "NoteGFlat3"
	case NoteG3:
		return "NoteG3"
	case NoteGSharp3:
		return "NoteGSharp3"
	case NoteAFlat3:
		return "NoteAFlat3"
	case NoteA3:
		return "NoteA3"
	case NoteASharp3:
		return "NoteASharp3"
	case NoteBFlat3:
		return "NoteBFlat3"
	case NoteB3:
		return "NoteB3"
	case NoteC4:
		return "NoteC4"
	case NoteCSharp4:
		return "NoteCSharp4"
	case NoteDFlat4:
		return "NoteDFlat4"
	case NoteD4:
		return "NoteD4"
	case NoteDSharp4:
		return "NoteDSharp4"
	case NoteEFlat4:
		return "NoteEFlat4"
	case NoteE4:
		return "NoteE4"
	case NoteF4:
		return "NoteF4"
	case NoteFSharp4:
		return "NoteFSharp4"
	case NoteGFlat4:
		return "NoteGFlat4"
	case NoteG4:
		return "NoteG4"
	case NoteGSharp4:
		return "NoteGSharp4"
	case NoteAFlat4:
		return "NoteAFlat4"
	case NoteA4:
		return "NoteA4"
	case NoteASharp4:
		return "NoteASharp4"
	case NoteBFlat4:
		return "NoteBFlat4"
	case NoteB4:
		return "NoteB4"
	case NoteC5:
		return "NoteC5"
	case NoteCSharp5:
		return "NoteCSharp5"
	case NoteDFlat5:
		return "NoteDFlat5"
	case NoteD5:
		return "NoteD5"
	case NoteDSharp5:
		return "NoteDSharp5"
	case NoteEFlat5:
		return "NoteEFlat5"
	case NoteE5:
		return "NoteE5"
	case NoteF5:
		return "NoteF5"
	case NoteFSharp5:
		return "NoteFSharp5"
	case NoteGFlat5:
		return "NoteGFlat5"
	case NoteG5:
		return "NoteG5"
	case NoteGSharp5:
		return "NoteGSharp5"
	case NoteAFlat5:
		return "NoteAFlat5"
	case NoteA5:
		return "NoteA5"
	case NoteASharp5:
		return "NoteASharp5"
	case NoteBFlat5:
		return "NoteBFlat5"
	case NoteB5:
		return "NoteB5"
	case NoteC6:
		return "NoteC6"
	case NoteCSharp6:
		return "NoteCSharp6"
	case NoteDFlat6:
		return "NoteDFlat6"
	case NoteD6:
		return "NoteD6"
	case NoteDSharp6:
		return "NoteDSharp6"
	case NoteEFlat6:
		return "NoteEFlat6"
	case NoteE6:
		return "NoteE6"
	case NoteF6:
		return "NoteF6"
	case NoteFSharp6:
		return "NoteFSharp6"
	case NoteGFlat6:
		return "NoteGFlat6"
	case NoteG6:
		return "NoteG6"
	case NoteGSharp6:
		return "NoteGSharp6"
	case NoteAFlat6:
		return "NoteAFlat6"
	case NoteA6:
		return "NoteA6"
	case NoteASharp6:
		return "NoteASharp6"
	case NoteBFlat6:
		return "NoteBFlat6"
	case NoteB6:
		return "NoteB6"
	case NoteC7:
		return "NoteC7"
	case NoteCSharp7:
		return "NoteCSharp7"
	case NoteDFlat7:
		return "NoteDFlat7"
	case NoteD7:
		return "NoteD7"
	case NoteDSharp7:
		return "NoteDSharp7"
	case NoteEFlat7:
		return "NoteEFlat7"
	case NoteE7:
		return "NoteE7"
	case NoteF7:
		return "NoteF7"
	case NoteFSharp7:
		return "NoteFSharp7"
	case NoteGFlat7:
		return "NoteGFlat7"
	case NoteG7:
		return "NoteG7"
	case NoteGSharp7:
		return "NoteGSharp7"
	case NoteAFlat7:
		return "NoteAFlat7"
	case NoteA7:
		return "NoteA7"
	case NoteASharp7:
		return "NoteASharp7"
	case NoteBFlat7:
		return "NoteBFlat7"
	case NoteB7:
		return "NoteB7"
	case NoteC8:
		return "NoteC8"
	case NoteCSharp8:
		return "NoteCSharp8"
	case NoteDFlat8:
		return "NoteDFlat8"
	case NoteD8:
		return "NoteD8"
	case NoteDSharp8:
		return "NoteDSharp8"
	case NoteEFlat8:
		return "NoteEFlat8"
	case NoteE8:
		return "NoteE8"
	case NoteF8:
		return "NoteF8"
	case NoteFSharp8:
		return "NoteFSharp8"
	case NoteGFlat8:
		return "NoteGFlat8"
	case NoteG8:
		return "NoteG8"
	case NoteGSharp8:
		return "NoteGSharp8"
	case NoteAFlat8:
		return "NoteAFlat8"
	case NoteA8:
		return "NoteA8"
	case NoteASharp8:
		return "NoteASharp8"
	case NoteBFlat8:
		return "NoteBFlat8"
	case NoteB8:
		return "NoteB8"
	case NoteC9:
		return "NoteC9"
	case NoteCSharp9:
		return "NoteCSharp9"
	case NoteDFlat9:
		return "NoteDFlat9"
	case NoteD9:
		return "NoteD9"
	case NoteDSharp9:
		return "NoteDSharp9"
	case NoteEFlat9:
		return "NoteEFlat9"
	case NoteE9:
		return "NoteE9"
	case NoteF9:
		return "NoteF9"
	case NoteFSharp9:
		return "NoteFSharp9"
	case NoteGFlat9:
		return "NoteGFlat9"
	case NoteG9:
		return "NoteG9"
	}
	panic(fmt.Sprintf("ui: invalid midi key: %d", k))
}
