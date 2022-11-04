// Code generated by ebitenmobile. DO NOT EDIT.

package {{.JavaPkg}}.{{.PrefixLower}};

import android.content.Context;
import android.opengl.GLSurfaceView;
import android.os.Handler;
import android.os.Looper;
import android.util.AttributeSet;
import android.util.Log;

import javax.microedition.khronos.egl.EGLConfig;
import javax.microedition.khronos.opengles.GL10;

import {{.JavaPkg}}.ebitenmobileview.Ebitenmobileview;
import {{.JavaPkg}}.ebitenmobileview.RenderRequester;
import {{.JavaPkg}}.{{.PrefixLower}}.EbitenView;

class EbitenSurfaceView extends GLSurfaceView implements RenderRequester {

    private class EbitenRenderer implements GLSurfaceView.Renderer {

        private boolean errored_ = false;

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
            Ebitenmobileview.onContextLost();
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
        setEGLContextClientVersion(2);
        setEGLConfigChooser(8, 8, 8, 8, 0, 0);
        setRenderer(new EbitenRenderer());
        Ebitenmobileview.setRenderRequester(this);
    }

    private void onErrorOnGameUpdate(Exception e) {
        ((EbitenView)getParent()).onErrorOnGameUpdate(e);
    }

    @Override
    public synchronized void setExplicitRenderingMode(boolean explicitRendering) {
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
