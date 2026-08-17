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

import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;

import android.content.Context;
import android.graphics.Insets;
import android.graphics.Rect;
import android.hardware.input.InputManager;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.text.Editable;
import android.text.InputType;
import android.text.TextWatcher;
import android.util.AttributeSet;
import android.util.DisplayMetrics;
import android.util.Log;
import android.view.Display;
import android.view.KeyCharacterMap;
import android.view.KeyEvent;
import android.view.InputDevice;
import android.view.MotionEvent;
import android.view.View;
import android.view.ViewGroup;
import android.view.ViewTreeObserver;
import android.view.WindowInsets;
import android.view.WindowManager;
import android.view.inputmethod.BaseInputConnection;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputConnection;
import android.view.inputmethod.InputConnectionWrapper;
import android.view.inputmethod.InputMethodManager;
import android.view.inputmethod.TextAttribute;
import android.widget.EditText;

import go.Seq;

import $Placeholder_JavaPkg$.ebitenmobileview.Ebitenmobileview;
import $Placeholder_JavaPkg$.ebitenmobileview.TextInputDriver;

public class EbitenView extends ViewGroup implements InputManager.InputDeviceListener, TextInputDriver {
    static class Gamepad {
        public int deviceId;
        public ArrayList<InputDevice.MotionRange> axes;
        public ArrayList<InputDevice.MotionRange> hats;
    }

    // See https://github.com/libsdl-org/SDL/blob/2df2da11f627299c6e05b7e0aff407c915043372/android-project/app/src/main/java/org/libsdl/app/SDLControllerManager.java#L154-L173
    static class RangeComparator implements Comparator<InputDevice.MotionRange> {
        @Override
        public int compare(InputDevice.MotionRange arg0, InputDevice.MotionRange arg1) {
            int arg0Axis = arg0.getAxis();
            int arg1Axis = arg1.getAxis();
            if (arg0Axis == MotionEvent.AXIS_GAS) {
                arg0Axis = MotionEvent.AXIS_BRAKE;
            } else if (arg0Axis == MotionEvent.AXIS_BRAKE) {
                arg0Axis = MotionEvent.AXIS_GAS;
            }
            if (arg1Axis == MotionEvent.AXIS_GAS) {
                arg1Axis = MotionEvent.AXIS_BRAKE;
            } else if (arg1Axis == MotionEvent.AXIS_BRAKE) {
                arg1Axis = MotionEvent.AXIS_GAS;
            }

            // Make sure the AXIS_Z is sorted between AXIS_RY and AXIS_RZ.
            //
            // The value ordering on Android otherwise is AXIS_X, AXIS_Y,
            // all kinds of axes used by touchscreens or touchpads only,
            // AXIS_Z, AXIS_RX, AXIS_RY, AXIS_RZ, hats, triggers,
            // flight controls, car controls, misc stuff.
            //
            // This is because the usual pairing are:
            // - AXIS_X, AXIS_Y (left stick).
            // - AXIS_RX, AXIS_RY (sometimes the right stick, sometimes triggers).
            // - AXIS_Z, AXIS_RZ (sometimes the right stick, sometimes triggers).
            //
            // This sorts the axes in the above order, which tends to be correct
            // for Xbox-ish game pads that have the right stick on RX/RY and the
            // triggers on Z/RZ.
            //
            // Gamepads that don't have AXIS_Z/AXIS_RZ but use
            // AXIS_LTRIGGER/AXIS_RTRIGGER are unaffected by this.
            //
            // References:
            // - https://developer.android.com/develop/ui/views/touch-and-input/game-controllers/controller-input
            // - https://www.kernel.org/doc/html/latest/input/gamepad.html
            if (arg0Axis == MotionEvent.AXIS_Z) {
                arg0Axis = MotionEvent.AXIS_RZ - 1;
            } else if (arg0Axis > MotionEvent.AXIS_Z && arg0Axis < MotionEvent.AXIS_RZ) {
                arg0Axis--;
            }
            if (arg1Axis == MotionEvent.AXIS_Z) {
                arg1Axis = MotionEvent.AXIS_RZ - 1;
            } else if (arg1Axis > MotionEvent.AXIS_Z && arg1Axis < MotionEvent.AXIS_RZ) {
                arg1Axis--;
            }

            return arg0Axis - arg1Axis;
        }
    }

    private double pxToDp(double x) {
        return x / this.deviceScale;
    }

    private void updateDeviceScale() {
        WindowManager windowManager = (WindowManager)getContext().getSystemService(Context.WINDOW_SERVICE);
        Display display = windowManager.getDefaultDisplay();
        DisplayMetrics metrics = new DisplayMetrics();
        display.getRealMetrics(metrics);
        this.deviceScale = metrics.density;
    }

    public EbitenView(Context context) {
        super(context);
        initialize(context);
    }

    public EbitenView(Context context, AttributeSet attrs) {
        super(context, attrs);
        initialize(context);
    }

    private void initialize(Context context) {
        // Set the Android context for the Go side, which requires it e.g. to
        // get the display information or to vibrate. Seq.setContext leaks a
        // JNI global reference per call, so set the context at most once.
        if (!seqContextSet) {
            seqContextSet = true;
            Seq.setContext(context.getApplicationContext());
        }

        updateDeviceScale();

        this.gamepads = new ArrayList<Gamepad>();

        this.ebitenSurfaceView = new EbitenSurfaceView(getContext());
        LayoutParams params = new LayoutParams(LayoutParams.MATCH_PARENT, LayoutParams.MATCH_PARENT);
        addView(this.ebitenSurfaceView, params);

        this.inputManager = (InputManager)context.getSystemService(Context.INPUT_SERVICE);
        this.inputManager.registerInputDeviceListener(this, null);
        for (int id : this.inputManager.getInputDeviceIds()) {
            this.onInputDeviceAdded(id);
        }

        this.editText = new EbitenEditText(context);
        addView(this.editText, new LayoutParams(1, 1));
        Ebitenmobileview.setTextInputDriver(this);
        getViewTreeObserver().addOnGlobalLayoutListener(new ViewTreeObserver.OnGlobalLayoutListener() {
            @Override
            public void onGlobalLayout() {
                reportVirtualKeyboardState();
            }
        });
    }

    @Override
    protected void onLayout(boolean changed, int left, int top, int right, int bottom) {
        this.ebitenSurfaceView.layout(0, 0, right - left, bottom - top);
        layoutEditText();
        updateDeviceScale();
        double widthInDp = pxToDp(right - left);
        double heightInDp = pxToDp(bottom - top);
        Ebitenmobileview.layout(widthInDp, heightInDp);
    }

    // layoutEditText places the edit text at the game's caret bounds, so that
    // panning for the virtual keyboard and the IME's candidate UI are anchored
    // to the caret. The edit text is fully transparent and takes no touches.
    private void layoutEditText() {
        int x = this.caretX;
        int y = this.caretY;
        int width = Math.max(this.caretWidth, 1);
        int height = Math.max(this.caretHeight, 1);
        this.editText.layout(x, y, x + width, y + height);
    }

    @Override
    public boolean onKeyDown(int keyCode, KeyEvent event) {
        // Ignore key repeat events.
        if (event.getRepeatCount() > 0) {
            return super.onKeyDown(keyCode, event);
        }
        Ebitenmobileview.onKeyDownOnAndroid(keyCode, event.getUnicodeChar(), event.getSource(), event.getDeviceId(), event.getMetaState());
        return true;
    }

    @Override
    public boolean onKeyUp(int keyCode, KeyEvent event) {
        Ebitenmobileview.onKeyUpOnAndroid(keyCode, event.getSource(), event.getDeviceId(), event.getMetaState());
        return true;
    }

    @Override
    public boolean onTouchEvent(MotionEvent e) {
        // getActionIndex returns a valid value only for the action whose index is the returned value of getActionIndex (#2220).
        // See https://developer.android.com/reference/android/view/MotionEvent#getActionMasked().
        // For other pointers, treat their actions as MotionEvent.ACTION_MOVE.
        int touchIndex = e.getActionIndex();
        for (int i = 0; i < e.getPointerCount(); i++) {
            int id = e.getPointerId(i);
            double x = e.getX(i);
            double y = e.getY(i);
            int action = (i == touchIndex) ? e.getActionMasked() : MotionEvent.ACTION_MOVE;
            Ebitenmobileview.updateTouchesOnAndroid(action, id, pxToDp(x), pxToDp(y));
        }
        return true;
    }

    private Gamepad getGamepad(int deviceId) {
        for (Gamepad gamepad : this.gamepads) {
            if (gamepad.deviceId == deviceId) {
                return gamepad;
            }
        }
        return null;
    }

    @Override
    public boolean onGenericMotionEvent(MotionEvent event) {
        if ((event.getSource() & InputDevice.SOURCE_JOYSTICK) != InputDevice.SOURCE_JOYSTICK) {
            return super.onGenericMotionEvent(event);
        }
        if (event.getAction() != MotionEvent.ACTION_MOVE) {
            return super.onGenericMotionEvent(event);
        }

        // See https://github.com/libsdl-org/SDL/blob/2df2da11f627299c6e05b7e0aff407c915043372/android-project/app/src/main/java/org/libsdl/app/SDLControllerManager.java#L256-L277
        Gamepad gamepad = this.getGamepad(event.getDeviceId());
        if (gamepad == null) {
            return true;
        }

        int actionPointerIndex = event.getActionIndex();
        for (int i = 0; i < gamepad.axes.size(); i++) {
            InputDevice.MotionRange range = gamepad.axes.get(i);
            float axisValue = event.getAxisValue(range.getAxis(), actionPointerIndex);
            float value = (axisValue - range.getMin()) / range.getRange() * 2.0f - 1.0f;
            Ebitenmobileview.onGamepadAxisChanged(gamepad.deviceId, i, value);
        }
        for (int i = 0; i < gamepad.hats.size() / 2; i++) {
            int hatX = Math.round(event.getAxisValue(gamepad.hats.get(2*i).getAxis(), actionPointerIndex));
            int hatY = Math.round(event.getAxisValue(gamepad.hats.get(2*i+1).getAxis(), actionPointerIndex));
            Ebitenmobileview.onGamepadHatChanged(gamepad.deviceId, i, hatX, hatY);
        }
        return true;
    }

    @Override
    public void onInputDeviceAdded(int deviceId) {
        InputDevice inputDevice = this.inputManager.getInputDevice(deviceId);
        // The InputDevice can be null on some deivces (#1342).
        if (inputDevice == null) {
            return;
        }

        // A fingerprint reader is unexpectedly recognized as a joystick. Skip this (#1542).
        if (inputDevice.getName().equals("uinput-fpc")) {
            return;
        }

        int sources = inputDevice.getSources();
        if ((sources & InputDevice.SOURCE_GAMEPAD) != InputDevice.SOURCE_GAMEPAD &&
            (sources & InputDevice.SOURCE_JOYSTICK) != InputDevice.SOURCE_JOYSTICK) {
            return;
        }

        // See https://github.com/libsdl-org/SDL/blob/2df2da11f627299c6e05b7e0aff407c915043372/android-project/app/src/main/java/org/libsdl/app/SDLControllerManager.java#L182-L216
        List<InputDevice.MotionRange> ranges = inputDevice.getMotionRanges();
        Collections.sort(ranges, new RangeComparator());

        Gamepad gamepad = new Gamepad();
        gamepad.deviceId = deviceId;
        gamepad.axes = new ArrayList<InputDevice.MotionRange>();
        gamepad.hats = new ArrayList<InputDevice.MotionRange>();
        for (InputDevice.MotionRange range : ranges) {
            if (range.getAxis() == MotionEvent.AXIS_HAT_X || range.getAxis() == MotionEvent.AXIS_HAT_Y) {
                gamepad.hats.add(range);
            } else {
                gamepad.axes.add(range);
            }
        }
        this.gamepads.add(gamepad);

        String descriptor = inputDevice.getDescriptor();
        int vendorId = inputDevice.getVendorId();
        int productId = inputDevice.getProductId();

        // These values are required to calculate SDL's GUID.
        int buttonMask = getButtonMask(inputDevice, gamepad.hats.size()/2);
        int axisMask = getAxisMask(inputDevice);

        Ebitenmobileview.onGamepadAdded(deviceId, inputDevice.getName(), gamepad.axes.size(), gamepad.hats.size()/2, descriptor, vendorId, productId, buttonMask, axisMask);
    }

    // The implementation is copied from SDL:
    // https://github.com/libsdl-org/SDL/blob/0e9560aea22818884921e5e5064953257bfe7fa7/android-project/app/src/main/java/org/libsdl/app/SDLControllerManager.java#L308
    private static int getButtonMask(InputDevice joystickDevice, int nhats) {
        int buttonMask = 0;
        int[] keys = new int[] {
            KeyEvent.KEYCODE_BUTTON_A,
            KeyEvent.KEYCODE_BUTTON_B,
            KeyEvent.KEYCODE_BUTTON_X,
            KeyEvent.KEYCODE_BUTTON_Y,
            KeyEvent.KEYCODE_BACK,
            KeyEvent.KEYCODE_BUTTON_MODE,
            KeyEvent.KEYCODE_BUTTON_START,
            KeyEvent.KEYCODE_BUTTON_THUMBL,
            KeyEvent.KEYCODE_BUTTON_THUMBR,
            KeyEvent.KEYCODE_BUTTON_L1,
            KeyEvent.KEYCODE_BUTTON_R1,
            KeyEvent.KEYCODE_DPAD_UP,
            KeyEvent.KEYCODE_DPAD_DOWN,
            KeyEvent.KEYCODE_DPAD_LEFT,
            KeyEvent.KEYCODE_DPAD_RIGHT,
            KeyEvent.KEYCODE_BUTTON_SELECT,
            KeyEvent.KEYCODE_DPAD_CENTER,

            // These don't map into any SDL controller buttons directly
            KeyEvent.KEYCODE_BUTTON_L2,
            KeyEvent.KEYCODE_BUTTON_R2,
            KeyEvent.KEYCODE_BUTTON_C,
            KeyEvent.KEYCODE_BUTTON_Z,
            KeyEvent.KEYCODE_BUTTON_1,
            KeyEvent.KEYCODE_BUTTON_2,
            KeyEvent.KEYCODE_BUTTON_3,
            KeyEvent.KEYCODE_BUTTON_4,
            KeyEvent.KEYCODE_BUTTON_5,
            KeyEvent.KEYCODE_BUTTON_6,
            KeyEvent.KEYCODE_BUTTON_7,
            KeyEvent.KEYCODE_BUTTON_8,
            KeyEvent.KEYCODE_BUTTON_9,
            KeyEvent.KEYCODE_BUTTON_10,
            KeyEvent.KEYCODE_BUTTON_11,
            KeyEvent.KEYCODE_BUTTON_12,
            KeyEvent.KEYCODE_BUTTON_13,
            KeyEvent.KEYCODE_BUTTON_14,
            KeyEvent.KEYCODE_BUTTON_15,
            KeyEvent.KEYCODE_BUTTON_16,
        };
        int[] masks = new int[] {
            (1 << 0),   // A -> A
            (1 << 1),   // B -> B
            (1 << 2),   // X -> X
            (1 << 3),   // Y -> Y
            (1 << 4),   // BACK -> BACK
            (1 << 5),   // MODE -> GUIDE
            (1 << 6),   // START -> START
            (1 << 7),   // THUMBL -> LEFTSTICK
            (1 << 8),   // THUMBR -> RIGHTSTICK
            (1 << 9),   // L1 -> LEFTSHOULDER
            (1 << 10),  // R1 -> RIGHTSHOULDER
            (1 << 11),  // DPAD_UP -> DPAD_UP
            (1 << 12),  // DPAD_DOWN -> DPAD_DOWN
            (1 << 13),  // DPAD_LEFT -> DPAD_LEFT
            (1 << 14),  // DPAD_RIGHT -> DPAD_RIGHT
            (1 << 4),   // SELECT -> BACK
            (1 << 0),   // DPAD_CENTER -> A
            (1 << 15),  // L2 -> ??
            (1 << 16),  // R2 -> ??
            (1 << 17),  // C -> ??
            (1 << 18),  // Z -> ??
            (1 << 20),  // 1 -> ??
            (1 << 21),  // 2 -> ??
            (1 << 22),  // 3 -> ??
            (1 << 23),  // 4 -> ??
            (1 << 24),  // 5 -> ??
            (1 << 25),  // 6 -> ??
            (1 << 26),  // 7 -> ??
            (1 << 27),  // 8 -> ??
            (1 << 28),  // 9 -> ??
            (1 << 29),  // 10 -> ??
            (1 << 30),  // 11 -> ??
            (1 << 31),  // 12 -> ??
            // We're out of room...
            0xFFFFFFFF,  // 13 -> ??
            0xFFFFFFFF,  // 14 -> ??
            0xFFFFFFFF,  // 15 -> ??
            0xFFFFFFFF,  // 16 -> ??
        };
        boolean[] hasKeys = joystickDevice.hasKeys(keys);
        for (int i = 0; i < keys.length; ++i) {
            if (hasKeys[i]) {
                buttonMask |= masks[i];
            }
        }
        // https://github.com/libsdl-org/SDL/blob/47f2373dc13b66c48bf4024fcdab53cd0bdd59bb/src/joystick/android/SDL_sysjoystick.c#L360-L367
        if (nhats > 0) {
            // Add Dpad buttons.
            buttonMask |= 1 << 11;
            buttonMask |= 1 << 12;
            buttonMask |= 1 << 13;
            buttonMask |= 1 << 14;
        }
        return buttonMask;
    }

    private static int getAxisMask(InputDevice joystickDevice) {
        final int SDL_CONTROLLER_AXIS_LEFTX = 0;
        final int SDL_CONTROLLER_AXIS_LEFTY = 1;
        final int SDL_CONTROLLER_AXIS_RIGHTX = 2;
        final int SDL_CONTROLLER_AXIS_RIGHTY = 3;
        final int SDL_CONTROLLER_AXIS_TRIGGERLEFT = 4;
        final int SDL_CONTROLLER_AXIS_TRIGGERRIGHT = 5;

        int naxes = 0;
        boolean haveZ = false;
        boolean havePastZBeforeRZ = false;
        for (InputDevice.MotionRange range : joystickDevice.getMotionRanges()) {
            if ((range.getSource() & InputDevice.SOURCE_CLASS_JOYSTICK) != 0) {
                int axis = range.getAxis();
                if (axis != MotionEvent.AXIS_HAT_X && axis != MotionEvent.AXIS_HAT_Y) {
                    naxes++;
                }
                if (axis == MotionEvent.AXIS_Z) {
                  haveZ = true;
                } else if (axis > MotionEvent.AXIS_Z && axis < MotionEvent.AXIS_RZ) {
                  havePastZBeforeRZ = true;
                }
            }
        }
        // The variable is_accelerometer seems always false, then skip the checking:
        // https://github.com/libsdl-org/SDL/blob/0e9560aea22818884921e5e5064953257bfe7fa7/android-project/app/src/main/java/org/libsdl/app/SDLControllerManager.java#L207
        int axisMask = 0;
        if (naxes >= 2) {
            axisMask |= ((1 << SDL_CONTROLLER_AXIS_LEFTX) | (1 << SDL_CONTROLLER_AXIS_LEFTY));
        }
        if (naxes >= 4) {
            axisMask |= ((1 << SDL_CONTROLLER_AXIS_RIGHTX) | (1 << SDL_CONTROLLER_AXIS_RIGHTY));
        }
        if (naxes >= 6) {
            axisMask |= ((1 << SDL_CONTROLLER_AXIS_TRIGGERLEFT) | (1 << SDL_CONTROLLER_AXIS_TRIGGERRIGHT));
        }
        // Also add an indicator bit for whether the sorting order has changed.
        // This serves to disable outdated gamecontrollerdb.txt mappings.
        if (haveZ && havePastZBeforeRZ) {
            axisMask |= 0x8000;
        }
        return axisMask;
    }

    @Override
    public void onInputDeviceChanged(int deviceId) {
        // Do nothing.
    }

    @Override
    public void onInputDeviceRemoved(int deviceId) {
        // Do not call inputManager.getInputDevice(), which returns null (#1185).
        Ebitenmobileview.onInputDeviceRemoved(deviceId);
        this.gamepads.remove(this.getGamepad(deviceId));
    }

    // suspendGame suspends the game.
    // It is recommended to call this when the application is being suspended e.g.,
    // Activity's onPause is called.
    public void suspendGame() {
        this.inputManager.unregisterInputDeviceListener(this);
        this.ebitenSurfaceView.onPause();
        try {
            Ebitenmobileview.suspend();
        } catch (final Exception e) {
            onErrorOnGameUpdate(e);
        }
    }

    // resumeGame resumes the game.
    // It is recommended to call this when the application is being resumed e.g.,
    // Activity's onResume is called.
    public void resumeGame() {
        this.inputManager.registerInputDeviceListener(this, null);
        this.ebitenSurfaceView.onResume();
        try {
            Ebitenmobileview.resume();
        } catch (final Exception e) {
            onErrorOnGameUpdate(e);
        }
    }

    // onErrorOnGameUpdate is called on the main thread when an error happens when updating a game.
    // You can define your own error handler, e.g., using Crashlytics, by overriding this method.
    protected void onErrorOnGameUpdate(Exception e) {
        Log.e("Go", e.toString());
    }

    // saveGPUResources starts to save GPU resources.
    // saveGPUResources is non-blocking.
    // You can check whether the GPU resources are being saved by areGPUResourcesSaved.
    //
    // Even without saveGPUResources, the GPU resources can be reclaimed by Android OS,
    // but this is not 100% guaranteed.
    // If saveGPUResources is called, Ebitengine saves GPU resources and restores them
    // in the case they are lost.
    //
    // It is recommended to call saveGPUResources to true only when necessary e.g. showing a reward ad.
    //
    // After GPU resources are saved, rendering is suspended until resumeGame is called or
    // GLSurfaceView is recreated by an actual context loss.
    public void saveGPUResources() {
        Ebitenmobileview.saveGPUResources();
    }

    // areGPUResourcesSaved reports whether GPU resources are being saved.
    public boolean areGPUResourcesSaved() {
        return Ebitenmobileview.areGPUResourcesSaved();
    }

    // EbitenEditText is the hidden edit text serving the IME. Its content is
    // seeded and observed by the Go side; the game renders the text itself.
    private class EbitenEditText extends EditText {
        public EbitenEditText(Context context) {
            super(context);
            setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_FLAG_MULTI_LINE);
            setImeOptions(EditorInfo.IME_FLAG_NO_EXTRACT_UI | EditorInfo.IME_FLAG_NO_FULLSCREEN);
            setFocusableInTouchMode(true);
            // The edit text sits at the game's caret but must not be seen. An
            // invisible or offscreen view could not keep the focus or anchor
            // panning, so make it fully transparent instead.
            setAlpha(0.0f);
            if (Build.VERSION.SDK_INT >= 26) {
                setImportantForAutofill(IMPORTANT_FOR_AUTOFILL_NO);
            }
            addTextChangedListener(new TextWatcher() {
                @Override
                public void beforeTextChanged(CharSequence s, int start, int count, int after) {
                }

                @Override
                public void onTextChanged(CharSequence s, int start, int before, int count) {
                }

                @Override
                public void afterTextChanged(Editable s) {
                    reportTextInputState();
                }
            });
        }

        @Override
        protected void onSelectionChanged(int selStart, int selEnd) {
            super.onSelectionChanged(selStart, selEnd);
            reportTextInputState();
        }

        // Composition operations like finishComposingText and
        // setComposingRegion can change only the composing spans, firing
        // neither the TextWatcher nor onSelectionChanged. Wrap the input
        // connection to report the state after such an operation completes.
        @Override
        public InputConnection onCreateInputConnection(EditorInfo outAttrs) {
            InputConnection connection = super.onCreateInputConnection(outAttrs);
            if (connection == null) {
                return null;
            }
            return new InputConnectionWrapper(connection, false) {
                // An input method commits the Return key as a line break,
                // appended to the text it finalizes. The game decides what a
                // Return does, so commit only that text and report the key.
                @Override
                public boolean commitText(CharSequence text, int newCursorPosition) {
                    if (endsWithLineBreak(text)) {
                        return commitBeforeReturnKey(text, newCursorPosition, null);
                    }
                    return super.commitText(text, newCursorPosition);
                }

                // The TextAttribute overload (API 33) delegates directly to
                // the wrapped connection, bypassing the two-argument override.
                // The method is never called on older systems, where merely
                // declaring it is harmless.
                @Override
                public boolean commitText(CharSequence text, int newCursorPosition, TextAttribute textAttribute) {
                    if (endsWithLineBreak(text)) {
                        return commitBeforeReturnKey(text, newCursorPosition, textAttribute);
                    }
                    return super.commitText(text, newCursorPosition, textAttribute);
                }

                private boolean commitBeforeReturnKey(CharSequence text, int newCursorPosition, TextAttribute textAttribute) {
                    CharSequence committed = text.subSequence(0, text.length() - 1);
                    boolean result;
                    // The state this commit reports carries the key, so that
                    // the game acts on the Return as well.
                    passthroughKeyPending = true;
                    try {
                        if (textAttribute == null) {
                            result = super.commitText(committed, newCursorPosition);
                        } else {
                            result = super.commitText(committed, newCursorPosition, textAttribute);
                        }
                    } finally {
                        passthroughKeyPending = false;
                    }
                    dispatchKeyToGame(KeyEvent.KEYCODE_ENTER);
                    return result;
                }

                @Override
                public boolean finishComposingText() {
                    boolean result = super.finishComposingText();
                    reportTextInputState();
                    return result;
                }

                @Override
                public boolean setComposingRegion(int start, int end) {
                    boolean result = super.setComposingRegion(start, end);
                    reportTextInputState();
                    return result;
                }

                // The TextAttribute overload (API 33) delegates directly to
                // the wrapped connection, bypassing the two-argument override.
                // The method is never called on older systems, where merely
                // declaring it is harmless.
                @Override
                public boolean setComposingRegion(int start, int end, TextAttribute textAttribute) {
                    boolean result = super.setComposingRegion(start, end, textAttribute);
                    reportTextInputState();
                    return result;
                }

                @Override
                public boolean endBatchEdit() {
                    boolean result = super.endBatchEdit();
                    if (!result) {
                        // The last nested batch edit ended.
                        reportTextInputState();
                    }
                    return result;
                }
            };
        }

        // The game handles the touches at the caret; never take them.
        @Override
        public boolean onTouchEvent(MotionEvent event) {
            return false;
        }

        // The focused edit text receives the hardware key events instead of
        // EbitenView; forward them to the game like EbitenView's onKeyDown and
        // onKeyUp do. The Return key, which an input method delivers as a key
        // event too, is consumed after forwarding: the game acts on the key
        // press, so a line break in the text buffer would double it.
        @Override
        public boolean onKeyDown(int keyCode, KeyEvent event) {
            if (event.getRepeatCount() == 0) {
                Ebitenmobileview.onKeyDownOnAndroid(keyCode, event.getUnicodeChar(), event.getSource(), event.getDeviceId(), event.getMetaState());
            }
            if (keyCode == KeyEvent.KEYCODE_ENTER) {
                return true;
            }
            return super.onKeyDown(keyCode, event);
        }

        @Override
        public boolean onKeyUp(int keyCode, KeyEvent event) {
            Ebitenmobileview.onKeyUpOnAndroid(keyCode, event.getSource(), event.getDeviceId(), event.getMetaState());
            if (keyCode == KeyEvent.KEYCODE_ENTER) {
                return true;
            }
            return super.onKeyUp(keyCode, event);
        }

        // endsWithLineBreak reports whether text ends with the line break an
        // input method appends for the Return key.
        private boolean endsWithLineBreak(CharSequence text) {
            return text != null && text.length() > 0 && text.charAt(text.length() - 1) == '\n';
        }

        // dispatchKeyToGame reports a key press, and its release, to the game.
        // The input method delivered the key as an edit, so no key event
        // exists to forward; the virtual keyboard's device is its source.
        private void dispatchKeyToGame(int keyCode) {
            Ebitenmobileview.onKeyDownOnAndroid(keyCode, 0, 0, KeyCharacterMap.VIRTUAL_KEYBOARD, 0);
            Ebitenmobileview.onKeyUpOnAndroid(keyCode, 0, KeyCharacterMap.VIRTUAL_KEYBOARD, 0);
        }
    }

    // startTextInput implements TextInputDriver. It is called from the Go side
    // on an arbitrary thread.
    @Override
    public void startTextInput(final String text, final long selectionStartInUTF16, final long selectionEndInUTF16, final long caretX, final long caretY, final long caretWidth, final long caretHeight, final long generation) {
        new Handler(Looper.getMainLooper()).post(new Runnable() {
            @Override
            public void run() {
                textInputGeneration = generation;
                EbitenView.this.caretX = (int)caretX;
                EbitenView.this.caretY = (int)caretY;
                EbitenView.this.caretWidth = (int)caretWidth;
                EbitenView.this.caretHeight = (int)caretHeight;
                layoutEditText();
                // Programmatic seeding is not the user's input; do not report
                // it. setText invokes the TextWatcher synchronously.
                suppressTextInputReports = true;
                editText.setText(text);
                editText.setSelection((int)selectionStartInUTF16, (int)selectionEndInUTF16);
                suppressTextInputReports = false;
                editText.requestFocus();
                showSoftInput();
            }
        });
    }

    // showSoftInput shows the virtual keyboard, or defers showing it until the
    // window gains the focus, where showSoftInput is ignored.
    private void showSoftInput() {
        if (!hasWindowFocus()) {
            this.pendingShowSoftInput = true;
            return;
        }
        this.pendingShowSoftInput = false;
        InputMethodManager imm = (InputMethodManager)getContext().getSystemService(Context.INPUT_METHOD_SERVICE);
        imm.showSoftInput(this.editText, 0);
    }

    @Override
    public void onWindowFocusChanged(boolean hasWindowFocus) {
        super.onWindowFocusChanged(hasWindowFocus);
        if (hasWindowFocus && this.pendingShowSoftInput) {
            showSoftInput();
        }
    }

    // dismissTextInput implements TextInputDriver. It is called from the Go
    // side on an arbitrary thread.
    @Override
    public void dismissTextInput() {
        new Handler(Looper.getMainLooper()).post(new Runnable() {
            @Override
            public void run() {
                pendingShowSoftInput = false;
                InputMethodManager imm = (InputMethodManager)getContext().getSystemService(Context.INPUT_METHOD_SERVICE);
                imm.hideSoftInputFromWindow(getWindowToken(), 0);
                // Programmatic clearing is not the user's input; do not report
                // it.
                suppressTextInputReports = true;
                editText.setText("");
                suppressTextInputReports = false;
                editText.clearFocus();
            }
        });
    }

    private void reportTextInputState() {
        if (this.editText == null || this.suppressTextInputReports) {
            return;
        }
        Editable e = this.editText.getText();
        int composingStart = BaseInputConnection.getComposingSpanStart(e);
        int composingEnd = BaseInputConnection.getComposingSpanEnd(e);
        Ebitenmobileview.onTextInputStateChanged(e.toString(), this.editText.getSelectionStart(), this.editText.getSelectionEnd(), composingStart, composingEnd, this.passthroughKeyPending, this.textInputGeneration);
    }

    private void reportVirtualKeyboardState() {
        // Measure the visible frame rather than assuming a soft-input mode, so
        // that any adjust mode reports a true region.
        Rect visibleFrame = new Rect();
        getWindowVisibleDisplayFrame(visibleFrame);
        int[] location = new int[2];
        getLocationOnScreen(location);
        int x0 = Math.min(Math.max(visibleFrame.left - location[0], 0), getWidth());
        int y0 = Math.min(Math.max(visibleFrame.top - location[1], 0), getHeight());
        int x1 = Math.min(Math.max(visibleFrame.right - location[0], x0), getWidth());
        int y1 = Math.min(Math.max(visibleFrame.bottom - location[1], y0), getHeight());
        boolean shown;
        if (Build.VERSION.SDK_INT >= 30) {
            WindowInsets insets = getRootWindowInsets();
            shown = insets != null && insets.isVisible(WindowInsets.Type.ime());
            if (shown) {
                // The visible frame does not reflect the virtual keyboard with
                // some soft-input modes (e.g. adjustNothing). Clip the region
                // by the IME inset, which is measured from the window's bottom
                // edge.
                Insets imeInsets = insets.getInsets(WindowInsets.Type.ime());
                View root = getRootView();
                int[] rootLocation = new int[2];
                root.getLocationOnScreen(rootLocation);
                int imeTop = rootLocation[1] + root.getHeight() - imeInsets.bottom - location[1];
                y1 = Math.min(Math.max(imeTop, y0), y1);
            }
        } else {
            // This view can already be resized to the visible frame (e.g. with
            // adjustResize), so measure against the root view, which keeps the
            // window's full size. Assume that a band taller than any system bar
            // at the bottom of the window is the virtual keyboard.
            View root = getRootView();
            int[] rootLocation = new int[2];
            root.getLocationOnScreen(rootLocation);
            int rootBottom = rootLocation[1] + root.getHeight();
            shown = rootBottom - visibleFrame.bottom > 100 * getResources().getDisplayMetrics().density;
        }
        Ebitenmobileview.onVirtualKeyboardChanged(shown, x0, y0, x1 - x0, y1 - y0);
    }

    private static boolean seqContextSet;

    private float deviceScale = 1.0f;
    private EbitenSurfaceView ebitenSurfaceView;
    private EbitenEditText editText;
    private InputManager inputManager;
    private ArrayList<Gamepad> gamepads;
    private long textInputGeneration;
    private int caretX;
    private int caretY;
    private int caretWidth;
    private int caretHeight;
    private boolean pendingShowSoftInput;
    private boolean suppressTextInputReports;
    private boolean passthroughKeyPending;
}
