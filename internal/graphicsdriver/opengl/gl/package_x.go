// SPDX-License-Identifier: MIT

//go:build freebsd || linux
// +build freebsd linux

package gl

// #include <GL/glx.h>
//
// static const char* RendererDeviceString() {
// #ifdef GLX_MESA_query_renderer
//   static PFNGLXQUERYCURRENTRENDERERSTRINGMESAPROC queryString;
//   if (!queryString) {
//     queryString = (PFNGLXQUERYCURRENTRENDERERSTRINGMESAPROC)
//       glXGetProcAddressARB((const GLubyte *)"glXQueryCurrentRendererStringMESA");
//   }
//   if (!queryString) {
//     return "";
//   }
//
//   static const char* rendererDevice;
//   if (!rendererDevice) {
//     rendererDevice = queryString(GLX_RENDERER_DEVICE_ID_MESA);
//   }
//
//   return rendererDevice;
// #else
//   return "";
// #endif
// }
import "C"

func RendererDeviceString() string {
	return C.GoString(C.RendererDeviceString())
}
