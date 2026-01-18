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
import android.opengl.GLSurfaceView;
import android.os.Handler;
import android.os.Looper;
import android.util.AttributeSet;
import android.util.Log;

import javax.microedition.khronos.egl.EGLConfig;
import javax.microedition.khronos.opengles.GL10;

import $Placeholder_JavaPkg$.ebitenmobileview.Ebitenmobileview;
import $Placeholder_JavaPkg$.ebitenmobileview.Renderer;
import $Placeholder_JavaPkg$.$Placeholder_PrefixLower$.EbitenView;

class EbitenSurfaceView extends GLSurfaceView implements Renderer {
    // As GLSurfaceView can be recreated, the states must be static (#3097).
    static private boolean errored_ = false;
    static private boolean onceSurfaceCreated_ = false;

    private class EbitenRenderer implements GLSurfaceView.Renderer {
        @Override
        public void onDrawFrame(GL10 gl) {
            if (errored_) {
                return;
            }
            try {
                Ebitenmobileview.update();
            } catch (final Exception e) {
                new Handler(Looper.getMainLooper()).post(new Runnable() {
                    @Override
                    public void run() {
                        onErrorOnGameUpdate(e);
                    }
                });
                errored_ = true;
            }
        }

        @Override
        public void onSurfaceCreated(GL10 gl, EGLConfig config) {
            if (!onceSurfaceCreated_) {
                onceSurfaceCreated_ = true;
                return;
            }
            if (Ebitenmobileview.onContextLost()) {
                Log.i("Go", "The OpenGL context was lost and restored");
                return;
            }
            Log.e("Go", "The application was killed due to context loss");
            Runtime.getRuntime().exit(0);
        }

        @Override
        public void onSurfaceChanged(GL10 gl, int width, int height) {
        }
    }

    public EbitenSurfaceView(Context context) {
        super(context);
        initialize();
    }

    public EbitenSurfaceView(Context context, AttributeSet attrs) {
        super(context, attrs);
        initialize();
    }

    private void initialize() {
        setEGLContextClientVersion(3);
        setEGLConfigChooser(8, 8, 8, 8, 0, 0);
        setPreserveEGLContextOnPause(true);
        // setRenderer must be called before Ebitenmobileview.setRenderer.
        // Otherwise, setRenderMode in setExplicitRenderingMode will crash.
        setRenderer(new EbitenRenderer());

        Ebitenmobileview.setRenderer(this);
    }

    private void onErrorOnGameUpdate(Exception e) {
        ((EbitenView)getParent()).onErrorOnGameUpdate(e);
    }

    @Override
    public synchronized void setExplicitRenderingMode(boolean explicitRendering) {
        // TODO: Remove this logic when FPSModeVsyncOffMinimum is removed.
        // This doesn't work when EbitenSurfaceView is recreated anyway.
        if (explicitRendering) {
            setRenderMode(RENDERMODE_WHEN_DIRTY);
        } else {
            setRenderMode(RENDERMODE_CONTINUOUSLY);
        }
    }

    @Override
    public synchronized void requestRenderIfNeeded() {
        if (getRenderMode() == RENDERMODE_WHEN_DIRTY) {
            requestRender();
        }
    }
}
