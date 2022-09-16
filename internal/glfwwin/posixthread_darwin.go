package glfwwin

import "fmt"

func (t *tls) create() error {
	if t.state.allocated {
		panic("glfwwin: TLS must not be allocated")
	}
	if pthread_key_create(&t.state.key, 0) != 0 {
		return fmt.Errorf("posix: failed to create context TLS")
	}
	t.state.allocated = true
	return nil
}

func (t *tls) destroy() error {
	panic("NOT IMPLEMENTED")
	return nil
}

func (t *tls) get() (uintptr, error) {
	panic("NOT IMPLEMENTED")
	return 0, nil
}

func (t *tls) set(value uintptr) error {
	panic("NOT IMPLEMENTED")
	return nil
}
