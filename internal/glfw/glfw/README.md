These files are basically copy of github.com/v3.3/glfw/glfw.

There is one change from the original files.

`GLFWscrollfun` takes pointers instead of values since all arguments of C functions have to be 32bit on 32bit Windows machine.

```diff
diff --git a/tmp/glfw-3.3.3/include/GLFW/glfw3.h b/./internal/glfw/glfw/include/GLFW/glfw3.h
index 35bbf075..b41c0dca 100644
--- a/tmp/glfw-3.3.3/include/GLFW/glfw3.h
+++ b/./internal/glfw/glfw/include/GLFW/glfw3.h
@@ -1496,7 +1496,7 @@ typedef void (* GLFWcursorenterfun)(GLFWwindow*,int);
  *
  *  @ingroup input
  */
-typedef void (* GLFWscrollfun)(GLFWwindow*,double,double);
+typedef void (* GLFWscrollfun)(GLFWwindow*,double*,double*);

 /*! @brief The function pointer type for keyboard key callbacks.
  *
```

```diff
diff --git a/tmp/glfw-3.3.3/src/input.c b/./internal/glfw/glfw/src/input.c
index 337d5cf0..4ac555cb 100644
--- a/tmp/glfw-3.3.3/src/input.c
+++ b/./internal/glfw/glfw/src/input.c
@@ -312,7 +312,7 @@ void _glfwInputChar(_GLFWwindow* window, unsigned int codepoint, int mods, GLFWb
 void _glfwInputScroll(_GLFWwindow* window, double xoffset, double yoffset)
 {
     if (window->callbacks.scroll)
-        window->callbacks.scroll((GLFWwindow*) window, xoffset, yoffset);
+        window->callbacks.scroll((GLFWwindow*) window, &xoffset, &yoffset);
 }

 // Notifies shared code of a mouse button click event
```

A fullscreened window doesn't float (#1506, glfw/glfw#1967).

```diff
diff --git a/tmp/glfw-3.3.4/src/win32_window.c b/./internal/glfw/glfw/src/win32_window.c
index d17b6da4..17e2f842 100644
--- a/tmp/glfw-3.3.4/src/win32_window.c
+++ b/./internal/glfw/glfw/src/win32_window.c
@@ -68,7 +68,7 @@ static DWORD getWindowExStyle(const _GLFWwindow* window)
 {
     DWORD style = WS_EX_APPWINDOW;
 
-    if (window->monitor || window->floating)
+    if (window->floating)
         style |= WS_EX_TOPMOST;
 
     return style;
@@ -436,7 +436,7 @@ static void fitToMonitor(_GLFWwindow* window)
 {
     MONITORINFO mi = { sizeof(mi) };
     GetMonitorInfo(window->monitor->win32.handle, &mi);
-    SetWindowPos(window->win32.handle, HWND_TOPMOST,
+    SetWindowPos(window->win32.handle, window->floating ? HWND_TOPMOST : HWND_NOTOPMOST,
                  mi.rcMonitor.left,
                  mi.rcMonitor.top,
                  mi.rcMonitor.right - mi.rcMonitor.left,
@@ -1756,7 +1756,7 @@ void _glfwPlatformSetWindowMonitor(_GLFWwindow* window,
         acquireMonitor(window);
 
         GetMonitorInfo(window->monitor->win32.handle, &mi);
-        SetWindowPos(window->win32.handle, HWND_TOPMOST,
+        SetWindowPos(window->win32.handle, window->floating ? HWND_TOPMOST : HWND_NOTOPMOST,
                      mi.rcMonitor.left,
                      mi.rcMonitor.top,
                      mi.rcMonitor.right - mi.rcMonitor.left,
```
