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

package opentypesvg

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
)

// ErrCompression identifies invalid compressed SVG data.
var ErrCompression = errors.New("opentypesvg: invalid compressed document")

// Decode decodes a raw or gzip-compressed OpenType SVG record.
func Decode(source []byte) ([]byte, error) {
	if len(source) > maxBytes {
		return nil, ErrLimit
	}
	if len(source) < 2 || source[0] != 0x1f || source[1] != 0x8b {
		return source, nil
	}
	// The compressed budget is independent of the decoded XML budget.
	if len(source) > 8<<20 {
		return nil, ErrLimit
	}
	r, err := gzip.NewReader(bytes.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCompression, err)
	}
	defer r.Close()
	result, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if len(result) > maxBytes {
		return nil, ErrLimit
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCompression, err)
	}
	return result, nil
}
