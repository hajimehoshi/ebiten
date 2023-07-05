#include "_cgo_export.h"

void glfwSetKeyCallbackCB(GLFWwindow *window) {
  glfwSetKeyCallback(window, (GLFWkeyfun)goKeyCB);
}

void glfwSetCharCallbackCB(GLFWwindow *window) {
  glfwSetCharCallback(window, (GLFWcharfun)goCharCB);
}

void glfwSetCharModsCallbackCB(GLFWwindow *window) {
  glfwSetCharModsCallback(window, (GLFWcharmodsfun)goCharModsCB);
}

void glfwSetMouseButtonCallbackCB(GLFWwindow *window) {
  glfwSetMouseButtonCallback(window, (GLFWmousebuttonfun)goMouseButtonCB);
}

void glfwSetCursorPosCallbackCB(GLFWwindow *window) {
  glfwSetCursorPosCallback(window, (GLFWcursorposfun)goCursorPosCB);
}

void glfwSetCursorEnterCallbackCB(GLFWwindow *window) {
  glfwSetCursorEnterCallback(window, (GLFWcursorenterfun)goCursorEnterCB);
}

void glfwSetScrollCallbackCB(GLFWwindow *window) {
  glfwSetScrollCallback(window, (GLFWscrollfun)goScrollCB);
}

void glfwSetDropCallbackCB(GLFWwindow *window) {
  glfwSetDropCallback(window, (GLFWdropfun)goDropCB);
}
