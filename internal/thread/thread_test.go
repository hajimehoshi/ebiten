package thread

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
)

func Test0ParamsReturnError(t *testing.T) {
	const errorString = "foobar"
	var err error
	err = thread.Call(func() error {
		return errors.New(errorString)
	})

	if err.Error() != errorString {
		t.Error("Error string was wrong")
	}

	err = thread.Call(func() error {
		return nil
	})

	if err != nil {
		t.Error("Error should have been nil")
	}
}

func Test1ParamReturnsBool(t *testing.T) {
	var result bool
	f := func(param uintptr) bool {
		return param > 10
	}

	result = thread.BoolCall1(11, f)
	if !result {
		t.Error("Should return true with param above 10")
	}

	result = thread.BoolCall1(10, f)
	if result {
		t.Error("Should return false with param not above 10")
	}
}

func Test2ParamsReturnsBool(t *testing.T) {
	var result bool
	f := func(param1, param2 uintptr) bool {
		return param1 > 10 && param2 > 10
	}

	result = thread.BoolCall2(11, 11, f)
	if !result {
		t.Error("Should return true with both params above 10")
	}

	result = thread.BoolCall2(10, 11, f)
	if result {
		t.Error("Should return false with param1 not above 10")
	}

	result = thread.BoolCall2(11, 10, f)
	if result {
		t.Error("Should return false with param2 not above 10")
	}
}

func TestReturns2(t *testing.T) {
	var r1, r2 uintptr

	r1, r2 = thread.CallReturn2(func() (uintptr, uintptr) {
		return 100, 100
	})
	if r1 != 100 || r2 != 100 {
		t.Errorf("Expected 100, 100, got %d, %d", r1, r2)
	}

	r1, r2 = thread.CallReturn2(func() (uintptr, uintptr) {
		return 250, 150
	})
	if r1 != 250 || r2 != 150 {
		t.Errorf("Expected 250, 150, got %d, %d", r1, r2)
	}
}

var thread *Thread

func TestMain(m *testing.M) {
	// Start the thread
	ctx, cancel := context.WithCancel(context.Background())
	thread = New()

	var wg sync.WaitGroup
	wg.Add(1)
	go runThread(ctx, &wg)

	result := m.Run()

	cancel()
	wg.Wait()

	os.Exit(result)
}

func runThread(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	thread.Loop(ctx)
}
