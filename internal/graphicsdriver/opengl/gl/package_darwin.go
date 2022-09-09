// SPDX-License-Identifier: MIT

package gl

import (
	"errors"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	gpActiveTexture               uintptr
	gpAttachShader                uintptr
	gpBindAttribLocation          uintptr
	gpBindBuffer                  uintptr
	gpBindFramebufferEXT          uintptr
	gpBindRenderbufferEXT         uintptr
	gpBindTexture                 uintptr
	gpBlendFunc                   uintptr
	gpBufferData                  uintptr
	gpBufferSubData               uintptr
	gpCheckFramebufferStatusEXT   uintptr
	gpClear                       uintptr
	gpColorMask                   uintptr
	gpCompileShader               uintptr
	gpCreateProgram               uintptr
	gpCreateShader                uintptr
	gpDeleteBuffers               uintptr
	gpDeleteFramebuffersEXT       uintptr
	gpDeleteProgram               uintptr
	gpDeleteRenderbuffersEXT      uintptr
	gpDeleteShader                uintptr
	gpDeleteTextures              uintptr
	gpDisable                     uintptr
	gpDisableVertexAttribArray    uintptr
	gpDrawElements                uintptr
	gpEnable                      uintptr
	gpEnableVertexAttribArray     uintptr
	gpFlush                       uintptr
	gpFramebufferRenderbufferEXT  uintptr
	gpFramebufferTexture2DEXT     uintptr
	gpGenBuffers                  uintptr
	gpGenFramebuffersEXT          uintptr
	gpGenRenderbuffersEXT         uintptr
	gpGenTextures                 uintptr
	gpGetDoublei_v                uintptr
	gpGetDoublei_vEXT             uintptr
	gpGetError                    uintptr
	gpGetFloati_v                 uintptr
	gpGetFloati_vEXT              uintptr
	gpGetIntegeri_v               uintptr
	gpGetIntegerui64i_vNV         uintptr
	gpGetIntegerv                 uintptr
	gpGetPointeri_vEXT            uintptr
	gpGetProgramInfoLog           uintptr
	gpGetProgramiv                uintptr
	gpGetShaderInfoLog            uintptr
	gpGetShaderiv                 uintptr
	gpGetTransformFeedbacki64_v   uintptr
	gpGetTransformFeedbacki_v     uintptr
	gpGetUniformLocation          uintptr
	gpGetUnsignedBytei_vEXT       uintptr
	gpGetVertexArrayIntegeri_vEXT uintptr
	gpGetVertexArrayPointeri_vEXT uintptr
	gpIsFramebufferEXT            uintptr
	gpIsProgram                   uintptr
	gpIsRenderbufferEXT           uintptr
	gpIsTexture                   uintptr
	gpLinkProgram                 uintptr
	gpPixelStorei                 uintptr
	gpReadPixels                  uintptr
	gpRenderbufferStorageEXT      uintptr
	gpScissor                     uintptr
	gpShaderSource                uintptr
	gpStencilFunc                 uintptr
	gpStencilOp                   uintptr
	gpTexImage2D                  uintptr
	gpTexParameteri               uintptr
	gpTexSubImage2D               uintptr
	gpUniform1f                   uintptr
	gpUniform1i                   uintptr
	gpUniform1fv                  uintptr
	gpUniform2fv                  uintptr
	gpUniform3fv                  uintptr
	gpUniform4fv                  uintptr
	gpUniformMatrix2fv            uintptr
	gpUniformMatrix3fv            uintptr
	gpUniformMatrix4fv            uintptr
	gpUseProgram                  uintptr
	gpVertexAttribPointer         uintptr
	gpViewport                    uintptr
)

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func ActiveTexture(texture uint32) {
	purego.SyscallN(gpActiveTexture, uintptr(texture))
}

func AttachShader(program uint32, shader uint32) {
	purego.SyscallN(gpAttachShader, uintptr(program), uintptr(shader))
}

func BindAttribLocation(program uint32, index uint32, name *uint8) {
	purego.SyscallN(gpBindAttribLocation, uintptr(program), uintptr(index), uintptr(unsafe.Pointer(name)))
}

func BindBuffer(target uint32, buffer uint32) {
	purego.SyscallN(gpBindBuffer, uintptr(target), uintptr(buffer))
}

func BindFramebufferEXT(target uint32, framebuffer uint32) {
	purego.SyscallN(gpBindFramebufferEXT, uintptr(target), uintptr(framebuffer))
}

func BindRenderbufferEXT(target uint32, renderbuffer uint32) {
	purego.SyscallN(gpBindRenderbufferEXT, uintptr(target), uintptr(renderbuffer))
}

func BindTexture(target uint32, texture uint32) {
	purego.SyscallN(gpBindTexture, uintptr(target), uintptr(texture))
}

func BlendFunc(sfactor uint32, dfactor uint32) {
	purego.SyscallN(gpBlendFunc, uintptr(sfactor), uintptr(dfactor))
}

func BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	purego.SyscallN(gpBufferData, uintptr(target), uintptr(size), uintptr(data), uintptr(usage))
}

func BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	purego.SyscallN(gpBufferSubData, uintptr(target), uintptr(offset), uintptr(size), uintptr(data))
}

func CheckFramebufferStatusEXT(target uint32) uint32 {
	ret, _, _ := purego.SyscallN(gpCheckFramebufferStatusEXT, uintptr(target))
	return uint32(ret)
}

func Clear(mask uint32) {
	purego.SyscallN(gpClear, uintptr(mask))
}

func ColorMask(red bool, green bool, blue bool, alpha bool) {
	purego.SyscallN(gpColorMask, uintptr(boolToInt(red)), uintptr(boolToInt(green)), uintptr(boolToInt(blue)), uintptr(boolToInt(alpha)))
}

func CompileShader(shader uint32) {
	purego.SyscallN(gpCompileShader, uintptr(shader))
}

func CreateProgram() uint32 {
	ret, _, _ := purego.SyscallN(gpCreateProgram)
	return uint32(ret)
}

func CreateShader(xtype uint32) uint32 {
	ret, _, _ := purego.SyscallN(gpCreateShader, uintptr(xtype))
	return uint32(ret)
}

func DeleteBuffers(n int32, buffers *uint32) {
	purego.SyscallN(gpDeleteBuffers, uintptr(n), uintptr(unsafe.Pointer(buffers)))
}

func DeleteFramebuffersEXT(n int32, framebuffers *uint32) {
	purego.SyscallN(gpDeleteFramebuffersEXT, uintptr(n), uintptr(unsafe.Pointer(framebuffers)))
}

func DeleteProgram(program uint32) {
	purego.SyscallN(gpDeleteProgram, uintptr(program))
}

func DeleteRenderbuffersEXT(n int32, renderbuffers *uint32) {
	purego.SyscallN(gpDeleteRenderbuffersEXT, uintptr(n), uintptr(unsafe.Pointer(renderbuffers)))
}

func DeleteShader(shader uint32) {
	purego.SyscallN(gpDeleteShader, uintptr(shader))
}

func DeleteTextures(n int32, textures *uint32) {
	purego.SyscallN(gpDeleteTextures, uintptr(n), uintptr(unsafe.Pointer(textures)))
}

func Disable(cap uint32) {
	purego.SyscallN(gpDisable, uintptr(cap))
}

func DisableVertexAttribArray(index uint32) {
	purego.SyscallN(gpDisableVertexAttribArray, uintptr(index))
}

func DrawElements(mode uint32, count int32, xtype uint32, indices uintptr) {
	purego.SyscallN(gpDrawElements, uintptr(mode), uintptr(count), uintptr(xtype), uintptr(indices))
}

func Enable(cap uint32) {
	purego.SyscallN(gpEnable, uintptr(cap))
}

func EnableVertexAttribArray(index uint32) {
	purego.SyscallN(gpEnableVertexAttribArray, uintptr(index))
}

func Flush() {
	purego.SyscallN(gpFlush)
}

func FramebufferRenderbufferEXT(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	purego.SyscallN(gpFramebufferRenderbufferEXT, uintptr(target), uintptr(attachment), uintptr(renderbuffertarget), uintptr(renderbuffer))
}

func FramebufferTexture2DEXT(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	purego.SyscallN(gpFramebufferTexture2DEXT, uintptr(target), uintptr(attachment), uintptr(textarget), uintptr(texture), uintptr(level))
}

func GenBuffers(n int32, buffers *uint32) {
	purego.SyscallN(gpGenBuffers, uintptr(n), uintptr(unsafe.Pointer(buffers)))
}

func GenFramebuffersEXT(n int32, framebuffers *uint32) {
	purego.SyscallN(gpGenFramebuffersEXT, uintptr(n), uintptr(unsafe.Pointer(framebuffers)))
}

func GenRenderbuffersEXT(n int32, renderbuffers *uint32) {
	purego.SyscallN(gpGenRenderbuffersEXT, uintptr(n), uintptr(unsafe.Pointer(renderbuffers)))
}

func GenTextures(n int32, textures *uint32) {
	purego.SyscallN(gpGenTextures, uintptr(n), uintptr(unsafe.Pointer(textures)))
}

func GetDoublei_v(target uint32, index uint32, data *float64) {
	purego.SyscallN(gpGetDoublei_v, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetDoublei_vEXT(pname uint32, index uint32, params *float64) {
	purego.SyscallN(gpGetDoublei_vEXT, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetError() uint32 {
	ret, _, _ := purego.SyscallN(gpGetError)
	return uint32(ret)
}
func GetFloati_v(target uint32, index uint32, data *float32) {
	purego.SyscallN(gpGetFloati_v, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetFloati_vEXT(pname uint32, index uint32, params *float32) {
	purego.SyscallN(gpGetFloati_vEXT, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetIntegeri_v(target uint32, index uint32, data *int32) {
	purego.SyscallN(gpGetIntegeri_v, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetIntegerui64i_vNV(value uint32, index uint32, result *uint64) {
	purego.SyscallN(gpGetIntegerui64i_vNV, uintptr(value), uintptr(index), uintptr(unsafe.Pointer(result)))
}
func GetIntegerv(pname uint32, data *int32) {
	purego.SyscallN(gpGetIntegerv, uintptr(pname), uintptr(unsafe.Pointer(data)))
}

func GetPointeri_vEXT(pname uint32, index uint32, params *unsafe.Pointer) {
	purego.SyscallN(gpGetPointeri_vEXT, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetProgramInfoLog(program uint32, bufSize int32, length *int32, infoLog *uint8) {
	purego.SyscallN(gpGetProgramInfoLog, uintptr(program), uintptr(bufSize), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(infoLog)))
}

func GetProgramiv(program uint32, pname uint32, params *int32) {
	purego.SyscallN(gpGetProgramiv, uintptr(program), uintptr(pname), uintptr(unsafe.Pointer(params)))
}

func GetShaderInfoLog(shader uint32, bufSize int32, length *int32, infoLog *uint8) {
	purego.SyscallN(gpGetShaderInfoLog, uintptr(shader), uintptr(bufSize), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(infoLog)))
}

func GetShaderiv(shader uint32, pname uint32, params *int32) {
	purego.SyscallN(gpGetShaderiv, uintptr(shader), uintptr(pname), uintptr(unsafe.Pointer(params)))
}

func GetTransformFeedbacki64_v(xfb uint32, pname uint32, index uint32, param *int64) {
	purego.SyscallN(gpGetTransformFeedbacki64_v, uintptr(xfb), uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(param)))
}
func GetTransformFeedbacki_v(xfb uint32, pname uint32, index uint32, param *int32) {
	purego.SyscallN(gpGetTransformFeedbacki_v, uintptr(xfb), uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(param)))
}

func GetUniformLocation(program uint32, name *uint8) int32 {
	ret, _, _ := purego.SyscallN(gpGetUniformLocation, uintptr(program), uintptr(unsafe.Pointer(name)))
	return int32(ret)
}

func GetUnsignedBytei_vEXT(target uint32, index uint32, data *uint8) {
	purego.SyscallN(gpGetUnsignedBytei_vEXT, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetVertexArrayIntegeri_vEXT(vaobj uint32, index uint32, pname uint32, param *int32) {
	purego.SyscallN(gpGetVertexArrayIntegeri_vEXT, uintptr(vaobj), uintptr(index), uintptr(pname), uintptr(unsafe.Pointer(param)))
}
func GetVertexArrayPointeri_vEXT(vaobj uint32, index uint32, pname uint32, param *unsafe.Pointer) {
	purego.SyscallN(gpGetVertexArrayPointeri_vEXT, uintptr(vaobj), uintptr(index), uintptr(pname), uintptr(unsafe.Pointer(param)))
}

func IsFramebufferEXT(framebuffer uint32) bool {
	ret, _, _ := purego.SyscallN(gpIsFramebufferEXT, uintptr(framebuffer))
	return byte(ret) != 0
}

func IsProgram(program uint32) bool {
	ret, _, _ := purego.SyscallN(gpIsProgram, uintptr(program))
	return byte(ret) != 0
}

func IsRenderbufferEXT(renderbuffer uint32) bool {
	ret, _, _ := purego.SyscallN(gpIsRenderbufferEXT, uintptr(renderbuffer))
	return byte(ret) != 0
}

func IsTexture(texture uint32) bool {
	ret, _, _ := purego.SyscallN(gpIsTexture, uintptr(texture))
	return byte(ret) != 0
}

func LinkProgram(program uint32) {
	purego.SyscallN(gpLinkProgram, uintptr(program))
}

func PixelStorei(pname uint32, param int32) {
	purego.SyscallN(gpPixelStorei, uintptr(pname), uintptr(param))
}

func ReadPixels(x int32, y int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	purego.SyscallN(gpReadPixels, uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(pixels))
}

func RenderbufferStorageEXT(target uint32, internalformat uint32, width int32, height int32) {
	purego.SyscallN(gpRenderbufferStorageEXT, uintptr(target), uintptr(internalformat), uintptr(width), uintptr(height))
}

func Scissor(x int32, y int32, width int32, height int32) {
	purego.SyscallN(gpScissor, uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func ShaderSource(shader uint32, count int32, xstring **uint8, length *int32) {
	purego.SyscallN(gpShaderSource, uintptr(shader), uintptr(count), uintptr(unsafe.Pointer(xstring)), uintptr(unsafe.Pointer(length)))
}

func StencilFunc(xfunc uint32, ref int32, mask uint32) {
	purego.SyscallN(gpStencilFunc, uintptr(xfunc), uintptr(ref), uintptr(mask))
}

func StencilOp(fail uint32, zfail uint32, zpass uint32) {
	purego.SyscallN(gpStencilOp, uintptr(fail), uintptr(zfail), uintptr(zpass))
}

func TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	purego.SyscallN(gpTexImage2D, uintptr(target), uintptr(level), uintptr(internalformat), uintptr(width), uintptr(height), uintptr(border), uintptr(format), uintptr(xtype), uintptr(pixels))
}

func TexParameteri(target uint32, pname uint32, param int32) {
	purego.SyscallN(gpTexParameteri, uintptr(target), uintptr(pname), uintptr(param))
}

func TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	purego.SyscallN(gpTexSubImage2D, uintptr(target), uintptr(level), uintptr(xoffset), uintptr(yoffset), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(pixels))
}

func Uniform1f(location int32, v0 float32) {
	Uniform1fv(location, 1, (*float32)(Ptr([]float32{v0})))
}

func Uniform1i(location int32, v0 int32) {
	purego.SyscallN(gpUniform1i, uintptr(location), uintptr(v0))
}

func Uniform1fv(location int32, count int32, value *float32) {
	purego.SyscallN(gpUniform1fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform2fv(location int32, count int32, value *float32) {
	purego.SyscallN(gpUniform2fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform3fv(location int32, count int32, value *float32) {
	purego.SyscallN(gpUniform3fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform4fv(location int32, count int32, value *float32) {
	purego.SyscallN(gpUniform4fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func UniformMatrix2fv(location int32, count int32, transpose bool, value *float32) {
	purego.SyscallN(gpUniformMatrix2fv, uintptr(location), uintptr(count), uintptr(boolToInt(transpose)), uintptr(unsafe.Pointer(value)))
}

func UniformMatrix3fv(location int32, count int32, transpose bool, value *float32) {
	purego.SyscallN(gpUniformMatrix3fv, uintptr(location), uintptr(count), uintptr(boolToInt(transpose)), uintptr(unsafe.Pointer(value)))
}

func UniformMatrix4fv(location int32, count int32, transpose bool, value *float32) {
	purego.SyscallN(gpUniformMatrix4fv, uintptr(location), uintptr(count), uintptr(boolToInt(transpose)), uintptr(unsafe.Pointer(value)))
}

func UseProgram(program uint32) {
	purego.SyscallN(gpUseProgram, uintptr(program))
}

func VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, pointer uintptr) {
	purego.SyscallN(gpVertexAttribPointer, uintptr(index), uintptr(size), uintptr(xtype), uintptr(boolToInt(normalized)), uintptr(stride), uintptr(pointer))
}

func Viewport(x int32, y int32, width int32, height int32) {
	purego.SyscallN(gpViewport, uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

// InitWithProcAddrFunc intializes the package using the specified OpenGL
// function pointer loading function.
//
// For more cases Init should be used.
func InitWithProcAddrFunc(getProcAddr func(name string) uintptr) error {
	gpActiveTexture = getProcAddr("glActiveTexture")
	if gpActiveTexture == 0 {
		return errors.New("glActiveTexture")
	}
	gpAttachShader = getProcAddr("glAttachShader")
	if gpAttachShader == 0 {
		return errors.New("glAttachShader")
	}
	gpBindAttribLocation = getProcAddr("glBindAttribLocation")
	if gpBindAttribLocation == 0 {
		return errors.New("glBindAttribLocation")
	}
	gpBindBuffer = getProcAddr("glBindBuffer")
	if gpBindBuffer == 0 {
		return errors.New("glBindBuffer")
	}
	gpBindFramebufferEXT = getProcAddr("glBindFramebufferEXT")
	gpBindRenderbufferEXT = getProcAddr("glBindRenderbufferEXT")
	gpBindTexture = getProcAddr("glBindTexture")
	if gpBindTexture == 0 {
		return errors.New("glBindTexture")
	}
	gpBlendFunc = getProcAddr("glBlendFunc")
	if gpBlendFunc == 0 {
		return errors.New("glBlendFunc")
	}
	gpBufferData = getProcAddr("glBufferData")
	if gpBufferData == 0 {
		return errors.New("glBufferData")
	}
	gpBufferSubData = getProcAddr("glBufferSubData")
	if gpBufferSubData == 0 {
		return errors.New("glBufferSubData")
	}
	gpCheckFramebufferStatusEXT = getProcAddr("glCheckFramebufferStatusEXT")
	gpClear = getProcAddr("glClear")
	if gpClear == 0 {
		return errors.New("glClear")
	}
	gpColorMask = getProcAddr("glColorMask")
	if gpColorMask == 0 {
		return errors.New("glColorMask")
	}
	gpCompileShader = getProcAddr("glCompileShader")
	if gpCompileShader == 0 {
		return errors.New("glCompileShader")
	}
	gpCreateProgram = getProcAddr("glCreateProgram")
	if gpCreateProgram == 0 {
		return errors.New("glCreateProgram")
	}
	gpCreateShader = getProcAddr("glCreateShader")
	if gpCreateShader == 0 {
		return errors.New("glCreateShader")
	}
	gpDeleteBuffers = getProcAddr("glDeleteBuffers")
	if gpDeleteBuffers == 0 {
		return errors.New("glDeleteBuffers")
	}
	gpDeleteFramebuffersEXT = getProcAddr("glDeleteFramebuffersEXT")
	gpDeleteProgram = getProcAddr("glDeleteProgram")
	if gpDeleteProgram == 0 {
		return errors.New("glDeleteProgram")
	}
	gpDeleteRenderbuffersEXT = getProcAddr("glDeleteRenderbuffersEXT")
	gpDeleteShader = getProcAddr("glDeleteShader")
	if gpDeleteShader == 0 {
		return errors.New("glDeleteShader")
	}
	gpDeleteTextures = getProcAddr("glDeleteTextures")
	if gpDeleteTextures == 0 {
		return errors.New("glDeleteTextures")
	}
	gpDisable = getProcAddr("glDisable")
	if gpDisable == 0 {
		return errors.New("glDisable")
	}
	gpDisableVertexAttribArray = getProcAddr("glDisableVertexAttribArray")
	if gpDisableVertexAttribArray == 0 {
		return errors.New("glDisableVertexAttribArray")
	}
	gpDrawElements = getProcAddr("glDrawElements")
	if gpDrawElements == 0 {
		return errors.New("glDrawElements")
	}
	gpEnable = getProcAddr("glEnable")
	if gpEnable == 0 {
		return errors.New("glEnable")
	}
	gpEnableVertexAttribArray = getProcAddr("glEnableVertexAttribArray")
	if gpEnableVertexAttribArray == 0 {
		return errors.New("glEnableVertexAttribArray")
	}
	gpFlush = getProcAddr("glFlush")
	if gpFlush == 0 {
		return errors.New("glFlush")
	}
	gpFramebufferRenderbufferEXT = getProcAddr("glFramebufferRenderbufferEXT")
	gpFramebufferTexture2DEXT = getProcAddr("glFramebufferTexture2DEXT")
	gpGenBuffers = getProcAddr("glGenBuffers")
	if gpGenBuffers == 0 {
		return errors.New("glGenBuffers")
	}
	gpGenFramebuffersEXT = getProcAddr("glGenFramebuffersEXT")
	gpGenRenderbuffersEXT = getProcAddr("glGenRenderbuffersEXT")
	gpGenTextures = getProcAddr("glGenTextures")
	if gpGenTextures == 0 {
		return errors.New("glGenTextures")
	}
	gpGetDoublei_v = getProcAddr("glGetDoublei_v")
	gpGetDoublei_vEXT = getProcAddr("glGetDoublei_vEXT")
	gpGetError = getProcAddr("glGetError")
	if gpGetError == 0 {
		return errors.New("glGetError")
	}
	gpGetFloati_v = getProcAddr("glGetFloati_v")
	gpGetFloati_vEXT = getProcAddr("glGetFloati_vEXT")
	gpGetIntegeri_v = getProcAddr("glGetIntegeri_v")
	gpGetIntegerui64i_vNV = getProcAddr("glGetIntegerui64i_vNV")
	gpGetIntegerv = getProcAddr("glGetIntegerv")
	if gpGetIntegerv == 0 {
		return errors.New("glGetIntegerv")
	}
	gpGetPointeri_vEXT = getProcAddr("glGetPointeri_vEXT")
	gpGetProgramInfoLog = getProcAddr("glGetProgramInfoLog")
	if gpGetProgramInfoLog == 0 {
		return errors.New("glGetProgramInfoLog")
	}
	gpGetProgramiv = getProcAddr("glGetProgramiv")
	if gpGetProgramiv == 0 {
		return errors.New("glGetProgramiv")
	}
	gpGetShaderInfoLog = getProcAddr("glGetShaderInfoLog")
	if gpGetShaderInfoLog == 0 {
		return errors.New("glGetShaderInfoLog")
	}
	gpGetShaderiv = getProcAddr("glGetShaderiv")
	if gpGetShaderiv == 0 {
		return errors.New("glGetShaderiv")
	}
	gpGetTransformFeedbacki64_v = getProcAddr("glGetTransformFeedbacki64_v")
	gpGetTransformFeedbacki_v = getProcAddr("glGetTransformFeedbacki_v")
	gpGetUniformLocation = getProcAddr("glGetUniformLocation")
	if gpGetUniformLocation == 0 {
		return errors.New("glGetUniformLocation")
	}
	gpGetUnsignedBytei_vEXT = getProcAddr("glGetUnsignedBytei_vEXT")
	gpGetVertexArrayIntegeri_vEXT = getProcAddr("glGetVertexArrayIntegeri_vEXT")
	gpGetVertexArrayPointeri_vEXT = getProcAddr("glGetVertexArrayPointeri_vEXT")
	gpIsFramebufferEXT = getProcAddr("glIsFramebufferEXT")
	gpIsProgram = getProcAddr("glIsProgram")
	if gpIsProgram == 0 {
		return errors.New("glIsProgram")
	}
	gpIsRenderbufferEXT = getProcAddr("glIsRenderbufferEXT")
	gpIsTexture = getProcAddr("glIsTexture")
	if gpIsTexture == 0 {
		return errors.New("glIsTexture")
	}
	gpLinkProgram = getProcAddr("glLinkProgram")
	if gpLinkProgram == 0 {
		return errors.New("glLinkProgram")
	}
	gpPixelStorei = getProcAddr("glPixelStorei")
	if gpPixelStorei == 0 {
		return errors.New("glPixelStorei")
	}
	gpReadPixels = getProcAddr("glReadPixels")
	if gpReadPixels == 0 {
		return errors.New("glReadPixels")
	}
	gpRenderbufferStorageEXT = getProcAddr("glRenderbufferStorageEXT")
	gpScissor = getProcAddr("glScissor")
	if gpScissor == 0 {
		return errors.New("glScissor")
	}
	gpShaderSource = getProcAddr("glShaderSource")
	if gpShaderSource == 0 {
		return errors.New("glShaderSource")
	}
	gpStencilFunc = getProcAddr("glStencilFunc")
	if gpStencilFunc == 0 {
		return errors.New("glStencilFunc")
	}
	gpStencilOp = getProcAddr("glStencilOp")
	if gpStencilOp == 0 {
		return errors.New("glStencilOp")
	}
	gpTexImage2D = getProcAddr("glTexImage2D")
	if gpTexImage2D == 0 {
		return errors.New("glTexImage2D")
	}
	gpTexParameteri = getProcAddr("glTexParameteri")
	if gpTexParameteri == 0 {
		return errors.New("glTexParameteri")
	}
	gpTexSubImage2D = getProcAddr("glTexSubImage2D")
	if gpTexSubImage2D == 0 {
		return errors.New("glTexSubImage2D")
	}
	gpUniform1f = getProcAddr("glUniform1f")
	if gpUniform1f == 0 {
		return errors.New("glUniform1f")
	}
	gpUniform1i = getProcAddr("glUniform1i")
	if gpUniform1i == 0 {
		return errors.New("glUniform1i")
	}
	gpUniform1fv = getProcAddr("glUniform1fv")
	if gpUniform1fv == 0 {
		return errors.New("glUniform1fv")
	}
	gpUniform2fv = getProcAddr("glUniform2fv")
	if gpUniform2fv == 0 {
		return errors.New("glUniform2fv")
	}
	gpUniform3fv = getProcAddr("glUniform3fv")
	if gpUniform3fv == 0 {
		return errors.New("glUniform3fv")
	}
	gpUniform4fv = getProcAddr("glUniform4fv")
	if gpUniform4fv == 0 {
		return errors.New("glUniform4fv")
	}
	gpUniformMatrix2fv = getProcAddr("glUniformMatrix2fv")
	if gpUniformMatrix2fv == 0 {
		return errors.New("glUniformMatrix2fv")
	}
	gpUniformMatrix3fv = getProcAddr("glUniformMatrix3fv")
	if gpUniformMatrix3fv == 0 {
		return errors.New("glUniformMatrix3fv")
	}
	gpUniformMatrix4fv = getProcAddr("glUniformMatrix4fv")
	if gpUniformMatrix4fv == 0 {
		return errors.New("glUniformMatrix4fv")
	}
	gpUseProgram = getProcAddr("glUseProgram")
	if gpUseProgram == 0 {
		return errors.New("glUseProgram")
	}
	gpVertexAttribPointer = getProcAddr("glVertexAttribPointer")
	if gpVertexAttribPointer == 0 {
		return errors.New("glVertexAttribPointer")
	}
	gpViewport = getProcAddr("glViewport")
	if gpViewport == 0 {
		return errors.New("glViewport")
	}
	return nil
}
