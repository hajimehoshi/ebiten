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
	if t.state.allocated {
		pthread_key_delete(t.state.key)
	}
	*t = tls{}
	return nil
}

func (t *tls) get() (uintptr, error) {
	if !t.state.allocated {
		panic("glfwwin: TLS must be allocated")
	}
	return pthread_getspecific(t.state.key), nil
}

func (t *tls) set(value uintptr) error {
	panic("NOT IMPLEMENTED")
	return nil
}
