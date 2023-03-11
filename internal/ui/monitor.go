package ui

type Monitor struct {
	x, y  int
	w, h  int
	index int
	rate  int
	name  string
}

func (m *Monitor) Position() (x, y int) {
	return m.x, m.y
}

func (m *Monitor) Size() (w, h int) {
	return m.w, m.h
}

func (m *Monitor) Name() string {
	return m.name
}

func (m *Monitor) Index() int {
	return m.index
}

func (m *Monitor) RefreshRate() int {
	return m.rate
}
