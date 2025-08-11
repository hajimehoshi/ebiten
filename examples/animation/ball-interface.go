package main

type BallScene interface {
    GetT() float64
    SetT(float64)
    GetTDir() float64
    SetTDir(float64)
    GetCount() int
}

func (s *Scene1) GetT() float64 { return s.t }
func (s *Scene1) SetT(val float64) { s.t = val }

func (s *Scene1) GetTDir() float64 { return s.tDir }
func (s *Scene1) SetTDir(val float64) { s.tDir = val }

func (s *Scene1) GetCount() int { return s.count }

func (s *Scene4) GetT() float64 { return s.t }
func (s *Scene4) SetT(val float64) { s.t = val }

func (s *Scene4) GetTDir() float64 { return s.tDir }
func (s *Scene4) SetTDir(val float64) { s.tDir = val }

func (s *Scene4) GetCount() int { return s.count }
