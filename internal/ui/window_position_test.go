package ui

import "testing"

// TestWindowPositionBeforeRun ensures Window().Position does not panic before the
// main loop starts. SetPosition stores the initial position when the backend is not
// yet published, and Position must return it (consistently with Size) instead of
// panicking.
func TestWindowPositionBeforeRun(t *testing.T) {
	w := Get().Window()
	// SetPosition before the main loop stores the init position.
	w.SetPosition(10, 20)

	var x, y int
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Window().Position() panicked before the main loop: %v", r)
			}
		}()
		x, y = w.Position()
	}()
	if x != 10 || y != 20 {
		t.Errorf("Window().Position() = (%d, %d), want (10, 20)", x, y)
	}
}
