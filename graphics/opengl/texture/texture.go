package texture

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/shader"
	"image"
	"unsafe"
)

type Texture struct {
	native C.GLuint
	width  int
	height int
}

func glMatrix(matrix [4][4]float64) [16]float32 {
	result := [16]float32{}
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			result[i+j*4] = float32(matrix[i][j])
		}
	}
	return result
}

func createNativeTexture(textureWidth, textureHeight int, pixels []uint8,
	filter graphics.Filter) C.GLuint {
	nativeTexture := C.GLuint(0)

	C.glGenTextures(1, &nativeTexture)
	if nativeTexture < 0 {
		panic("glGenTexture failed")
	}
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(nativeTexture))
	defer C.glBindTexture(C.GL_TEXTURE_2D, 0)

	glFilter := C.GLint(0)
	switch filter {
	case graphics.FilterLinear:
		glFilter = C.GL_LINEAR
	case graphics.FilterNearest:
		glFilter = C.GL_NEAREST
	default:
		panic("not reached")
	}
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, glFilter)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, glFilter)

	ptr := unsafe.Pointer(nil)
	if pixels != nil {
		ptr = unsafe.Pointer(&pixels[0])
	}
	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_RGBA,
		C.GLsizei(textureWidth), C.GLsizei(textureHeight),
		0, C.GL_RGBA, C.GL_UNSIGNED_BYTE, ptr)

	return nativeTexture
}

func Create(width, height int, filter graphics.Filter) (*Texture, error) {
	native := createNativeTexture(
		graphics.AdjustSizeForTexture(width),
		graphics.AdjustSizeForTexture(height), nil, filter)
	return &Texture{native, width, height}, nil
}

func CreateFromImage(img image.Image, filter graphics.Filter) (*Texture, error) {
	adjustedImage := graphics.AdjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return &Texture{native, size.X, size.Y}, nil
}

func (t *Texture) CreateRenderTarget() *rendertarget.RenderTarget {
	return rendertarget.CreateFromTexture(
		rendertarget.NativeTexture(t.native), t.width, t.height)
}

func (t *Texture) Draw(projectionMatrix [4][4]float64, geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	quad := graphics.TextureQuadForTexture(t.width, t.height)
	shader.DrawTexture(shader.NativeTexture(t.native),
		glMatrix(projectionMatrix), []graphics.TextureQuad{quad},
		geometryMatrix, colorMatrix)
}

func (t *Texture) DrawParts(parts []graphics.TexturePart, projectionMatrix [4][4]float64,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	quads := graphics.TextureQuadsForTextureParts(parts, t.width, t.height)
	shader.DrawTexture(shader.NativeTexture(t.native),
		glMatrix(projectionMatrix), quads,
		geometryMatrix, colorMatrix)
}

func (t *Texture) Dispose() {
	C.glDeleteTextures(1, &t.native)
}
