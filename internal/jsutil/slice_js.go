// +build js,!wasm

package jsutil

import (
	"bytes"
	"encoding/binary"
)

func sliceToByteSlice(s interface{}) (bs []byte) {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, s)
	return b.Bytes()
}
