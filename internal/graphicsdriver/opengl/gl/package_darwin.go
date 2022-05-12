// SPDX-License-Identifier: MIT

package gl

import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2/internal/os/syscall"
	"unsafe"
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
	gpGetBufferSubData            uintptr
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
	syscall.SyscallX(gpActiveTexture, uintptr(texture), 0, 0)
}

func AttachShader(program uint32, shader uint32) {
	syscall.SyscallX(gpAttachShader, uintptr(program), uintptr(shader), 0)
}

func BindAttribLocation(program uint32, index uint32, name *uint8) {
	syscall.SyscallX(gpBindAttribLocation, uintptr(program), uintptr(index), uintptr(unsafe.Pointer(name)))
}

func BindBuffer(target uint32, buffer uint32) {
	syscall.SyscallX(gpBindBuffer, uintptr(target), uintptr(buffer), 0)
}

func BindFramebufferEXT(target uint32, framebuffer uint32) {
	syscall.SyscallX(gpBindFramebufferEXT, uintptr(target), uintptr(framebuffer), 0)
}

func BindRenderbufferEXT(target uint32, renderbuffer uint32) {
	syscall.SyscallX(gpBindRenderbufferEXT, uintptr(target), uintptr(renderbuffer), 0)
}

func BindTexture(target uint32, texture uint32) {
	syscall.SyscallX(gpBindTexture, uintptr(target), uintptr(texture), 0)
}

func BlendFunc(sfactor uint32, dfactor uint32) {
	syscall.SyscallX(gpBlendFunc, uintptr(sfactor), uintptr(dfactor), 0)
}

func BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	syscall.SyscallX6(gpBufferData, uintptr(target), uintptr(size), uintptr(data), uintptr(usage), 0, 0)
}

func BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	syscall.SyscallX6(gpBufferSubData, uintptr(target), uintptr(offset), uintptr(size), uintptr(data), 0, 0)
}

func CheckFramebufferStatusEXT(target uint32) uint32 {
	ret, _, _ := syscall.SyscallX(gpCheckFramebufferStatusEXT, uintptr(target), 0, 0)
	return (uint32)(ret)
}

func Clear(mask uint32) {
	syscall.SyscallX(gpClear, uintptr(mask), 0, 0)
}

func ColorMask(red bool, green bool, blue bool, alpha bool) {
	syscall.SyscallX6(gpColorMask, uintptr(boolToInt(red)), uintptr(boolToInt(green)), uintptr(boolToInt(blue)), uintptr(boolToInt(alpha)), 0, 0)
}

func CompileShader(shader uint32) {
	syscall.SyscallX(gpCompileShader, uintptr(shader), 0, 0)
}

func CreateProgram() uint32 {
	ret, _, _ := syscall.SyscallX(gpCreateProgram, 0, 0, 0)
	return (uint32)(ret)
}

func CreateShader(xtype uint32) uint32 {
	ret, _, _ := syscall.SyscallX(gpCreateShader, uintptr(xtype), 0, 0)
	return (uint32)(ret)
}

func DeleteBuffers(n int32, buffers *uint32) {
	syscall.SyscallX(gpDeleteBuffers, uintptr(n), uintptr(unsafe.Pointer(buffers)), 0)
}

func DeleteFramebuffersEXT(n int32, framebuffers *uint32) {
	syscall.SyscallX(gpDeleteFramebuffersEXT, uintptr(n), uintptr(unsafe.Pointer(framebuffers)), 0)
}

func DeleteProgram(program uint32) {
	syscall.SyscallX(gpDeleteProgram, uintptr(program), 0, 0)
}

func DeleteRenderbuffersEXT(n int32, renderbuffers *uint32) {
	syscall.SyscallX(gpDeleteRenderbuffersEXT, uintptr(n), uintptr(unsafe.Pointer(renderbuffers)), 0)
}

func DeleteShader(shader uint32) {
	syscall.SyscallX(gpDeleteShader, uintptr(shader), 0, 0)
}

func DeleteTextures(n int32, textures *uint32) {
	syscall.SyscallX(gpDeleteTextures, uintptr(n), uintptr(unsafe.Pointer(textures)), 0)
}

func Disable(cap uint32) {
	syscall.SyscallX(gpDisable, uintptr(cap), 0, 0)
}

func DisableVertexAttribArray(index uint32) {
	syscall.SyscallX(gpDisableVertexAttribArray, uintptr(index), 0, 0)
}

func DrawElements(mode uint32, count int32, xtype uint32, indices uintptr) {
	syscall.SyscallX6(gpDrawElements, uintptr(mode), uintptr(count), uintptr(xtype), uintptr(indices), 0, 0)
}

func Enable(cap uint32) {
	syscall.SyscallX(gpEnable, uintptr(cap), 0, 0)
}

func EnableVertexAttribArray(index uint32) {
	syscall.SyscallX(gpEnableVertexAttribArray, uintptr(index), 0, 0)
}

func Flush() {
	syscall.SyscallX(gpFlush, 0, 0, 0)
}

func FramebufferRenderbufferEXT(target uint32, attachment uint32, renderbuffertarget uint32, renderbuffer uint32) {
	syscall.SyscallX6(gpFramebufferRenderbufferEXT, uintptr(target), uintptr(attachment), uintptr(renderbuffertarget), uintptr(renderbuffer), 0, 0)
}

func FramebufferTexture2DEXT(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	syscall.SyscallX6(gpFramebufferTexture2DEXT, uintptr(target), uintptr(attachment), uintptr(textarget), uintptr(texture), uintptr(level), 0)
}

func GenBuffers(n int32, buffers *uint32) {
	syscall.SyscallX(gpGenBuffers, uintptr(n), uintptr(unsafe.Pointer(buffers)), 0)
}

func GenFramebuffersEXT(n int32, framebuffers *uint32) {
	syscall.SyscallX(gpGenFramebuffersEXT, uintptr(n), uintptr(unsafe.Pointer(framebuffers)), 0)
}

func GenRenderbuffersEXT(n int32, renderbuffers *uint32) {
	syscall.SyscallX(gpGenRenderbuffersEXT, uintptr(n), uintptr(unsafe.Pointer(renderbuffers)), 0)
}

func GenTextures(n int32, textures *uint32) {
	syscall.SyscallX(gpGenTextures, uintptr(n), uintptr(unsafe.Pointer(textures)), 0)
}

func GetBufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	syscall.SyscallX6(gpGetBufferSubData, uintptr(target), uintptr(offset), uintptr(size), uintptr(data), 0, 0)
}

func GetDoublei_v(target uint32, index uint32, data *float64) {
	syscall.SyscallX(gpGetDoublei_v, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetDoublei_vEXT(pname uint32, index uint32, params *float64) {
	syscall.SyscallX(gpGetDoublei_vEXT, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetError() uint32 {
	ret, _, _ := syscall.SyscallX(gpGetError, 0, 0, 0)
	return (uint32)(ret)
}
func GetFloati_v(target uint32, index uint32, data *float32) {
	syscall.SyscallX(gpGetFloati_v, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetFloati_vEXT(pname uint32, index uint32, params *float32) {
	syscall.SyscallX(gpGetFloati_vEXT, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetIntegeri_v(target uint32, index uint32, data *int32) {
	syscall.SyscallX(gpGetIntegeri_v, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetIntegerui64i_vNV(value uint32, index uint32, result *uint64) {
	syscall.SyscallX(gpGetIntegerui64i_vNV, uintptr(value), uintptr(index), uintptr(unsafe.Pointer(result)))
}
func GetIntegerv(pname uint32, data *int32) {
	syscall.SyscallX(gpGetIntegerv, uintptr(pname), uintptr(unsafe.Pointer(data)), 0)
}

func GetPointeri_vEXT(pname uint32, index uint32, params *unsafe.Pointer) {
	syscall.SyscallX(gpGetPointeri_vEXT, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetProgramInfoLog(program uint32, bufSize int32, length *int32, infoLog *uint8) {
	syscall.SyscallX6(gpGetProgramInfoLog, uintptr(program), uintptr(bufSize), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(infoLog)), 0, 0)
}

func GetProgramiv(program uint32, pname uint32, params *int32) {
	syscall.SyscallX(gpGetProgramiv, uintptr(program), uintptr(pname), uintptr(unsafe.Pointer(params)))
}

func GetShaderInfoLog(shader uint32, bufSize int32, length *int32, infoLog *uint8) {
	syscall.SyscallX6(gpGetShaderInfoLog, uintptr(shader), uintptr(bufSize), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(infoLog)), 0, 0)
}

func GetShaderiv(shader uint32, pname uint32, params *int32) {
	syscall.SyscallX(gpGetShaderiv, uintptr(shader), uintptr(pname), uintptr(unsafe.Pointer(params)))
}

func GetTransformFeedbacki64_v(xfb uint32, pname uint32, index uint32, param *int64) {
	syscall.SyscallX6(gpGetTransformFeedbacki64_v, uintptr(xfb), uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(param)), 0, 0)
}
func GetTransformFeedbacki_v(xfb uint32, pname uint32, index uint32, param *int32) {
	syscall.SyscallX6(gpGetTransformFeedbacki_v, uintptr(xfb), uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(param)), 0, 0)
}

func GetUniformLocation(program uint32, name *uint8) int32 {
	ret, _, _ := syscall.SyscallX(gpGetUniformLocation, uintptr(program), uintptr(unsafe.Pointer(name)), 0)
	return (int32)(ret)
}

func GetUnsignedBytei_vEXT(target uint32, index uint32, data *uint8) {
	syscall.SyscallX(gpGetUnsignedBytei_vEXT, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetVertexArrayIntegeri_vEXT(vaobj uint32, index uint32, pname uint32, param *int32) {
	syscall.SyscallX6(gpGetVertexArrayIntegeri_vEXT, uintptr(vaobj), uintptr(index), uintptr(pname), uintptr(unsafe.Pointer(param)), 0, 0)
}
func GetVertexArrayPointeri_vEXT(vaobj uint32, index uint32, pname uint32, param *unsafe.Pointer) {
	syscall.SyscallX6(gpGetVertexArrayPointeri_vEXT, uintptr(vaobj), uintptr(index), uintptr(pname), uintptr(unsafe.Pointer(param)), 0, 0)
}

func IsFramebufferEXT(framebuffer uint32) bool {
	ret, _, _ := syscall.SyscallX(gpIsFramebufferEXT, uintptr(framebuffer), 0, 0)
	return ret == TRUE
}

func IsProgram(program uint32) bool {
	ret, _, _ := syscall.SyscallX(gpIsProgram, uintptr(program), 0, 0)
	return ret == TRUE
}

func IsRenderbufferEXT(renderbuffer uint32) bool {
	ret, _, _ := syscall.SyscallX(gpIsRenderbufferEXT, uintptr(renderbuffer), 0, 0)
	return ret == TRUE
}

func IsTexture(texture uint32) bool {
	ret, _, _ := syscall.SyscallX(gpIsTexture, uintptr(texture), 0, 0)
	return ret == TRUE
}

func LinkProgram(program uint32) {
	syscall.SyscallX(gpLinkProgram, uintptr(program), 0, 0)
}

func PixelStorei(pname uint32, param int32) {
	syscall.SyscallX(gpPixelStorei, uintptr(pname), uintptr(param), 0)
}

func ReadPixels(x int32, y int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	syscall.SyscallX9(gpReadPixels, uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(pixels), 0, 0)
}

func RenderbufferStorageEXT(target uint32, internalformat uint32, width int32, height int32) {
	syscall.SyscallX6(gpRenderbufferStorageEXT, uintptr(target), uintptr(internalformat), uintptr(width), uintptr(height), 0, 0)
}

func Scissor(x int32, y int32, width int32, height int32) {
	syscall.SyscallX6(gpScissor, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0)
}

func ShaderSource(shader uint32, count int32, xstring **uint8, length *int32) {
	syscall.SyscallX6(gpShaderSource, uintptr(shader), uintptr(count), uintptr(unsafe.Pointer(xstring)), uintptr(unsafe.Pointer(length)), 0, 0)
}

func StencilFunc(xfunc uint32, ref int32, mask uint32) {
	syscall.SyscallX(gpStencilFunc, uintptr(xfunc), uintptr(ref), uintptr(mask))
}

func StencilOp(fail uint32, zfail uint32, zpass uint32) {
	syscall.SyscallX(gpStencilOp, uintptr(fail), uintptr(zfail), uintptr(zpass))
}

func TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	syscall.SyscallX9(gpTexImage2D, uintptr(target), uintptr(level), uintptr(internalformat), uintptr(width), uintptr(height), uintptr(border), uintptr(format), uintptr(xtype), uintptr(pixels))
}

func TexParameteri(target uint32, pname uint32, param int32) {
	syscall.SyscallX(gpTexParameteri, uintptr(target), uintptr(pname), uintptr(param))
}

func TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	syscall.SyscallX9(gpTexSubImage2D, uintptr(target), uintptr(level), uintptr(xoffset), uintptr(yoffset), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(pixels))
}

func Uniform1f(location int32, v0 float32) {
	syscall.SyscallXF(gpUniform1f, uintptr(location), 0, 0, float64(v0), 0, 0)
}

func Uniform1i(location int32, v0 int32) {
	syscall.SyscallX(gpUniform1i, uintptr(location), uintptr(v0), 0)
}

func Uniform1fv(location int32, count int32, value *float32) {
	syscall.SyscallX(gpUniform1fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform2fv(location int32, count int32, value *float32) {
	syscall.SyscallX(gpUniform2fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform3fv(location int32, count int32, value *float32) {
	syscall.SyscallX(gpUniform3fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform4fv(location int32, count int32, value *float32) {
	syscall.SyscallX(gpUniform4fv, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func UniformMatrix2fv(location int32, count int32, transpose bool, value *float32) {
	syscall.SyscallX6(gpUniformMatrix2fv, uintptr(location), uintptr(count), uintptr(boolToInt(transpose)), uintptr(unsafe.Pointer(value)), 0, 0)
}

func UniformMatrix3fv(location int32, count int32, transpose bool, value *float32) {
	syscall.SyscallX6(gpUniformMatrix3fv, uintptr(location), uintptr(count), uintptr(boolToInt(transpose)), uintptr(unsafe.Pointer(value)), 0, 0)
}

func UniformMatrix4fv(location int32, count int32, transpose bool, value *float32) {
	syscall.SyscallX6(gpUniformMatrix4fv, uintptr(location), uintptr(count), uintptr(boolToInt(transpose)), uintptr(unsafe.Pointer(value)), 0, 0)
}

func UseProgram(program uint32) {
	syscall.SyscallX(gpUseProgram, uintptr(program), 0, 0)
}

func VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, pointer uintptr) {
	syscall.SyscallX6(gpVertexAttribPointer, uintptr(index), uintptr(size), uintptr(xtype), uintptr(boolToInt(normalized)), uintptr(stride), uintptr(pointer))
}

func Viewport(x int32, y int32, width int32, height int32) {
	syscall.SyscallX6(gpViewport, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0)
}

// InitWithProcAddrFunc intializes the package using the specified OpenGL
// function pointer loading function.
//
// For more cases Init should be used.
func InitWithProcAddrFunc(getProcAddr func(name string) unsafe.Pointer) error {
	gpActiveTexture = uintptr(getProcAddr("glActiveTexture"))
	if gpActiveTexture == 0 {
		return errors.New("glActiveTexture")
	}
	gpAttachShader = uintptr(getProcAddr("glAttachShader"))
	if gpAttachShader == 0 {
		return errors.New("glAttachShader")
	}
	gpBindAttribLocation = uintptr(getProcAddr("glBindAttribLocation"))
	if gpBindAttribLocation == 0 {
		return errors.New("glBindAttribLocation")
	}
	gpBindBuffer = uintptr(getProcAddr("glBindBuffer"))
	if gpBindBuffer == 0 {
		return errors.New("glBindBuffer")
	}
	gpBindFramebufferEXT = uintptr(getProcAddr("glBindFramebufferEXT"))
	gpBindRenderbufferEXT = uintptr(getProcAddr("glBindRenderbufferEXT"))
	gpBindTexture = uintptr(getProcAddr("glBindTexture"))
	if gpBindTexture == 0 {
		return errors.New("glBindTexture")
	}
	gpBlendFunc = uintptr(getProcAddr("glBlendFunc"))
	if gpBlendFunc == 0 {
		return errors.New("glBlendFunc")
	}
	gpBufferData = uintptr(getProcAddr("glBufferData"))
	if gpBufferData == 0 {
		return errors.New("glBufferData")
	}
	gpBufferSubData = uintptr(getProcAddr("glBufferSubData"))
	if gpBufferSubData == 0 {
		return errors.New("glBufferSubData")
	}
	gpCheckFramebufferStatusEXT = uintptr(getProcAddr("glCheckFramebufferStatusEXT"))
	gpClear = uintptr(getProcAddr("glClear"))
	if gpClear == 0 {
		return errors.New("glClear")
	}
	gpColorMask = uintptr(getProcAddr("glColorMask"))
	if gpColorMask == 0 {
		return errors.New("glColorMask")
	}
	gpCompileShader = uintptr(getProcAddr("glCompileShader"))
	if gpCompileShader == 0 {
		return errors.New("glCompileShader")
	}
	gpCreateProgram = uintptr(getProcAddr("glCreateProgram"))
	if gpCreateProgram == 0 {
		return errors.New("glCreateProgram")
	}
	gpCreateShader = uintptr(getProcAddr("glCreateShader"))
	if gpCreateShader == 0 {
		return errors.New("glCreateShader")
	}
	gpDeleteBuffers = uintptr(getProcAddr("glDeleteBuffers"))
	if gpDeleteBuffers == 0 {
		return errors.New("glDeleteBuffers")
	}
	gpDeleteFramebuffersEXT = uintptr(getProcAddr("glDeleteFramebuffersEXT"))
	gpDeleteProgram = uintptr(getProcAddr("glDeleteProgram"))
	if gpDeleteProgram == 0 {
		return errors.New("glDeleteProgram")
	}
	gpDeleteRenderbuffersEXT = uintptr(getProcAddr("glDeleteRenderbuffersEXT"))
	gpDeleteShader = uintptr(getProcAddr("glDeleteShader"))
	if gpDeleteShader == 0 {
		return errors.New("glDeleteShader")
	}
	gpDeleteTextures = uintptr(getProcAddr("glDeleteTextures"))
	if gpDeleteTextures == 0 {
		return errors.New("glDeleteTextures")
	}
	gpDisable = uintptr(getProcAddr("glDisable"))
	if gpDisable == 0 {
		return errors.New("glDisable")
	}
	gpDisableVertexAttribArray = uintptr(getProcAddr("glDisableVertexAttribArray"))
	if gpDisableVertexAttribArray == 0 {
		return errors.New("glDisableVertexAttribArray")
	}
	gpDrawElements = uintptr(getProcAddr("glDrawElements"))
	if gpDrawElements == 0 {
		return errors.New("glDrawElements")
	}
	gpEnable = uintptr(getProcAddr("glEnable"))
	if gpEnable == 0 {
		return errors.New("glEnable")
	}
	gpEnableVertexAttribArray = uintptr(getProcAddr("glEnableVertexAttribArray"))
	if gpEnableVertexAttribArray == 0 {
		return errors.New("glEnableVertexAttribArray")
	}
	gpFlush = uintptr(getProcAddr("glFlush"))
	if gpFlush == 0 {
		return errors.New("glFlush")
	}
	gpFramebufferRenderbufferEXT = uintptr(getProcAddr("glFramebufferRenderbufferEXT"))
	gpFramebufferTexture2DEXT = uintptr(getProcAddr("glFramebufferTexture2DEXT"))
	gpGenBuffers = uintptr(getProcAddr("glGenBuffers"))
	if gpGenBuffers == 0 {
		return errors.New("glGenBuffers")
	}
	gpGenFramebuffersEXT = uintptr(getProcAddr("glGenFramebuffersEXT"))
	gpGenRenderbuffersEXT = uintptr(getProcAddr("glGenRenderbuffersEXT"))
	gpGenTextures = uintptr(getProcAddr("glGenTextures"))
	if gpGenTextures == 0 {
		return errors.New("glGenTextures")
	}
	gpGetBufferSubData = uintptr(getProcAddr("glGetBufferSubData"))
	if gpGetBufferSubData == 0 {
		return errors.New("glGetBufferSubData")
	}
	gpGetDoublei_v = uintptr(getProcAddr("glGetDoublei_v"))
	gpGetDoublei_vEXT = uintptr(getProcAddr("glGetDoublei_vEXT"))
	gpGetError = uintptr(getProcAddr("glGetError"))
	if gpGetError == 0 {
		return errors.New("glGetError")
	}
	gpGetFloati_v = uintptr(getProcAddr("glGetFloati_v"))
	gpGetFloati_vEXT = uintptr(getProcAddr("glGetFloati_vEXT"))
	gpGetIntegeri_v = uintptr(getProcAddr("glGetIntegeri_v"))
	gpGetIntegerui64i_vNV = uintptr(getProcAddr("glGetIntegerui64i_vNV"))
	gpGetIntegerv = uintptr(getProcAddr("glGetIntegerv"))
	if gpGetIntegerv == 0 {
		return errors.New("glGetIntegerv")
	}
	gpGetPointeri_vEXT = uintptr(getProcAddr("glGetPointeri_vEXT"))
	gpGetProgramInfoLog = uintptr(getProcAddr("glGetProgramInfoLog"))
	if gpGetProgramInfoLog == 0 {
		return errors.New("glGetProgramInfoLog")
	}
	gpGetProgramiv = uintptr(getProcAddr("glGetProgramiv"))
	if gpGetProgramiv == 0 {
		return errors.New("glGetProgramiv")
	}
	gpGetShaderInfoLog = uintptr(getProcAddr("glGetShaderInfoLog"))
	if gpGetShaderInfoLog == 0 {
		return errors.New("glGetShaderInfoLog")
	}
	gpGetShaderiv = uintptr(getProcAddr("glGetShaderiv"))
	if gpGetShaderiv == 0 {
		return errors.New("glGetShaderiv")
	}
	gpGetTransformFeedbacki64_v = uintptr(getProcAddr("glGetTransformFeedbacki64_v"))
	gpGetTransformFeedbacki_v = uintptr(getProcAddr("glGetTransformFeedbacki_v"))
	gpGetUniformLocation = uintptr(getProcAddr("glGetUniformLocation"))
	if gpGetUniformLocation == 0 {
		return errors.New("glGetUniformLocation")
	}
	gpGetUnsignedBytei_vEXT = uintptr(getProcAddr("glGetUnsignedBytei_vEXT"))
	gpGetVertexArrayIntegeri_vEXT = uintptr(getProcAddr("glGetVertexArrayIntegeri_vEXT"))
	gpGetVertexArrayPointeri_vEXT = uintptr(getProcAddr("glGetVertexArrayPointeri_vEXT"))
	gpIsFramebufferEXT = uintptr(getProcAddr("glIsFramebufferEXT"))
	gpIsProgram = uintptr(getProcAddr("glIsProgram"))
	if gpIsProgram == 0 {
		return errors.New("glIsProgram")
	}
	gpIsRenderbufferEXT = uintptr(getProcAddr("glIsRenderbufferEXT"))
	gpIsTexture = uintptr(getProcAddr("glIsTexture"))
	if gpIsTexture == 0 {
		return errors.New("glIsTexture")
	}
	gpLinkProgram = uintptr(getProcAddr("glLinkProgram"))
	if gpLinkProgram == 0 {
		return errors.New("glLinkProgram")
	}
	gpPixelStorei = uintptr(getProcAddr("glPixelStorei"))
	if gpPixelStorei == 0 {
		return errors.New("glPixelStorei")
	}
	gpReadPixels = uintptr(getProcAddr("glReadPixels"))
	if gpReadPixels == 0 {
		return errors.New("glReadPixels")
	}
	gpRenderbufferStorageEXT = uintptr(getProcAddr("glRenderbufferStorageEXT"))
	gpScissor = uintptr(getProcAddr("glScissor"))
	if gpScissor == 0 {
		return errors.New("glScissor")
	}
	gpShaderSource = uintptr(getProcAddr("glShaderSource"))
	if gpShaderSource == 0 {
		return errors.New("glShaderSource")
	}
	gpStencilFunc = uintptr(getProcAddr("glStencilFunc"))
	if gpStencilFunc == 0 {
		return errors.New("glStencilFunc")
	}
	gpStencilOp = uintptr(getProcAddr("glStencilOp"))
	if gpStencilOp == 0 {
		return errors.New("glStencilOp")
	}
	gpTexImage2D = uintptr(getProcAddr("glTexImage2D"))
	if gpTexImage2D == 0 {
		return errors.New("glTexImage2D")
	}
	gpTexParameteri = uintptr(getProcAddr("glTexParameteri"))
	if gpTexParameteri == 0 {
		return errors.New("glTexParameteri")
	}
	gpTexSubImage2D = uintptr(getProcAddr("glTexSubImage2D"))
	if gpTexSubImage2D == 0 {
		return errors.New("glTexSubImage2D")
	}
	gpUniform1f = uintptr(getProcAddr("glUniform1f"))
	if gpUniform1f == 0 {
		return errors.New("glUniform1f")
	}
	gpUniform1i = uintptr(getProcAddr("glUniform1i"))
	if gpUniform1i == 0 {
		return errors.New("glUniform1i")
	}
	gpUniform1fv = uintptr(getProcAddr("glUniform1fv"))
	if gpUniform1fv == 0 {
		return errors.New("glUniform1fv")
	}
	gpUniform2fv = uintptr(getProcAddr("glUniform2fv"))
	if gpUniform2fv == 0 {
		return errors.New("glUniform2fv")
	}
	gpUniform3fv = uintptr(getProcAddr("glUniform3fv"))
	if gpUniform3fv == 0 {
		return errors.New("glUniform3fv")
	}
	gpUniform4fv = uintptr(getProcAddr("glUniform4fv"))
	if gpUniform4fv == 0 {
		return errors.New("glUniform4fv")
	}
	gpUniformMatrix2fv = uintptr(getProcAddr("glUniformMatrix2fv"))
	if gpUniformMatrix2fv == 0 {
		return errors.New("glUniformMatrix2fv")
	}
	gpUniformMatrix3fv = uintptr(getProcAddr("glUniformMatrix3fv"))
	if gpUniformMatrix3fv == 0 {
		return errors.New("glUniformMatrix3fv")
	}
	gpUniformMatrix4fv = uintptr(getProcAddr("glUniformMatrix4fv"))
	if gpUniformMatrix4fv == 0 {
		return errors.New("glUniformMatrix4fv")
	}
	gpUseProgram = uintptr(getProcAddr("glUseProgram"))
	if gpUseProgram == 0 {
		return errors.New("glUseProgram")
	}
	gpVertexAttribPointer = uintptr(getProcAddr("glVertexAttribPointer"))
	if gpVertexAttribPointer == 0 {
		return errors.New("glVertexAttribPointer")
	}
	gpViewport = uintptr(getProcAddr("glViewport"))
	if gpViewport == 0 {
		return errors.New("glViewport")
	}
	return nil
}
