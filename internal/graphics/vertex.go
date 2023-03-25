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
	ShaderImageCount = 4

	// PreservedUniformVariablesCount represents the number of preserved uniform variables.
	// Any shaders in Ebitengine must have these uniform variables.
	PreservedUniformVariablesCount = 1 + // the destination texture size
		1 + // the texture sizes array
		1 + // the texture destination region's origin
		1 + // the texture destination region's size
		1 + // the offsets array of the second and the following images
		1 + // the texture source region's origin
		1 + // the texture source region's size
		1 // the projection matrix

	TextureDestinationSizeUniformVariableIndex         = 0
	TextureSourceSizesUniformVariableIndex             = 1
	TextureDestinationRegionOriginUniformVariableIndex = 2
	TextureDestinationRegionSizeUniformVariableIndex   = 3
	TextureSourceOffsetsUniformVariableIndex           = 4
	TextureSourceRegionOriginUniformVariableIndex      = 5
	TextureSourceRegionSizeUniformVariableIndex        = 6
	ProjectionMatrixUniformVariableIndex               = 7

	PreservedUniformUint32Count = 2 + // the destination texture size
		2*ShaderImageCount + // the texture sizes array
		2 + // the texture destination region's origin
		2 + // the texture destination region's size
		2*(ShaderImageCount-1) + // the offsets array of the second and the following images
		2 + // the texture source region's origin
		2 + // the texture source region's size
		16 // the projection matrix
)

const (
	IndicesCount     = (1 << 16) / 3 * 3 // Adjust num for triangles.
	VertexFloatCount = 8
)

var (
	quadIndices = []uint16{0, 1, 2, 1, 2, 3}
)

func QuadIndices() []uint16 {
	return quadIndices
}

// QuadVertices sets a float32 slice for a quadrangle.
// QuadVertices sets a slice that never overlaps with other slices returned this function,
// and users can do optimization based on this fact.
func QuadVertices(dst []float32, sx0, sy0, sx1, sy1 float32, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) {
	x := sx1 - sx0
	y := sy1 - sy0
	ax, by, cx, dy := a*x, b*y, c*x, d*y
	u0, v0, u1, v1 := sx0, sy0, sx1, sy1

	// This function is very performance-sensitive and implement in a very dumb way.
	_ = dst[:4*VertexFloatCount]

	dst[0] = adjustDestinationPixel(tx)
	dst[1] = adjustDestinationPixel(ty)
	dst[2] = u0
	dst[3] = v0
	dst[4] = cr
	dst[5] = cg
	dst[6] = cb
	dst[7] = ca

	dst[8] = adjustDestinationPixel(ax + tx)
	dst[9] = adjustDestinationPixel(cx + ty)
	dst[10] = u1
	dst[11] = v0
	dst[12] = cr
	dst[13] = cg
	dst[14] = cb
	dst[15] = ca

	dst[16] = adjustDestinationPixel(by + tx)
	dst[17] = adjustDestinationPixel(dy + ty)
	dst[18] = u0
	dst[19] = v1
	dst[20] = cr
	dst[21] = cg
	dst[22] = cb
	dst[23] = ca

	dst[24] = adjustDestinationPixel(ax + by + tx)
	dst[25] = adjustDestinationPixel(cx + dy + ty)
	dst[26] = u1
	dst[27] = v1
	dst[28] = cr
	dst[29] = cg
	dst[30] = cb
	dst[31] = ca
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
