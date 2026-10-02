// Copyright 2024 The Ebitengine Authors
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
	"strings"
)

type Point struct {
	X, Y float32
}

func (p Point) String() string {
	return fmt.Sprintf("(%f, %f)", p.X, p.Y)
}

func IsPointCloseToSegment(p, p0, p1 Point, allow float32) bool {
	return isPointCloseToSegment(point{
		x: p.X,
		y: p.Y,
	}, point{
		x: p0.X,
		y: p0.Y,
	}, point{
		x: p1.X,
		y: p1.Y,
	}, allow)
}

func CurrentPosition(path *Path) (Point, bool) {
	p, ok := path.currentPosition()
	if !ok {
		return Point{}, false
	}
	return Point{X: p.x, Y: p.y}, true
}

func SubPathCount(path *Path) int {
	return len(path.subPaths)
}

// PathOperationsString returns a string representation of the operations in the path.
func PathOperationsString(path *Path) string {
	var sb strings.Builder
	for _, subPath := range path.subPaths {
		sb.WriteString(fmt.Sprintf("MoveTo(%v, %v)\n", subPath.start.x, subPath.start.y))
		for _, o := range subPath.ops {
			switch o.typ {
			case opTypeLineTo:
				sb.WriteString(fmt.Sprintf("LineTo(%v, %v)\n", o.p1.x, o.p1.y))
			case opTypeQuadTo:
				sb.WriteString(fmt.Sprintf("QuadTo(%v, %v, %v, %v)\n", o.p1.x, o.p1.y, o.p2.x, o.p2.y))
			}
		}
		if subPath.closed {
			sb.WriteString("Close()\n")
		}
	}
	return sb.String()
}

func FillPathsStateCount() int {
	theFillPathM.Lock()
	defer theFillPathM.Unlock()
	return len(theFillPathsStates)
}

func CallbackTokenCount() int {
	theFillPathM.Lock()
	defer theFillPathM.Unlock()
	return len(theCallbackTokens)
}

func CircleVertexCount(radius float32) int {
	return circleVertexCount(radius)
}
