//go:build !android && !nintendosdk && !playstation5

package gamepad

import (
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	_UI_SET_EVBIT   = 0x40045564
	_UI_SET_KEYBIT  = 0x40045565
	_UI_SET_ABSBIT  = 0x40045567
	_UI_SET_FFBIT   = 0x4004556b
	_UI_DEV_CREATE  = 0x5501
	_UI_DEV_DESTROY = 0x5502

	_UI_BEGIN_FF_UPLOAD = 0xc03855c8
	_UI_END_FF_UPLOAD   = 0x403855c9

	_UI_FF_UPLOAD = 1
)

type uinputUserDev struct {
	name         [80]byte
	id           input_id
	ffEffectsMax uint32
	absmax       [_ABS_CNT]int32
	absmin       [_ABS_CNT]int32
	absfuzz      [_ABS_CNT]int32
	absflat      [_ABS_CNT]int32
}

type uinputFFUpload struct {
	requestID uint32
	retval    int32
	effect    ffEffect
}

func TestFFEffectLayout(t *testing.T) {
	want := uintptr(48)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		want = 44
	}
	if got := unsafe.Sizeof(ffEffect{}); got != want {
		t.Errorf("sizeof(ffEffect) = %d, want %d to match struct ff_effect", got, want)
	}
	if got := unsafe.Offsetof(ffEffect{}.u); got != 16 {
		t.Errorf("offsetof(ffEffect.u) = %d, want 16", got)
	}
	if got := unsafe.Sizeof(uinputFFUpload{}); got != 8+want {
		t.Errorf("sizeof(uinputFFUpload) = %d, want %d", got, 8+want)
	}
}

func TestLinuxUinputRumble(t *testing.T) {
	ufd, err := unix.Open("/dev/uinput", unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Skipf("cannot open /dev/uinput: %v", err)
	}
	defer func() { _ = unix.Close(ufd) }()

	setBit := func(req uint, bit uint) {
		t.Helper()
		r, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(ufd), uintptr(req), uintptr(bit))
		if int32(r) < 0 {
			t.Fatalf("ioctl %#x(%#x) failed: %v", req, bit, e)
		}
	}
	setBit(_UI_SET_EVBIT, unix.EV_KEY)
	setBit(_UI_SET_EVBIT, unix.EV_ABS)
	setBit(_UI_SET_EVBIT, unix.EV_FF)
	setBit(_UI_SET_KEYBIT, _BTN_GAMEPAD)
	setBit(_UI_SET_ABSBIT, _ABS_X)
	setBit(_UI_SET_ABSBIT, _ABS_Y)
	setBit(_UI_SET_FFBIT, _FF_RUMBLE)

	setup := uinputUserDev{
		ffEffectsMax: 4,
		id:           input_id{bustype: 0x03 /* BUS_USB */, vendor: 0x1234, product: 0x5678, version: 1},
	}
	copy(setup.name[:], "Ebitengine Rumble Test Pad")
	setup.absmin[_ABS_X] = -32767
	setup.absmax[_ABS_X] = 32767
	setup.absmin[_ABS_Y] = -32767
	setup.absmax[_ABS_Y] = 32767
	buf := (*[unsafe.Sizeof(setup)]byte)(unsafe.Pointer(&setup))[:]
	if _, err := unix.Write(ufd, buf); err != nil {
		t.Fatalf("writing uinput_user_dev failed: %v", err)
	}
	r, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(ufd), uintptr(_UI_DEV_CREATE), 0)
	if int32(r) < 0 {
		t.Fatalf("UI_DEV_CREATE failed: %v", e)
	}
	t.Cleanup(func() {
		unix.Syscall(unix.SYS_IOCTL, uintptr(ufd), uintptr(_UI_DEV_DESTROY), 0)
	})

	// Stale device nodes can be reused by a newly created device, so compare inodes
	// rather than directory entries.
	type nodeInfo struct{ ino, dev uint64 }
	before := map[string]nodeInfo{}
	if ents, err := os.ReadDir(dirName); err == nil {
		for _, ent := range ents {
			if !ent.IsDir() && reEvent.MatchString(ent.Name()) {
				if st, err := ent.Info(); err == nil {
					s := st.Sys().(*syscall.Stat_t)
					before[ent.Name()] = nodeInfo{ino: s.Ino, dev: s.Dev}
				}
			}
		}
	}
	var node string
	for i := 0; i < 50 && node == ""; i++ {
		time.Sleep(200 * time.Millisecond)
		ents, err := os.ReadDir(dirName)
		if err != nil {
			continue
		}
		for _, ent := range ents {
			name := ent.Name()
			if !reEvent.MatchString(name) || ent.IsDir() {
				continue
			}
			st, err := ent.Info()
			if err != nil {
				continue
			}
			s := st.Sys().(*syscall.Stat_t)
			b, ok := before[name]
			if ok && b.ino == s.Ino && b.dev == s.Dev {
				continue
			}
			fd, err := unix.Open(dirName+"/"+name, unix.O_RDWR|unix.O_NONBLOCK, 0)
			if err != nil {
				if err == unix.EACCES || err == unix.EPERM {
					t.Skipf("no permission for the created event node %s: %v", name, err)
				}
				continue // A stale node: ENODEV or similar.
			}
			node = dirName + "/" + name
			unix.Close(fd)
			break
		}
	}
	if node == "" {
		t.Skip("the created input event node was not found")
	}

	probe, err := unix.Open(node, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Skipf("no permission for the created event node %s: %v", node, err)
	}

	g := &gamepads{}
	np := &nativeGamepadsImpl{}
	if err := np.openGamepad(g, node); err != nil {
		_ = unix.Close(probe)
		t.Fatalf("openGamepad failed: %v", err)
	}
	gp := g.find(func(*Gamepad) bool { return true })
	if gp == nil {
		_ = unix.Close(probe)
		t.Fatal("gamepad not found after openGamepad")
	}
	native := gp.native.(*nativeGamepadImpl)
	if !native.hasVibration {
		_ = unix.Close(probe)
		t.Error("hasVibration is false")
		return
	}
	if native.effectID != -1 {
		_ = unix.Close(probe)
		t.Errorf("effectID should be initialized to -1, got %d", native.effectID)
		return
	}

	stop := make(chan struct{})
	drained := make(chan int16, 16)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			var ev input_event
			b := (*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev))[:]
			if _, err := unix.Read(ufd, b); err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			if ev.typ == _UI_FF_UPLOAD {
				req := uinputFFUpload{requestID: uint32(ev.code)}
				if err := ioctl(ufd, _UI_BEGIN_FF_UPLOAD, unsafe.Pointer(&req)); err != nil {
					t.Errorf("UI_BEGIN_FF_UPLOAD failed: %v", err)
					return
				}
				select {
				case drained <- req.effect.id:
				default:
				}
				if err := ioctl(ufd, _UI_END_FF_UPLOAD, unsafe.Pointer(&req)); err != nil {
					t.Errorf("UI_END_FF_UPLOAD failed: %v", err)
					return
				}
			}
		}
	}()

	readEvent := func(typ uint16, value int32, timeout time.Duration) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			var ev input_event
			b := (*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev))[:]
			n, err := unix.Read(probe, b)
			if err != nil || n < len(b) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			if ev.typ == typ && ev.value == value {
				return true
			}
		}
		return false
	}

	gp.Vibrate(time.Second, 0.75, 0.25)

	select {
	case id := <-drained:
		if id < 0 {
			t.Errorf("uploaded effect ID should be non-negative, got %d", id)
		}
	case <-time.After(5 * time.Second):
		close(stop)
		_ = unix.Close(probe)
		t.Fatal("no FF upload was requested")
	}

	if !readEvent(unix.EV_FF_STATUS, 1, 5*time.Second) {
		t.Error("FF_STATUS ON was not observed")
	}

	gp.Vibrate(0, 0, 0)

	if !readEvent(unix.EV_FF_STATUS, 0, 5*time.Second) {
		t.Error("FF_STATUS OFF was not observed")
	}

	close(stop)
	_ = unix.Close(probe)
}
