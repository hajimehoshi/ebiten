// Copyright 2019 The Ebiten Authors
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

package graphics

const (
	ShaderSrcImageCount = 4

	// PreservedUniformVariablesCount represents the number of preserved uniform variables.
	// Any shaders in Ebitengine must have these uniform variables.
	PreservedUniformVariablesCount = 1 + // the destination texture size
		1 + // the source texture sizes array
		1 + // the destination image region origin
		1 + // the destination image region size
		1 + // the source image region origins
		1 + // the source image region sizes array
		1 // the projection matrix

	ProjectionMatrixUniformVariableIndex = 6

	PreservedUniformDwordCount = 2 + // the destination texture size
		2*ShaderSrcImageCount + // the source texture sizes array
		2 + // the destination image region origin
		2 + // the destination image region size
		2*ShaderSrcImageCount + // the source image region origins array
		2*ShaderSrcImageCount + // the source image region sizes array
		16 // the projection matrix

	ProjectionMatrixUniformDwordIndex = 2 +
		2*ShaderSrcImageCount +
		2 +
		2 +
		2*ShaderSrcImageCount +
		2*ShaderSrcImageCount
)

const (
	VertexFloatCount = 12
)

var (
	quadIndices = []uint32{0, 1, 2, 1, 2, 3}
)

func QuadIndices() []uint32 {
	return quadIndices
}

// QuadVerticesFromSrcAndMatrix sets a float32 slice for a quadrangle.
func QuadVerticesFromSrcAndMatrix(dst []float32, sx0, sy0, sx1, sy1 float32, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) {
	x := sx1 - sx0
	y := sy1 - sy0
	ax, by, cx, dy := a*x, b*y, c*x, d*y
	u0, v0, u1, v1 := sx0, sy0, sx1, sy1

	// This function is very performance-sensitive and implement in a very dumb way.

	// Remove the boundary check.
	dst = dst[:4*VertexFloatCount]

	dst[0] = adjustDestinationPixel(tx)
	dst[1] = adjustDestinationPixel(ty)
	dst[2] = u0
	dst[3] = v0
	dst[4] = cr
	dst[5] = cg
	dst[6] = cb
	dst[7] = ca

	dst[VertexFloatCount] = adjustDestinationPixel(ax + tx)
	dst[VertexFloatCount+1] = adjustDestinationPixel(cx + ty)
	dst[VertexFloatCount+2] = u1
	dst[VertexFloatCount+3] = v0
	dst[VertexFloatCount+4] = cr
	dst[VertexFloatCount+5] = cg
	dst[VertexFloatCount+6] = cb
	dst[VertexFloatCount+7] = ca

	dst[2*VertexFloatCount] = adjustDestinationPixel(by + tx)
	dst[2*VertexFloatCount+1] = adjustDestinationPixel(dy + ty)
	dst[2*VertexFloatCount+2] = u0
	dst[2*VertexFloatCount+3] = v1
	dst[2*VertexFloatCount+4] = cr
	dst[2*VertexFloatCount+5] = cg
	dst[2*VertexFloatCount+6] = cb
	dst[2*VertexFloatCount+7] = ca

	dst[3*VertexFloatCount] = adjustDestinationPixel(ax + by + tx)
	dst[3*VertexFloatCount+1] = adjustDestinationPixel(cx + dy + ty)
	dst[3*VertexFloatCount+2] = u1
	dst[3*VertexFloatCount+3] = v1
	dst[3*VertexFloatCount+4] = cr
	dst[3*VertexFloatCount+5] = cg
	dst[3*VertexFloatCount+6] = cb
	dst[3*VertexFloatCount+7] = ca
}

// QuadVerticesFromDstAndSrc sets a float32 slice for a quadrangle.
func QuadVerticesFromDstAndSrc(dst []float32, dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1, cr, cg, cb, ca float32) {
	dx0 = adjustDestinationPixel(dx0)
	dy0 = adjustDestinationPixel(dy0)
	dx1 = adjustDestinationPixel(dx1)
	dy1 = adjustDestinationPixel(dy1)

	// Remove the boundary check.
	dst = dst[:4*VertexFloatCount]

	dst[0] = dx0
	dst[1] = dy0
	dst[2] = sx0
	dst[3] = sy0
	dst[4] = cr
	dst[5] = cg
	dst[6] = cb
	dst[7] = ca

	dst[VertexFloatCount] = dx1
	dst[VertexFloatCount+1] = dy0
	dst[VertexFloatCount+2] = sx1
	dst[VertexFloatCount+3] = sy0
	dst[VertexFloatCount+4] = cr
	dst[VertexFloatCount+5] = cg
	dst[VertexFloatCount+6] = cb
	dst[VertexFloatCount+7] = ca

	dst[2*VertexFloatCount] = dx0
	dst[2*VertexFloatCount+1] = dy1
	dst[2*VertexFloatCount+2] = sx0
	dst[2*VertexFloatCount+3] = sy1
	dst[2*VertexFloatCount+4] = cr
	dst[2*VertexFloatCount+5] = cg
	dst[2*VertexFloatCount+6] = cb
	dst[2*VertexFloatCount+7] = ca

	dst[3*VertexFloatCount] = dx1
	dst[3*VertexFloatCount+1] = dy1
	dst[3*VertexFloatCount+2] = sx1
	dst[3*VertexFloatCount+3] = sy1
	dst[3*VertexFloatCount+4] = cr
	dst[3*VertexFloatCount+5] = cg
	dst[3*VertexFloatCount+6] = cb
	dst[3*VertexFloatCount+7] = ca
}

func adjustDestinationPixel(x float32) float32 {
	// Avoid the center of the pixel, which is problematic (#929, #1171).
	// Instead, align the vertices with about 1/3 pixels.
	//
	// The intention here is roughly this code:
	//
	//     float32(math.Floor((float64(x)+1.0/6.0)*3) / 3)
	//
	// The actual implementation is more optimized than the above implementation.
	ix := float32(int(x))
	if x < 0 && x != ix {
		ix -= 1
	}
	frac := x - ix
	switch {
	case frac < 3.0/16.0:
		return ix
	case frac < 8.0/16.0:
		return ix + 5.0/16.0
	case frac < 13.0/16.0:
		return ix + 11.0/16.0
	default:
		return ix + 16.0/16.0
	}
}
