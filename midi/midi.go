package midi

import (
	"sync"

	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/reader"
)

type MidiContext struct {
	driver midi.Driver
}

func NewMidiContext(driver midi.Driver) *MidiContext {
	return &MidiContext{
		driver: driver,
	}
}

func (m *MidiContext) InputPorts() ([]*MidiInput, error) {
	ports := []*MidiInput{}
	ins, err := m.driver.Ins()
	if err != nil {
		return nil, err
	}

	for _, in := range ins {
		ports = append(ports, newInput(in))
	}

	return ports, nil
}

func newInput(in midi.In) *MidiInput {
	return &MidiInput{
		in:           in,
		notesPressed: map[uint8]bool{},
		pressedMu:    sync.RWMutex{},
	}
}

type MidiInput struct {
	in           midi.In
	notesPressed map[uint8]bool
	pressedMu    sync.RWMutex
}

func (i *MidiInput) Read() error {
	if !i.in.IsOpen() {
		if err := i.in.Open(); err != nil {
			return err
		}
	}

	r := reader.New(
		//reader.NoLogger(),
		reader.NoteOn(i.handlePress),
		reader.NoteOff(i.handleRelease),
	)

	return r.ListenTo(i.in)
}

func (i *MidiInput) Close() error {
	if i.in.IsOpen() {
		return i.in.Close()
	}

	return nil
}

func (i *MidiInput) String() string {
	return i.in.String()
}

func (i *MidiInput) Number() int {
	return i.in.Number()
}

func (i *MidiInput) handlePress(p *reader.Position, channel, key, velocity uint8) {
	i.pressedMu.Lock()
	i.notesPressed[key] = true
	i.pressedMu.Unlock()
}

func (i *MidiInput) handleRelease(p *reader.Position, channel, key, velocity uint8) {
	i.pressedMu.Lock()
	i.notesPressed[key] = false
	i.pressedMu.Unlock()
}

func (i *MidiInput) GetMidiKey(k uint8) bool {
	if !i.in.IsOpen() {
		return false
	}

	i.pressedMu.RLock()
	isPressed, found := i.notesPressed[k]
	i.pressedMu.RUnlock()

	return found && isPressed
}
