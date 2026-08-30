// Copyright 2022 The Ebitengine Authors
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

package $Placeholder_JavaPkg$.$Placeholder_PrefixLower$;

import android.content.Context;
import android.graphics.Rect;
import android.opengl.EGL14;
import android.opengl.EGLConfig;
import android.opengl.EGLContext;
import android.opengl.EGLDisplay;
import android.opengl.EGLSurface;
import android.os.Handler;
import android.os.Looper;
import android.util.AttributeSet;
import android.util.Log;
import android.view.Surface;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.View;

import $Placeholder_JavaPkg$.ebitenmobileview.Ebitenmobileview;
import $Placeholder_JavaPkg$.ebitenmobileview.Renderer;
import $Placeholder_JavaPkg$.$Placeholder_PrefixLower$.EbitenView;

// EbitenSurfaceView renders a game on a surface with OpenGL ES.
//
// EbitenSurfaceView doesn't use GLSurfaceView, for GLSurfaceView can lose an OpenGL ES context
// although a context does not have to be lost (#3371). Losing the context means losing all the GPU
// objects the Go side holds in the context. Instead, EbitenSurfaceView manages an EGL context on
// its own render thread: the context is kept as long as the process lives, and only the EGL surface
// is recreated when the Android surface is recreated.
class EbitenSurfaceView extends SurfaceView implements SurfaceHolder.Callback, Renderer {
    // renderThread_ is shared with all the EbitenSurfaceView instances and is never terminated, as
    // the EGL context is created on the thread and must not be lost even when all the
    // EbitenSurfaceView instances are destroyed (#3097, #3371).
    private static final RenderThread renderThread_ = new RenderThread();

    // paused_ is set while rendering is suspended by onPause. This is kept per view for a new view
    // can take rendering over from another view.
    //
    // paused_ is guarded by renderThread_'s lock.
    private boolean paused_ = false;

    public EbitenSurfaceView(Context context) {
        super(context);
        initialize();
    }

    public EbitenSurfaceView(Context context, AttributeSet attrs) {
        super(context, attrs);
        initialize();
    }

    private void initialize() {
        // Keep the screen on while the game is rendered, as GLSurfaceView did.
        setKeepScreenOn(true);

        getHolder().addCallback(this);

        renderThread_.startIfNeeded();
        Ebitenmobileview.setRenderer(this);
    }

    @Override
    public void surfaceCreated(SurfaceHolder holder) {
        final Rect frame = holder.getSurfaceFrame();
        renderThread_.setSurface(this, holder.getSurface(), frame.width(), frame.height());
    }

    @Override
    public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {
        renderThread_.setSurface(this, holder.getSurface(), width, height);
    }

    @Override
    public void surfaceDestroyed(SurfaceHolder holder) {
        // The surface must not be used after this method returns, so this waits until the render
        // thread stops using the surface.
        renderThread_.unsetSurface(this);
    }

    @Override
    protected void onWindowVisibilityChanged(int visibility) {
        super.onWindowVisibilityChanged(visibility);
        renderThread_.setWindowVisible(this, visibility == View.VISIBLE);
    }

    // onPause suspends rendering on the surface.
    // This doesn't have to be paired with onResume: the Android surface lifecycle also suspends and
    // resumes rendering.
    public void onPause() {
        renderThread_.setPaused(this, true);
    }

    // onResume resumes rendering on the surface suspended by onPause.
    public void onResume() {
        renderThread_.setPaused(this, false);
    }

    @Override
    public void setExplicitRenderingMode(boolean explicitRendering) {
        renderThread_.setExplicitRenderingMode(explicitRendering);
    }

    @Override
    public void requestRenderIfNeeded() {
        renderThread_.requestRenderIfNeeded();
    }

    // reportErrorOnGameUpdate reports an error happened on the render thread to the game.
    private void reportErrorOnGameUpdate(final Exception e) {
        new Handler(Looper.getMainLooper()).post(new Runnable() {
            @Override
            public void run() {
                ((EbitenView)getParent()).onErrorOnGameUpdate(e);
            }
        });
    }

    private static class RenderThread extends Thread {
        // Rendering is stopped when it fails this number of times in a row, assuming the rendering
        // cannot succeed anymore.
        private static final int MAX_RENDER_FAILURES = 100;

        // The wait times in milliseconds to retry rendering. The maximum time must be short enough
        // not to block surfaceDestroyed, which waits for the render thread.
        private static final int RETRY_WAIT_MILLIS = 16;
        private static final int MAX_RETRY_WAIT_MILLIS = 100;

        // lock_ guards the fields below and EbitenSurfaceView.paused_.
        private final Object lock_ = new Object();

        // renderLock_ is held while the EGL surface is used, so that the UI thread can wait for the
        // in-flight frame before the Android surface is destroyed.
        private final Object renderLock_ = new Object();

        // The followings are guarded by lock_.

        private boolean started_ = false;

        // errored_ becomes true when the game or the rendering cannot continue anymore. As
        // EbitenSurfaceView can be recreated, this is not reset even after the view is destroyed.
        private boolean errored_ = false;

        // view_ and surface_ are the view and the surface rendering on. surface_ is null when no
        // surface is available.
        private EbitenSurfaceView view_ = null;
        private Surface surface_ = null;
        private int width_ = 0;
        private int height_ = 0;

        private boolean viewPaused_ = false;
        private boolean windowVisible_ = true;

        // explicitRendering_ corresponds to FPSModeVsyncOffMinimum. Without explicit rendering,
        // frames are rendered as long as a surface is available, and the buffer swapping waits for
        // a vsync.
        private boolean explicitRendering_ = false;
        private boolean renderingRequested_ = false;

        // surfaceReleased_ reports whether the EGL surface is not used. The UI thread waits for this
        // when the Android surface is destroyed.
        private boolean surfaceReleased_ = true;

        // The followings are used only on this thread.

        private EGLDisplay display_ = EGL14.EGL_NO_DISPLAY;
        private EGLConfig config_ = null;
        private EGLContext context_ = EGL14.EGL_NO_CONTEXT;

        // window_ is the EGL surface created for windowOwner_, and is EGL14.EGL_NO_SURFACE when no
        // EGL surface is used.
        private EGLSurface window_ = EGL14.EGL_NO_SURFACE;
        private Surface windowOwner_ = null;

        private int reportedWidth_ = 0;
        private int reportedHeight_ = 0;

        // contextLost_ becomes true when the GPU reported a context lost, and contextRecreated_
        // becomes true just after the EGL context is recreated for the lost context.
        private boolean contextLost_ = false;
        private boolean contextRecreated_ = false;

        // renderFailures_ is the number of the frames which failed to render in a row.
        private int renderFailures_ = 0;

        RenderThread() {
            super("EbitenRenderThread");
            // The thread never exits, so this must be a daemon thread not to prevent the process
            // from exiting.
            setDaemon(true);
        }

        void startIfNeeded() {
            synchronized (lock_) {
                if (started_) {
                    return;
                }
                started_ = true;
            }
            start();
        }

        void setSurface(EbitenSurfaceView view, Surface surface, int width, int height) {
            synchronized (lock_) {
                // A newly created surface takes rendering over.
                view_ = view;
                viewPaused_ = view.paused_;
                windowVisible_ = view.getWindowVisibility() == View.VISIBLE;
                surface_ = surface;
                width_ = width;
                height_ = height;
                // A new surface must be rendered at least once even in the explicit rendering mode,
                // for the surface is initialized by rendering.
                renderingRequested_ = true;
                lock_.notifyAll();
            }
        }

        void unsetSurface(EbitenSurfaceView view) {
            synchronized (lock_) {
                if (view_ != view) {
                    return;
                }
                // Stop rendering on the surface first, and then wait until the surface is released.
                view_ = null;
                surface_ = null;
                // The render thread can be waiting until a frame is requested, and it must notice
                // the surface destruction even in that case.
                lock_.notifyAll();
            }
            synchronized (renderLock_) {
                synchronized (lock_) {
                    while (!surfaceReleased_) {
                        try {
                            lock_.wait();
                        } catch (InterruptedException e) {
                            return;
                        }
                    }
                }
            }
        }

        void setPaused(EbitenSurfaceView view, boolean paused) {
            synchronized (lock_) {
                view.paused_ = paused;
                if (view_ != view) {
                    return;
                }
                viewPaused_ = paused;
                lock_.notifyAll();
            }
        }

        void setWindowVisible(EbitenSurfaceView view, boolean windowVisible) {
            synchronized (lock_) {
                if (view_ != view) {
                    return;
                }
                windowVisible_ = windowVisible;
                lock_.notifyAll();
            }
        }

        void setExplicitRenderingMode(boolean explicitRendering) {
            synchronized (lock_) {
                explicitRendering_ = explicitRendering;
                lock_.notifyAll();
            }
        }

        void requestRenderIfNeeded() {
            synchronized (lock_) {
                if (!explicitRendering_) {
                    return;
                }
                renderingRequested_ = true;
                lock_.notifyAll();
            }
        }

        private void setErroredLocked() {
            errored_ = true;
            lock_.notifyAll();
        }

        // hasSurfaceLocked reports whether a surface is ready to render on no matter whether a
        // frame is requested.
        private boolean hasSurfaceLocked() {
            return !errored_ && view_ != null && surface_ != null && !viewPaused_ && windowVisible_ &&
                width_ > 0 && height_ > 0;
        }

        private boolean canRenderLocked() {
            if (!hasSurfaceLocked()) {
                return false;
            }
            return !explicitRendering_ || renderingRequested_;
        }

        @Override
        public void run() {
            while (true) {
                synchronized (lock_) {
                    while (!canRenderLocked()) {
                        // Release the EGL surface when nothing is rendered, so that the Android
                        // surface is not used while e.g. the game is suspended.
                        if (!hasSurfaceLocked()) {
                            releaseWindowSurface();
                            surfaceReleased_ = true;
                        }
                        lock_.notifyAll();
                        try {
                            lock_.wait();
                        } catch (InterruptedException e) {
                            setErroredLocked();
                            releaseWindowSurface();
                            surfaceReleased_ = true;
                            return;
                        }
                    }
                }

                synchronized (renderLock_) {
                    final Surface surface;
                    final int width;
                    final int height;
                    synchronized (lock_) {
                        if (!canRenderLocked()) {
                            continue;
                        }
                        renderingRequested_ = false;
                        // The UI thread must not destroy the Android surface until the EGL surface
                        // is released.
                        surfaceReleased_ = false;
                        surface = surface_;
                        width = width_;
                        height = height_;
                    }
                    try {
                        render(surface, width, height);
                    } catch (Throwable t) {
                        Log.e("Go", "rendering failed: " + Log.getStackTraceString(t));
                        synchronized (lock_) {
                            setErroredLocked();
                        }
                    }
                }
            }
        }

        // render renders a frame on the given surface.
        //
        // render must be called while renderLock_ is held.
        private void render(Surface surface, int width, int height) {
            synchronized (lock_) {
                // The surface can have been destroyed or taken over by another view while
                // renderLock_ was being acquired. Rendering on such a surface fails, so skip this
                // frame.
                if (surface_ != surface) {
                    return;
                }
            }

            if (contextLost_) {
                // The GPU was reset and the EGL context is no longer valid. Recreate one.
                contextLost_ = false;
                contextRecreated_ = true;
                releaseWindowSurface();
                if (context_ != EGL14.EGL_NO_CONTEXT) {
                    EGL14.eglDestroyContext(display_, context_);
                    context_ = EGL14.EGL_NO_CONTEXT;
                }
            }

            if (!initializeContext()) {
                return;
            }

            if (window_ == EGL14.EGL_NO_SURFACE || windowOwner_ != surface) {
                // The Android surface can already be destroyed although surfaceDestroyed is not
                // called yet.
                if (!surface.isValid()) {
                    sleepBriefly(RETRY_WAIT_MILLIS);
                    return;
                }
                // The surface can be another view's one, which has taken rendering over.
                releaseWindowSurface();
                int[] windowAttribs = new int[] {EGL14.EGL_NONE};
                window_ = EGL14.eglCreateWindowSurface(display_, config_, surface, windowAttribs, 0);
                if (window_ == EGL14.EGL_NO_SURFACE) {
                    final int error = EGL14.eglGetError();
                    onRenderFailure("eglCreateWindowSurface failed: 0x" + Integer.toHexString(error));
                    return;
                }
                windowOwner_ = surface;
                // The size must be reported again for the new EGL surface.
                reportedWidth_ = 0;
                reportedHeight_ = 0;
                renderFailures_ = 0;
            }

            if (!EGL14.eglMakeCurrent(display_, window_, window_, context_)) {
                onRenderFailure("eglMakeCurrent failed: 0x" + Integer.toHexString(EGL14.eglGetError()));
                return;
            }

            // Wait for a vsync on swapping buffers, as GLSurfaceView did.
            // A failure here is not fatal, so the result is ignored.
            EGL14.eglSwapInterval(display_, 1);

            if (contextRecreated_) {
                contextRecreated_ = false;
                if (!Ebitenmobileview.onContextLost()) {
                    Log.e("Go", "The application was killed due to context loss");
                    Runtime.getRuntime().exit(0);
                }
                Log.i("Go", "The OpenGL context was lost and restored");
            }

            if (width != reportedWidth_ || height != reportedHeight_) {
                reportedWidth_ = width;
                reportedHeight_ = height;
                // The surface's size in pixels is exact, unlike a size converted to and back from
                // density-independent pixels.
                Ebitenmobileview.onSurfaceChanged(width, height);
            }

            try {
                Ebitenmobileview.update();
            } catch (final Exception e) {
                final EbitenSurfaceView view;
                synchronized (lock_) {
                    view = view_;
                    setErroredLocked();
                }
                if (view != null) {
                    view.reportErrorOnGameUpdate(e);
                }
                return;
            }

            if (!EGL14.eglSwapBuffers(display_, window_)) {
                final int error = EGL14.eglGetError();
                if (error == EGL14.EGL_CONTEXT_LOST) {
                    Log.e("Go", "The OpenGL ES context was lost");
                    contextLost_ = true;
                    return;
                }
                onRenderFailure("eglSwapBuffers failed: 0x" + Integer.toHexString(error));
                return;
            }
            renderFailures_ = 0;
        }

        // onRenderFailure handles a failure to render a frame.
        //
        // A failure can happen just because the surface is being destroyed, so rendering is not
        // stopped at once. Repeated failures mean that rendering cannot succeed anymore.
        private void onRenderFailure(String message) {
            Log.e("Go", message);
            // The EGL surface can be no longer usable. Recreate it in the next frame.
            releaseWindowSurface();
            if (++renderFailures_ < MAX_RENDER_FAILURES) {
                // Wait not to busy loop. The wait must be short enough not to block
                // surfaceDestroyed, which waits for the render thread.
                sleepBriefly(Math.min((long) RETRY_WAIT_MILLIS * renderFailures_, MAX_RETRY_WAIT_MILLIS));
                return;
            }
            Log.e("Go", "Rendering is stopped because it kept failing");
            synchronized (lock_) {
                setErroredLocked();
            }
        }

        private void sleepBriefly(long millis) {
            try {
                Thread.sleep(millis);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }

        // initializeContext initializes the EGL display and creates the EGL context if needed.
        //
        // The EGL context is not destroyed unless the context is lost, for the Go side holds GPU
        // objects belonging to the context.
        private boolean initializeContext() {
            if (context_ != EGL14.EGL_NO_CONTEXT) {
                return true;
            }

            if (display_ == EGL14.EGL_NO_DISPLAY) {
                display_ = EGL14.eglGetDisplay(EGL14.EGL_DEFAULT_DISPLAY);
                if (display_ == EGL14.EGL_NO_DISPLAY) {
                    Log.e("Go", "eglGetDisplay failed: 0x" + Integer.toHexString(EGL14.eglGetError()));
                    synchronized (lock_) {
                        setErroredLocked();
                    }
                    return false;
                }
                final int[] version = new int[2];
                if (!EGL14.eglInitialize(display_, version, 0, version, 1)) {
                    Log.e("Go", "eglInitialize failed: 0x" + Integer.toHexString(EGL14.eglGetError()));
                    synchronized (lock_) {
                        setErroredLocked();
                    }
                    return false;
                }
                config_ = chooseConfig(display_);
                if (config_ == null) {
                    Log.e("Go", "No suitable EGLConfig found; RGB8888 and RGB565 formats are not supported");
                    synchronized (lock_) {
                        setErroredLocked();
                    }
                    return false;
                }
            }

            context_ = EGL14.eglCreateContext(display_, config_, EGL14.EGL_NO_CONTEXT,
                new int[] {
                    EGL14.EGL_CONTEXT_CLIENT_VERSION, 3,
                    EGL14.EGL_NONE,
                }, 0);
            if (context_ == EGL14.EGL_NO_CONTEXT) {
                Log.e("Go", "eglCreateContext failed: 0x" + Integer.toHexString(EGL14.eglGetError()));
                synchronized (lock_) {
                    setErroredLocked();
                }
                return false;
            }
            return true;
        }

        // releaseWindowSurface releases the EGL surface, but not the EGL context.
        private void releaseWindowSurface() {
            if (window_ == EGL14.EGL_NO_SURFACE) {
                return;
            }
            // Unbind the context so that the context is not current on a destroyed surface.
            EGL14.eglMakeCurrent(display_, EGL14.EGL_NO_SURFACE, EGL14.EGL_NO_SURFACE, EGL14.EGL_NO_CONTEXT);
            EGL14.eglDestroySurface(display_, window_);
            window_ = EGL14.EGL_NO_SURFACE;
            windowOwner_ = null;
        }

        private static EGLConfig chooseConfig(EGLDisplay display) {
            final EGLConfig[] configs = new EGLConfig[1];
            final int[] numConfig = new int[1];

            int[] attribs = new int[] {
                EGL14.EGL_RED_SIZE, 8,
                EGL14.EGL_GREEN_SIZE, 8,
                EGL14.EGL_BLUE_SIZE, 8,
                EGL14.EGL_ALPHA_SIZE, 8,
                EGL14.EGL_DEPTH_SIZE, 0,
                EGL14.EGL_STENCIL_SIZE, 0,
                EGL14.EGL_SURFACE_TYPE, EGL14.EGL_WINDOW_BIT,
                // ES2 bit is often compatible with ES3.
                EGL14.EGL_RENDERABLE_TYPE, EGL14.EGL_OPENGL_ES2_BIT,
                EGL14.EGL_NONE,
            };
            if (EGL14.eglChooseConfig(display, attribs, 0, configs, 0, 1, numConfig, 0) && numConfig[0] > 0) {
                return configs[0];
            }

            // Try the fallback config.
            attribs = new int[] {
                EGL14.EGL_RED_SIZE, 5,
                EGL14.EGL_GREEN_SIZE, 6,
                EGL14.EGL_BLUE_SIZE, 5,
                EGL14.EGL_SURFACE_TYPE, EGL14.EGL_WINDOW_BIT,
                EGL14.EGL_RENDERABLE_TYPE, EGL14.EGL_OPENGL_ES2_BIT,
                EGL14.EGL_NONE,
            };
            if (EGL14.eglChooseConfig(display, attribs, 0, configs, 0, 1, numConfig, 0) && numConfig[0] > 0) {
                return configs[0];
            }

            return null;
        }
    }
}
