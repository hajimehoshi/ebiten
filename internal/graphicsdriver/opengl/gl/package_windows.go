// SPDX-License-Identifier: MIT

package gl

import (
	"errors"
	"math"
	"syscall"
	"unsafe"
)

var (
	gpActiveTexture               uintptr
	gpAttachShader                uintptr
	gpBindAttribLocation          uintptr
	gpBindBuffer                  uintptr
	gpBindFramebufferEXT          uintptr
	gpBindTexture                 uintptr
	gpBlendFunc                   uintptr
	gpBufferData                  uintptr
	gpBufferSubData               uintptr
	gpCheckFramebufferStatusEXT   uintptr
	gpCompileShader               uintptr
	gpCreateProgram               uintptr
	gpCreateShader                uintptr
	gpDeleteBuffers               uintptr
	gpDeleteFramebuffersEXT       uintptr
	gpDeleteProgram               uintptr
	gpDeleteShader                uintptr
	gpDeleteTextures              uintptr
	gpDisableVertexAttribArray    uintptr
	gpDrawElements                uintptr
	gpEnable                      uintptr
	gpEnableVertexAttribArray     uintptr
	gpFlush                       uintptr
	gpFramebufferTexture2DEXT     uintptr
	gpGenBuffers                  uintptr
	gpGenFramebuffersEXT          uintptr
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
	gpIsTexture                   uintptr
	gpLinkProgram                 uintptr
	gpPixelStorei                 uintptr
	gpReadPixels                  uintptr
	gpShaderSource                uintptr
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

func boolToUintptr(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

func ActiveTexture(texture uint32) {
	syscall.Syscall(gpActiveTexture, 1, uintptr(texture), 0, 0)
}

func AttachShader(program uint32, shader uint32) {
	syscall.Syscall(gpAttachShader, 2, uintptr(program), uintptr(shader), 0)
}

func BindAttribLocation(program uint32, index uint32, name *uint8) {
	syscall.Syscall(gpBindAttribLocation, 3, uintptr(program), uintptr(index), uintptr(unsafe.Pointer(name)))
}

func BindBuffer(target uint32, buffer uint32) {
	syscall.Syscall(gpBindBuffer, 2, uintptr(target), uintptr(buffer), 0)
}

func BindFramebufferEXT(target uint32, framebuffer uint32) {
	syscall.Syscall(gpBindFramebufferEXT, 2, uintptr(target), uintptr(framebuffer), 0)
}

func BindTexture(target uint32, texture uint32) {
	syscall.Syscall(gpBindTexture, 2, uintptr(target), uintptr(texture), 0)
}

func BlendFunc(sfactor uint32, dfactor uint32) {
	syscall.Syscall(gpBlendFunc, 2, uintptr(sfactor), uintptr(dfactor), 0)
}

func BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	syscall.Syscall6(gpBufferData, 4, uintptr(target), uintptr(size), uintptr(data), uintptr(usage), 0, 0)
}

func BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	syscall.Syscall6(gpBufferSubData, 4, uintptr(target), uintptr(offset), uintptr(size), uintptr(data), 0, 0)
}

func CheckFramebufferStatusEXT(target uint32) uint32 {
	ret, _, _ := syscall.Syscall(gpCheckFramebufferStatusEXT, 1, uintptr(target), 0, 0)
	return (uint32)(ret)
}

func CompileShader(shader uint32) {
	syscall.Syscall(gpCompileShader, 1, uintptr(shader), 0, 0)
}

func CreateProgram() uint32 {
	ret, _, _ := syscall.Syscall(gpCreateProgram, 0, 0, 0, 0)
	return (uint32)(ret)
}

func CreateShader(xtype uint32) uint32 {
	ret, _, _ := syscall.Syscall(gpCreateShader, 1, uintptr(xtype), 0, 0)
	return (uint32)(ret)
}

func DeleteBuffers(n int32, buffers *uint32) {
	syscall.Syscall(gpDeleteBuffers, 2, uintptr(n), uintptr(unsafe.Pointer(buffers)), 0)
}

func DeleteFramebuffersEXT(n int32, framebuffers *uint32) {
	syscall.Syscall(gpDeleteFramebuffersEXT, 2, uintptr(n), uintptr(unsafe.Pointer(framebuffers)), 0)
}

func DeleteProgram(program uint32) {
	syscall.Syscall(gpDeleteProgram, 1, uintptr(program), 0, 0)
}

func DeleteShader(shader uint32) {
	syscall.Syscall(gpDeleteShader, 1, uintptr(shader), 0, 0)
}

func DeleteTextures(n int32, textures *uint32) {
	syscall.Syscall(gpDeleteTextures, 2, uintptr(n), uintptr(unsafe.Pointer(textures)), 0)
}

func DisableVertexAttribArray(index uint32) {
	syscall.Syscall(gpDisableVertexAttribArray, 1, uintptr(index), 0, 0)
}

func DrawElements(mode uint32, count int32, xtype uint32, indices uintptr) {
	syscall.Syscall6(gpDrawElements, 4, uintptr(mode), uintptr(count), uintptr(xtype), uintptr(indices), 0, 0)
}

func Enable(cap uint32) {
	syscall.Syscall(gpEnable, 1, uintptr(cap), 0, 0)
}

func EnableVertexAttribArray(index uint32) {
	syscall.Syscall(gpEnableVertexAttribArray, 1, uintptr(index), 0, 0)
}

func Flush() {
	syscall.Syscall(gpFlush, 0, 0, 0, 0)
}

func FramebufferTexture2DEXT(target uint32, attachment uint32, textarget uint32, texture uint32, level int32) {
	syscall.Syscall6(gpFramebufferTexture2DEXT, 5, uintptr(target), uintptr(attachment), uintptr(textarget), uintptr(texture), uintptr(level), 0)
}

func GenBuffers(n int32, buffers *uint32) {
	syscall.Syscall(gpGenBuffers, 2, uintptr(n), uintptr(unsafe.Pointer(buffers)), 0)
}

func GenFramebuffersEXT(n int32, framebuffers *uint32) {
	syscall.Syscall(gpGenFramebuffersEXT, 2, uintptr(n), uintptr(unsafe.Pointer(framebuffers)), 0)
}

func GenTextures(n int32, textures *uint32) {
	syscall.Syscall(gpGenTextures, 2, uintptr(n), uintptr(unsafe.Pointer(textures)), 0)
}

func GetDoublei_v(target uint32, index uint32, data *float64) {
	syscall.Syscall(gpGetDoublei_v, 3, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetDoublei_vEXT(pname uint32, index uint32, params *float64) {
	syscall.Syscall(gpGetDoublei_vEXT, 3, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetError() uint32 {
	ret, _, _ := syscall.Syscall(gpGetError, 0, 0, 0, 0)
	return (uint32)(ret)
}
func GetFloati_v(target uint32, index uint32, data *float32) {
	syscall.Syscall(gpGetFloati_v, 3, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetFloati_vEXT(pname uint32, index uint32, params *float32) {
	syscall.Syscall(gpGetFloati_vEXT, 3, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetIntegeri_v(target uint32, index uint32, data *int32) {
	syscall.Syscall(gpGetIntegeri_v, 3, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetIntegerui64i_vNV(value uint32, index uint32, result *uint64) {
	syscall.Syscall(gpGetIntegerui64i_vNV, 3, uintptr(value), uintptr(index), uintptr(unsafe.Pointer(result)))
}
func GetIntegerv(pname uint32, data *int32) {
	syscall.Syscall(gpGetIntegerv, 2, uintptr(pname), uintptr(unsafe.Pointer(data)), 0)
}

func GetPointeri_vEXT(pname uint32, index uint32, params *unsafe.Pointer) {
	syscall.Syscall(gpGetPointeri_vEXT, 3, uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(params)))
}

func GetProgramInfoLog(program uint32, bufSize int32, length *int32, infoLog *uint8) {
	syscall.Syscall6(gpGetProgramInfoLog, 4, uintptr(program), uintptr(bufSize), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(infoLog)), 0, 0)
}

func GetProgramiv(program uint32, pname uint32, params *int32) {
	syscall.Syscall(gpGetProgramiv, 3, uintptr(program), uintptr(pname), uintptr(unsafe.Pointer(params)))
}

func GetShaderInfoLog(shader uint32, bufSize int32, length *int32, infoLog *uint8) {
	syscall.Syscall6(gpGetShaderInfoLog, 4, uintptr(shader), uintptr(bufSize), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(infoLog)), 0, 0)
}

func GetShaderiv(shader uint32, pname uint32, params *int32) {
	syscall.Syscall(gpGetShaderiv, 3, uintptr(shader), uintptr(pname), uintptr(unsafe.Pointer(params)))
}

func GetTransformFeedbacki64_v(xfb uint32, pname uint32, index uint32, param *int64) {
	syscall.Syscall6(gpGetTransformFeedbacki64_v, 4, uintptr(xfb), uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(param)), 0, 0)
}
func GetTransformFeedbacki_v(xfb uint32, pname uint32, index uint32, param *int32) {
	syscall.Syscall6(gpGetTransformFeedbacki_v, 4, uintptr(xfb), uintptr(pname), uintptr(index), uintptr(unsafe.Pointer(param)), 0, 0)
}

func GetUniformLocation(program uint32, name *uint8) int32 {
	ret, _, _ := syscall.Syscall(gpGetUniformLocation, 2, uintptr(program), uintptr(unsafe.Pointer(name)), 0)
	return (int32)(ret)
}

func GetUnsignedBytei_vEXT(target uint32, index uint32, data *uint8) {
	syscall.Syscall(gpGetUnsignedBytei_vEXT, 3, uintptr(target), uintptr(index), uintptr(unsafe.Pointer(data)))
}
func GetVertexArrayIntegeri_vEXT(vaobj uint32, index uint32, pname uint32, param *int32) {
	syscall.Syscall6(gpGetVertexArrayIntegeri_vEXT, 4, uintptr(vaobj), uintptr(index), uintptr(pname), uintptr(unsafe.Pointer(param)), 0, 0)
}
func GetVertexArrayPointeri_vEXT(vaobj uint32, index uint32, pname uint32, param *unsafe.Pointer) {
	syscall.Syscall6(gpGetVertexArrayPointeri_vEXT, 4, uintptr(vaobj), uintptr(index), uintptr(pname), uintptr(unsafe.Pointer(param)), 0, 0)
}

func IsFramebufferEXT(framebuffer uint32) bool {
	ret, _, _ := syscall.Syscall(gpIsFramebufferEXT, 1, uintptr(framebuffer), 0, 0)
	return ret != 0
}

func IsProgram(program uint32) bool {
	ret, _, _ := syscall.Syscall(gpIsProgram, 1, uintptr(program), 0, 0)
	return ret != 0
}

func IsTexture(texture uint32) bool {
	ret, _, _ := syscall.Syscall(gpIsTexture, 1, uintptr(texture), 0, 0)
	return ret != 0
}

func LinkProgram(program uint32) {
	syscall.Syscall(gpLinkProgram, 1, uintptr(program), 0, 0)
}

func PixelStorei(pname uint32, param int32) {
	syscall.Syscall(gpPixelStorei, 2, uintptr(pname), uintptr(param), 0)
}

func ReadPixels(x int32, y int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	syscall.Syscall9(gpReadPixels, 7, uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(pixels), 0, 0)
}

func ShaderSource(shader uint32, count int32, xstring **uint8, length *int32) {
	syscall.Syscall6(gpShaderSource, 4, uintptr(shader), uintptr(count), uintptr(unsafe.Pointer(xstring)), uintptr(unsafe.Pointer(length)), 0, 0)
}

func TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	syscall.Syscall9(gpTexImage2D, 9, uintptr(target), uintptr(level), uintptr(internalformat), uintptr(width), uintptr(height), uintptr(border), uintptr(format), uintptr(xtype), uintptr(pixels))
}

func TexParameteri(target uint32, pname uint32, param int32) {
	syscall.Syscall(gpTexParameteri, 3, uintptr(target), uintptr(pname), uintptr(param))
}

func TexSubImage2D(target uint32, level int32, xoffset int32, yoffset int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	syscall.Syscall9(gpTexSubImage2D, 9, uintptr(target), uintptr(level), uintptr(xoffset), uintptr(yoffset), uintptr(width), uintptr(height), uintptr(format), uintptr(xtype), uintptr(pixels))
}

func Uniform1f(location int32, v0 float32) {
	syscall.Syscall(gpUniform1f, 2, uintptr(location), uintptr(math.Float32bits(v0)), 0)
}

func Uniform1i(location int32, v0 int32) {
	syscall.Syscall(gpUniform1i, 2, uintptr(location), uintptr(v0), 0)
}

func Uniform1fv(location int32, count int32, value *float32) {
	syscall.Syscall(gpUniform1fv, 3, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform2fv(location int32, count int32, value *float32) {
	syscall.Syscall(gpUniform2fv, 3, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform3fv(location int32, count int32, value *float32) {
	syscall.Syscall(gpUniform3fv, 3, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func Uniform4fv(location int32, count int32, value *float32) {
	syscall.Syscall(gpUniform4fv, 3, uintptr(location), uintptr(count), uintptr(unsafe.Pointer(value)))
}

func UniformMatrix2fv(location int32, count int32, transpose bool, value *float32) {
	syscall.Syscall6(gpUniformMatrix2fv, 4, uintptr(location), uintptr(count), boolToUintptr(transpose), uintptr(unsafe.Pointer(value)), 0, 0)
}

func UniformMatrix3fv(location int32, count int32, transpose bool, value *float32) {
	syscall.Syscall6(gpUniformMatrix3fv, 4, uintptr(location), uintptr(count), boolToUintptr(transpose), uintptr(unsafe.Pointer(value)), 0, 0)
}

func UniformMatrix4fv(location int32, count int32, transpose bool, value *float32) {
	syscall.Syscall6(gpUniformMatrix4fv, 4, uintptr(location), uintptr(count), boolToUintptr(transpose), uintptr(unsafe.Pointer(value)), 0, 0)
}

func UseProgram(program uint32) {
	syscall.Syscall(gpUseProgram, 1, uintptr(program), 0, 0)
}

func VertexAttribPointer(index uint32, size int32, xtype uint32, normalized bool, stride int32, pointer uintptr) {
	syscall.Syscall6(gpVertexAttribPointer, 6, uintptr(index), uintptr(size), uintptr(xtype), boolToUintptr(normalized), uintptr(stride), uintptr(pointer))
}

func Viewport(x int32, y int32, width int32, height int32) {
	syscall.Syscall6(gpViewport, 4, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0)
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
	gpDeleteShader = getProcAddr("glDeleteShader")
	if gpDeleteShader == 0 {
		return errors.New("glDeleteShader")
	}
	gpDeleteTextures = getProcAddr("glDeleteTextures")
	if gpDeleteTextures == 0 {
		return errors.New("glDeleteTextures")
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
	gpFramebufferTexture2DEXT = getProcAddr("glFramebufferTexture2DEXT")
	gpGenBuffers = getProcAddr("glGenBuffers")
	if gpGenBuffers == 0 {
		return errors.New("glGenBuffers")
	}
	gpGenFramebuffersEXT = getProcAddr("glGenFramebuffersEXT")
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
	gpShaderSource = getProcAddr("glShaderSource")
	if gpShaderSource == 0 {
		return errors.New("glShaderSource")
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
