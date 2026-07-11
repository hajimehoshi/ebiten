// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2017 The oksvg Authors
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

package oksvg

import (
	"encoding/xml"
)

// definition is used to store XML-tags of SVG source definitions data.
type definition struct {
	ID, Tag string
	Attrs   []xml.Attr
}
